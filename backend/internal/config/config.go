package config

import (
	"os"
)

type Config struct {
	DatabasePath  string
	Port          string
	LogLevel      string
	CronSchedule  string // e.g. "0 * * * *" (hourly)
	GitHubToken   string // for API changelogs
	FrontendURL   string // optional explicit frontend redirect URL
	OIDC          OIDCConfig
	GitOps        GitOpsConfig
}

type OIDCConfig struct {
	Enabled      bool
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	CookieSecret string // secret key to sign session cookies
}

type GitOpsConfig struct {
	Enabled      bool
	RepoURL      string
	Branch       string
	SSHKeyPath   string
	Username     string
	Password     string // personal access token or password
}

var GlobalConfig *Config

func Load() {
	c := &Config{
		DatabasePath: getEnv("DATABASE_PATH", "/app/data/db.sqlite"),
		Port:         getEnv("PORT", "8080"),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		CronSchedule: getEnv("CRON_SCHEDULE", "*/15 * * * *"), // Default check: every 15 minutes
		GitHubToken:  os.Getenv("GITHUB_TOKEN"),
		FrontendURL:  getEnv("FRONTEND_URL", os.Getenv("OIDC_FRONTEND_URL")),
	}

	oidcIssuer := os.Getenv("OIDC_ISSUER")
	if oidcIssuer == "" {
		oidcIssuer = os.Getenv("OIDC_PROVIDER_URL")
	}

	c.OIDC = OIDCConfig{
		Enabled:      os.Getenv("OIDC_ENABLED") == "true",
		Issuer:       oidcIssuer,
		ClientID:     os.Getenv("OIDC_CLIENT_ID"),
		ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("OIDC_REDIRECT_URL"),
		CookieSecret: getEnv("OIDC_COOKIE_SECRET", "super-secret-key-change-me"),
	}

	c.GitOps = GitOpsConfig{
		Enabled:    os.Getenv("GITOPS_ENABLED") == "true",
		RepoURL:    os.Getenv("GITOPS_REPO_URL"),
		Branch:     getEnv("GITOPS_BRANCH", "main"),
		SSHKeyPath: os.Getenv("GITOPS_SSH_KEY_PATH"),
		Username:   os.Getenv("GITOPS_USERNAME"),
		Password:   os.Getenv("GITOPS_PASSWORD"),
	}

	GlobalConfig = c
}

func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
