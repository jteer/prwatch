package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestApplyDefaults(t *testing.T) {
	tests := []struct {
		name  string
		input Config
		want  Config
	}{
		{
			name:  "all empty",
			input: Config{},
			want: Config{
				GitHub:          GitHubCfg{TokenEnv: "PRWATCH_TOKEN"},
				RefreshInterval: "60s",
				Layout:          "twopane",
			},
		},
		{
			name: "all set — no override",
			input: Config{
				GitHub:          GitHubCfg{TokenEnv: "MY_TOKEN"},
				RefreshInterval: "30s",
				Layout:          "split",
			},
			want: Config{
				GitHub:          GitHubCfg{TokenEnv: "MY_TOKEN"},
				RefreshInterval: "30s",
				Layout:          "split",
			},
		},
		{
			name: "partial — only token set",
			input: Config{
				GitHub: GitHubCfg{TokenEnv: "CUSTOM_TOKEN"},
			},
			want: Config{
				GitHub:          GitHubCfg{TokenEnv: "CUSTOM_TOKEN"},
				RefreshInterval: "60s",
				Layout:          "twopane",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.input.applyDefaults()
			if tc.input.GitHub.TokenEnv != tc.want.GitHub.TokenEnv {
				t.Errorf("TokenEnv = %q, want %q", tc.input.GitHub.TokenEnv, tc.want.GitHub.TokenEnv)
			}
			if tc.input.RefreshInterval != tc.want.RefreshInterval {
				t.Errorf("RefreshInterval = %q, want %q", tc.input.RefreshInterval, tc.want.RefreshInterval)
			}
			if tc.input.Layout != tc.want.Layout {
				t.Errorf("Layout = %q, want %q", tc.input.Layout, tc.want.Layout)
			}
		})
	}
}

func TestInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval string
		want     time.Duration
	}{
		{"valid seconds", "30s", 30 * time.Second},
		{"valid minutes", "5m", 5 * time.Minute},
		{"invalid falls back", "not-a-duration", 60 * time.Second},
		{"empty falls back", "", 60 * time.Second},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{RefreshInterval: tc.interval}
			if got := c.Interval(); got != tc.want {
				t.Errorf("Interval() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolvedRepos(t *testing.T) {
	tests := []struct {
		name  string
		org   string
		repos []string
		want  []string
	}{
		{
			name:  "all qualified",
			repos: []string{"owner/repo1", "other/repo2"},
			want:  []string{"owner/repo1", "other/repo2"},
		},
		{
			name:  "bare names with org",
			org:   "myorg",
			repos: []string{"repo1", "repo2"},
			want:  []string{"myorg/repo1", "myorg/repo2"},
		},
		{
			name:  "mixed — qualified and bare",
			org:   "myorg",
			repos: []string{"other/repo1", "bare-repo"},
			want:  []string{"other/repo1", "myorg/bare-repo"},
		},
		{
			name:  "bare without org — best effort",
			repos: []string{"orphan"},
			want:  []string{"orphan"},
		},
		{
			name:  "empty repos",
			org:   "myorg",
			repos: []string{},
			want:  []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{Org: tc.org, Repos: tc.repos}
			got := c.ResolvedRepos()
			if len(got) != len(tc.want) {
				t.Fatalf("len = %d, want %d; got %v", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestLoadAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		GitHub:          GitHubCfg{Username: "jared", TokenEnv: "MY_TOKEN"},
		Org:             "myorg",
		Repos:           []string{"myorg/repo1"},
		RefreshInterval: "120s",
		Layout:          "twopane",
	}

	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.GitHub.Username != cfg.GitHub.Username {
		t.Errorf("Username = %q, want %q", loaded.GitHub.Username, cfg.GitHub.Username)
	}
	if loaded.Org != cfg.Org {
		t.Errorf("Org = %q, want %q", loaded.Org, cfg.Org)
	}
	if loaded.Layout != cfg.Layout {
		t.Errorf("Layout = %q, want %q", loaded.Layout, cfg.Layout)
	}
}

func TestNotificationsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		GitHub:        GitHubCfg{Username: "alice", TokenEnv: "MY_TOKEN"},
		Notifications: true,
	}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !loaded.Notifications {
		t.Error("Notifications = false after round-trip, want true")
	}
}

func TestNotificationsDefaultsFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// Config written without notifications field
	cfg := &Config{GitHub: GitHubCfg{Username: "alice", TokenEnv: "MY_TOKEN"}}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Notifications {
		t.Error("Notifications should default to false when absent")
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/does/not/exist/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(":\tinvalid:yaml:::\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}
