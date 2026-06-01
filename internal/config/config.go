package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GitHub          GitHubCfg `yaml:"github"`
	Org             string    `yaml:"org"`             // fallback owner for bare repo names
	Repos           []string  `yaml:"repos"`           // "owner/repo" or bare name (uses Org)
	RefreshInterval string    `yaml:"refresh_interval"`
	Team            []string  `yaml:"team"`
	Layout          string    `yaml:"layout"`
	Notifications   bool      `yaml:"notifications"`
}

type GitHubCfg struct {
	Username string `yaml:"username"`
	TokenEnv string `yaml:"token_env"`
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "prwatch", "config.yaml")
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.applyDefaults()
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.GitHub.TokenEnv == "" {
		c.GitHub.TokenEnv = "PRWATCH_TOKEN"
	}
	if c.RefreshInterval == "" {
		c.RefreshInterval = "60s"
	}
	if c.Layout == "" {
		c.Layout = "twopane"
	}
}

// ResolvedRepos returns each repo as "owner/name", using Org as fallback owner
// for repos without a slash (backward-compatible with the old single-org config).
func (c *Config) ResolvedRepos() []string {
	out := make([]string, 0, len(c.Repos))
	for _, r := range c.Repos {
		if strings.Contains(r, "/") {
			out = append(out, r)
		} else if c.Org != "" {
			out = append(out, c.Org+"/"+r)
		} else {
			out = append(out, r) // best effort — will error at query time
		}
	}
	return out
}

func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func (c *Config) Interval() time.Duration {
	d, err := time.ParseDuration(c.RefreshInterval)
	if err != nil {
		return 60 * time.Second
	}
	return d
}
