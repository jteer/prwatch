package config

import (
	"errors"
	"testing"
)

type fakeCommander struct {
	lookPathErr error
	output      []byte
	outputErr   error
}

func (f fakeCommander) LookPath(string) (string, error) {
	if f.lookPathErr != nil {
		return "", f.lookPathErr
	}
	return "/usr/bin/gh", nil
}

func (f fakeCommander) Output(string, ...string) ([]byte, error) {
	return f.output, f.outputErr
}

var errExec = errors.New("exec failed")

func TestResolveToken(t *testing.T) {
	tests := []struct {
		name       string
		cmd        fakeCommander
		tokenEnv   string
		tokenVal   string
		wantToken  string
		wantSource TokenSource
	}{
		{
			name:       "gh cli token takes priority over env var",
			cmd:        fakeCommander{output: []byte("ghtoken\n")},
			tokenEnv:   "MY_TOK",
			tokenVal:   "envtoken",
			wantToken:  "ghtoken",
			wantSource: SourceGHCLI,
		},
		{
			name:       "gh output fails — env var fallback",
			cmd:        fakeCommander{outputErr: errExec},
			tokenEnv:   "MY_TOK",
			tokenVal:   "envtoken",
			wantToken:  "envtoken",
			wantSource: SourceEnvVar,
		},
		{
			name:       "gh not in PATH — env var fallback",
			cmd:        fakeCommander{lookPathErr: errExec},
			tokenEnv:   "MY_TOK",
			tokenVal:   "envtoken",
			wantToken:  "envtoken",
			wantSource: SourceEnvVar,
		},
		{
			name:       "gh fails and env var unset — SourceNone",
			cmd:        fakeCommander{outputErr: errExec},
			tokenEnv:   "MY_TOK",
			wantToken:  "",
			wantSource: SourceNone,
		},
		{
			name:       "gh fails and empty tokenEnv — SourceNone",
			cmd:        fakeCommander{outputErr: errExec},
			tokenEnv:   "",
			wantToken:  "",
			wantSource: SourceNone,
		},
		{
			name:       "gh not in PATH and no env var — SourceNone",
			cmd:        fakeCommander{lookPathErr: errExec},
			tokenEnv:   "",
			wantToken:  "",
			wantSource: SourceNone,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.tokenVal != "" {
				t.Setenv(tc.tokenEnv, tc.tokenVal)
			}

			tok, src := resolveToken(tc.cmd, tc.tokenEnv)

			if tok != tc.wantToken {
				t.Errorf("token = %q, want %q", tok, tc.wantToken)
			}
			if src != tc.wantSource {
				t.Errorf("source = %q, want %q", src, tc.wantSource)
			}
		})
	}
}

func TestResolveUsername(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		cmd        fakeCommander
		want       string
	}{
		{
			name:       "configured value used directly — no gh call",
			configured: "alice",
			cmd:        fakeCommander{outputErr: errExec},
			want:       "alice",
		},
		{
			name: "empty configured — gh cli fallback",
			cmd:  fakeCommander{output: []byte("ghuser\n")},
			want: "ghuser",
		},
		{
			name: "empty configured — gh output fails — empty result",
			cmd:  fakeCommander{outputErr: errExec},
			want: "",
		},
		{
			name: "empty configured — gh not in PATH — empty result",
			cmd:  fakeCommander{lookPathErr: errExec},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveUsername(tc.cmd, tc.configured); got != tc.want {
				t.Errorf("resolveUsername(%q) = %q, want %q", tc.configured, got, tc.want)
			}
		})
	}
}
