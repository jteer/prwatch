package config

import (
	"os"
	"os/exec"
	"strings"
)

// TokenSource identifies how a token was obtained.
type TokenSource string

const (
	SourceGHCLI  TokenSource = "gh cli"
	SourceEnvVar TokenSource = "env"
	SourceNone   TokenSource = "none"
)

// ResolveToken returns the best available GitHub token and its source.
// Priority:
//  1. `gh auth token` — piggybacks on an active `gh auth login` session
//  2. os.Getenv(tokenEnv) — explicit token from config/env
func ResolveToken(tokenEnv string) (token string, source TokenSource) {
	if tok := ghCLIToken(); tok != "" {
		return tok, SourceGHCLI
	}
	if tokenEnv != "" {
		if tok := os.Getenv(tokenEnv); tok != "" {
			return tok, SourceEnvVar
		}
	}
	return "", SourceNone
}

// ResolveUsername returns the GitHub username.
// If cfg has one set, uses it. Otherwise asks `gh api user -q .login`.
func ResolveUsername(configured string) string {
	if configured != "" {
		return configured
	}
	return ghCLIUsername()
}

// ghCLIToken runs `gh auth token` and returns the active session token.
// Returns "" if gh is not installed or the user is not authenticated.
func ghCLIToken() string {
	path, err := exec.LookPath("gh")
	if err != nil {
		return ""
	}
	out, err := exec.Command(path, "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ghCLIUsername returns the authenticated GitHub username via `gh api user`.
// Returns "" on any failure.
func ghCLIUsername() string {
	path, err := exec.LookPath("gh")
	if err != nil {
		return ""
	}
	out, err := exec.Command(path, "api", "user", "-q", ".login").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
