package twopane

import (
	"strings"
	"testing"
	"time"

	gh "github.com/jteer/prwatch/internal/github"
	"github.com/jteer/prwatch/internal/ui/layout"
)

func TestItoa(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{100, "100"},
		{9999, "9999"},
	}

	for _, tc := range tests {
		if got := itoa(tc.n); got != tc.want {
			t.Errorf("itoa(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 2},
		{5, 3, 5},
		{0, 0, 0},
		{-1, 1, 1},
	}

	for _, tc := range tests {
		if got := max(tc.a, tc.b); got != tc.want {
			t.Errorf("max(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func baseState() layout.State {
	return layout.State{
		PRs: []gh.PR{
			{Repo: "owner/repo", Number: 1, Title: "Test PR", Author: "alice", CreatedAt: time.Now()},
		},
		Selected:    0,
		FilterQuery: "",
		SortField:   "age",
		Scope:       "ALL",
		FocusPane:   0,
		ShowHelp:    false,
		Loading:     false,
		LastUpdated: "12:00",
		RepoCount:   1,
		Width:       120,
		Height:      30,
	}
}

func TestViewLineCount(t *testing.T) {
	l := New()
	s := baseState()
	out := l.View(s)
	lines := strings.Split(out, "\n")
	if len(lines) != s.Height {
		t.Errorf("View line count = %d, want %d", len(lines), s.Height)
	}
}

func TestViewName(t *testing.T) {
	l := New()
	if l.Name() != "twopane" {
		t.Errorf("Name() = %q, want %q", l.Name(), "twopane")
	}
}

func TestViewMinSizeClamped(t *testing.T) {
	l := New()
	s := baseState()
	s.Width = 1
	s.Height = 1
	// Should not panic, width/height clamped internally.
	out := l.View(s)
	if out == "" {
		t.Error("View returned empty string for min size")
	}
}

func TestViewLoadingState(t *testing.T) {
	l := New()
	s := baseState()
	s.Loading = true
	s.PRs = nil
	out := l.View(s)
	if !strings.Contains(out, "loading") {
		t.Errorf("loading state output missing 'loading'")
	}
}

func TestViewHelpOverlay(t *testing.T) {
	l := New()
	s := baseState()
	s.ShowHelp = true
	out := l.View(s)
	if !strings.Contains(out, "keybindings") {
		t.Errorf("help overlay missing 'keybindings'")
	}
}

func TestViewNoPRsSelected(t *testing.T) {
	l := New()
	s := baseState()
	s.PRs = nil
	s.Selected = 0
	// Should not panic with no PRs.
	_ = l.View(s)
}
