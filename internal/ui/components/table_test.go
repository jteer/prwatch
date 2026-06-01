package components

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	gh "github.com/jteer/prwatch/internal/github"
)

func TestFixedWidth(t *testing.T) {
	withoutLabels := fixedWidth(false)
	withLabels := fixedWidth(true)

	if withLabels <= withoutLabels {
		t.Errorf("fixedWidth(true)=%d should be > fixedWidth(false)=%d", withLabels, withoutLabels)
	}
	expected := wLabels + 1 // extra label col + gap
	if withLabels-withoutLabels != expected {
		t.Errorf("diff = %d, want %d (wLabels + 1 gap)", withLabels-withoutLabels, expected)
	}
}

func TestTitleWidth(t *testing.T) {
	tests := []struct {
		name          string
		termW         int
		wantShowLabels bool
		wantMinTitle  int
	}{
		{"wide terminal — labels fit", 200, true, 10},
		{"narrow terminal — no labels", 80, false, 10},
		{"very narrow — floor at minTitle", 10, false, 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tw, showLabels := titleWidth(tc.termW)
			if showLabels != tc.wantShowLabels {
				t.Errorf("showLabels = %v, want %v (termW=%d)", showLabels, tc.wantShowLabels, tc.termW)
			}
			if tw < tc.wantMinTitle {
				t.Errorf("titleWidth = %d, want >= %d", tw, tc.wantMinTitle)
			}
		})
	}
}

func TestPRTableLineCount(t *testing.T) {
	prs := []gh.PR{
		{Repo: "owner/repo", Number: 1, Title: "Fix bug", Author: "alice", CreatedAt: time.Now()},
		{Repo: "owner/repo", Number: 2, Title: "Add feature", Author: "bob", CreatedAt: time.Now()},
	}

	tests := []struct {
		name     string
		height   int
		prs      []gh.PR
		selected int
	}{
		{"height matches row count", 5, prs, 0},
		{"height exceeds row count — padded", 10, prs, 0},
		{"empty PRs", 6, nil, 0},
		{"selected row in range", 4, prs, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := PRTable(120, tc.height, tc.prs, tc.selected)
			lines := strings.Split(out, "\n")
			if len(lines) != tc.height {
				t.Errorf("line count = %d, want %d", len(lines), tc.height)
			}
		})
	}
}

func TestPRTableHeaderContent(t *testing.T) {
	out := PRTable(120, 5, nil, 0)
	plain := ansi.Strip(out)
	for _, col := range []string{"REPO", "TITLE", "AUTHOR", "STATUS"} {
		if !strings.Contains(plain, col) {
			t.Errorf("header missing column %q", col)
		}
	}
}

func TestPRTableRowContent(t *testing.T) {
	prs := []gh.PR{
		{Repo: "myorg/myrepo", Number: 99, Title: "My PR title", Author: "alice", CreatedAt: time.Now()},
	}
	out := PRTable(120, 5, prs, 0)
	plain := ansi.Strip(out)

	for _, want := range []string{"#99", "alice"} {
		if !strings.Contains(plain, want) {
			t.Errorf("table missing %q", want)
		}
	}
}

func TestLoadingView(t *testing.T) {
	out := LoadingView(80, 5)
	plain := ansi.Strip(out)
	if !strings.Contains(plain, "loading") {
		t.Errorf("LoadingView output missing 'loading', got: %q", plain)
	}
}
