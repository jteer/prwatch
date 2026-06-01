package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/jteer/prwatch/internal/ui/layout"
)

func TestPlural(t *testing.T) {
	tests := []struct {
		n    int
		word string
		want string
	}{
		{0, "PR", "0 PRs"},
		{1, "PR", "1 PR"},
		{2, "PR", "2 PRs"},
		{1, "repo", "1 repo"},
		{5, "repo", "5 repos"},
	}

	for _, tc := range tests {
		if got := plural(tc.n, tc.word); got != tc.want {
			t.Errorf("plural(%d, %q) = %q, want %q", tc.n, tc.word, got, tc.want)
		}
	}
}

func TestTitleBarLineCount(t *testing.T) {
	s := layout.State{PRs: nil, RepoCount: 2, LastUpdated: "now", Width: 80}
	out := TitleBar(80, s)
	lines := strings.Split(out, "\n")
	if len(lines) != 1 {
		t.Errorf("TitleBar returned %d lines, want 1", len(lines))
	}
}

func TestTitleBarContent(t *testing.T) {
	s := layout.State{RepoCount: 3, LastUpdated: "12:00", Width: 80}
	out := ansi.Strip(TitleBar(80, s))
	if !strings.Contains(out, "prwatch") {
		t.Errorf("TitleBar missing 'prwatch': %q", out)
	}
	if !strings.Contains(out, "3 repos") {
		t.Errorf("TitleBar missing '3 repos': %q", out)
	}
}

func TestTitleBarLoadingState(t *testing.T) {
	s := layout.State{Loading: true, Width: 80}
	out := ansi.Strip(TitleBar(80, s))
	if !strings.Contains(out, "refreshing") {
		t.Errorf("TitleBar loading state missing 'refreshing': %q", out)
	}
}

func TestPaneHeadFocus(t *testing.T) {
	focused := ansi.Strip(PaneHead(80, "┤ Pane ├", "right", true))
	unfocused := ansi.Strip(PaneHead(80, "┤ Pane ├", "right", false))

	if !strings.Contains(focused, "●") {
		t.Errorf("focused PaneHead missing ●: %q", focused)
	}
	if strings.Contains(unfocused, "●") {
		t.Errorf("unfocused PaneHead should not contain ●: %q", unfocused)
	}
}

func TestPaneHeadContent(t *testing.T) {
	out := ansi.Strip(PaneHead(80, "┤ Detail ├", "owner/repo #42", false))
	if !strings.Contains(out, "┤ Detail ├") {
		t.Errorf("PaneHead missing label: %q", out)
	}
	if !strings.Contains(out, "owner/repo #42") {
		t.Errorf("PaneHead missing right text: %q", out)
	}
}

func TestFooterLineCount(t *testing.T) {
	s := layout.State{Scope: "ALL", LastUpdated: "now"}
	out := Footer(120, s)
	lines := strings.Split(out, "\n")
	if len(lines) != 1 {
		t.Errorf("Footer returned %d lines, want 1", len(lines))
	}
}

func TestFooterContainsKeys(t *testing.T) {
	s := layout.State{Scope: "ALL", LastUpdated: "now"}
	out := ansi.Strip(Footer(120, s))
	for _, key := range []string{"j/k", "q"} {
		if !strings.Contains(out, key) {
			t.Errorf("Footer missing key %q: %q", key, out)
		}
	}
}

func TestFilterBarContent(t *testing.T) {
	out := ansi.Strip(FilterBar(80, "myquery"))
	if !strings.Contains(out, "myquery") {
		t.Errorf("FilterBar missing query text: %q", out)
	}
	if !strings.Contains(out, "esc") {
		t.Errorf("FilterBar missing hint text: %q", out)
	}
}
