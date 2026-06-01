package config

import (
	"os"
	"os/exec"
	"strings"
)

// Commander abstracts exec.LookPath and command output for testing.
type Commander interface {
	LookPath(file string) (string, error)
	Output(name string, args ...string) ([]byte, error)
}

type realCommander struct{}

func (realCommander) LookPath(file string) (string, error) { return exec.LookPath(file) }
func (realCommander) Output(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

var defaultCommander Commander = realCommander{}

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
func ResolveToken(tokenEnv string) (string, TokenSource) {
	return resolveToken(defaultCommander, tokenEnv)
}

// ResolveUsername returns the GitHub username.
// If cfg has one set, uses it. Otherwise asks `gh api user -q .login`.
func ResolveUsername(configured string) string {
	return resolveUsername(defaultCommander, configured)
}

func resolveToken(cmd Commander, tokenEnv string) (string, TokenSource) {
	if tok := ghCLIToken(cmd); tok != "" {
		return tok, SourceGHCLI
	}
	if tokenEnv != "" {
		if tok := os.Getenv(tokenEnv); tok != "" {
			return tok, SourceEnvVar
		}
	}
	return "", SourceNone
}

func resolveUsername(cmd Commander, configured string) string {
	if configured != "" {
		return configured
	}
	return ghCLIUsername(cmd)
}

func ghCLIToken(cmd Commander) string {
	path, err := cmd.LookPath("gh")
	if err != nil {
		return ""
	}
	out, err := cmd.Output(path, "auth", "token")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func ghCLIUsername(cmd Commander) string {
	path, err := cmd.LookPath("gh")
	if err != nil {
		return ""
	}
	out, err := cmd.Output(path, "api", "user", "-q", ".login")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
