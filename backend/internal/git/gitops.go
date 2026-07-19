package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"container-updater/backend/internal/config"
	"container-updater/backend/internal/logger"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	cryptossh "golang.org/x/crypto/ssh"
)

// ProcessGitOpsCommit orchestrates cloning/pulling the repository, staging changes, committing, and pushing.
func ProcessGitOpsCommit(ctx context.Context, workloadName, newImage string) error {
	if !config.GlobalConfig.GitOps.Enabled {
		return nil
	}

	gitConf := config.GlobalConfig.GitOps
	if gitConf.RepoURL == "" {
		return fmt.Errorf("GitOps enabled but GITOPS_REPO_URL is not configured")
	}

	cloneDir := os.Getenv("GITOPS_CLONE_DIR")
	if cloneDir == "" {
		cloneDir = "/app/gitops"
	}

	logger.Log.Info("initiating GitOps transaction...", "repo", gitConf.RepoURL, "branch", gitConf.Branch, "clone_dir", cloneDir)

	// 1. Build Authentication
	auth, err := getGitAuth()
	if err != nil {
		return fmt.Errorf("failed to configure Git authentication: %w", err)
	}

	var repo *git.Repository

	// 2. Clone or Pull changes
	if _, err := os.Stat(filepath.Join(cloneDir, ".git")); os.IsNotExist(err) {
		// Clean dir if it exists but is not a git repo
		os.RemoveAll(cloneDir)
		os.MkdirAll(cloneDir, 0755)

		logger.Log.Info("cloning remote GitOps repository...", "url", gitConf.RepoURL)
		repo, err = git.PlainCloneContext(ctx, cloneDir, false, &git.CloneOptions{
			URL:           gitConf.RepoURL,
			ReferenceName: plumbing.NewBranchReferenceName("refs/heads/" + gitConf.Branch),
			Auth:          auth,
			SingleBranch:  true,
		})
		if err != nil {
			return fmt.Errorf("git clone failed: %w", err)
		}
	} else {
		logger.Log.Info("opening existing local GitOps repository...")
		repo, err = git.PlainOpen(cloneDir)
		if err != nil {
			return fmt.Errorf("failed to open local git repo: %w", err)
		}

		w, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("failed to get worktree: %w", err)
		}

		logger.Log.Info("pulling latest changes from remote GitOps repository...")
		err = w.PullContext(ctx, &git.PullOptions{
			ReferenceName: plumbing.NewBranchReferenceName("refs/heads/" + gitConf.Branch),
			Auth:          auth,
			Force:         true,
		})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return fmt.Errorf("git pull failed: %w", err)
		}
	}

	// 3. Stage manifest changes
	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to retrieve git worktree for staging: %w", err)
	}

	status, err := w.Status()
	if err != nil {
		return fmt.Errorf("failed to fetch git status: %w", err)
	}

	if status.IsClean() {
		logger.Log.Info("no changes detected in git worktree, skipping commit")
		return nil
	}

	logger.Log.Info("staging changes...")
	_, err = w.Add(".")
	if err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// 4. Commit changes
	commitMsg := fmt.Sprintf("chore: update workload %s to image %s", workloadName, newImage)
	logger.Log.Info("committing changes...", "msg", commitMsg)
	
	_, err = w.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Container Updater",
			Email: "updater@container-updater.local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	// 5. Push to Remote Origin
	logger.Log.Info("pushing commits to remote origin...")
	err = repo.PushContext(ctx, &git.PushOptions{
		Auth: auth,
	})
	if err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}

	logger.Log.Info("GitOps transaction completed successfully.")
	return nil
}

func getGitAuth() (transport.AuthMethod, error) {
	gitConf := config.GlobalConfig.GitOps

	// 1. SSH Keys auth fallback
	if gitConf.SSHKeyPath != "" {
		logger.Log.Debug("using SSH Private Key authentication", "key_path", gitConf.SSHKeyPath)
		publicKeys, err := ssh.NewPublicKeysFromFile("git", gitConf.SSHKeyPath, "")
		if err != nil {
			return nil, fmt.Errorf("failed to parse SSH private key file: %w", err)
		}
		
		// Bypass host key verification in sandbox environments or local runs
		publicKeys.HostKeyCallback = cryptossh.InsecureIgnoreHostKey()
		return publicKeys, nil
	}

	// 2. HTTP Basic auth
	if gitConf.Username != "" || gitConf.Password != "" {
		logger.Log.Debug("using HTTP Basic authentication", "username", gitConf.Username)
		return &http.BasicAuth{
			Username: gitConf.Username,
			Password: gitConf.Password,
		}, nil
	}

	// 3. No auth (anonymous or agent-based)
	logger.Log.Debug("using default anonymous Git authentication")
	return nil, nil
}
