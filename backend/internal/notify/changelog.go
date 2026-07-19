package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"container-updater/backend/internal/config"
	"container-updater/backend/internal/logger"
)

type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
}

// ExtractGithubSource checks standard labels for a GitHub URL
func ExtractGithubSource(labels map[string]string) string {
	keys := []string{
		"org.opencontainers.image.source",
		"org.label-schema.vcs-url",
	}

	for _, k := range keys {
		if val, ok := labels[k]; ok && strings.Contains(val, "github.com") {
			return val
		}
	}

	// Dynamic fallback: look for any label containing a github.com link
	for _, val := range labels {
		if strings.Contains(val, "github.com") {
			return val
		}
	}

	return ""
}

// FetchLatestReleaseNotes queries the GitHub API for the latest release of a repository
func FetchLatestReleaseNotes(ctx context.Context, githubURL string) (string, error) {
	owner, repo := parseGithubURL(githubURL)
	if owner == "" || repo == "" {
		return "", fmt.Errorf("invalid github URL: %s", githubURL)
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "container-updater")

	// Set GitHub Token if configured to avoid rate-limiting
	if token := config.GlobalConfig.GitHubToken; token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("no release found for repository %s/%s", owner, repo)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned error status: %s", resp.Status)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	// Format release notes snippet (truncate if too long)
	body := release.Body
	if len(body) > 1000 {
		body = body[:1000] + "... (truncated)"
	}

	changelog := fmt.Sprintf("### Release [%s](%s) (%s)\n\n%s", 
		release.TagName, release.HTMLURL, release.PublishedAt.Format("2006-01-02"), body)

	return changelog, nil
}

// parseGithubURL extracts owner and repo names from a URL like https://github.com/owner/repo[.git]
func parseGithubURL(url string) (string, string) {
	// Clean up scheme and prefix
	cleaned := strings.TrimSuffix(url, "/")
	cleaned = strings.TrimSuffix(cleaned, ".git")

	// Regexp to capture owner and repo
	re := regexp.MustCompile(`github\.com/([^/]+)/([^/]+)`)
	matches := re.FindStringSubmatch(cleaned)
	if len(matches) == 3 {
		return matches[1], matches[2]
	}

	return "", ""
}
