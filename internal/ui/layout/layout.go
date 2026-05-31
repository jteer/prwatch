// Package layout defines the Layout interface.
// To add a new layout: implement Layout, register it in the model's layout slice.
package layout

import gh "github.com/jteer/prwatch/internal/github"

// State is the read-only snapshot passed to each Layout for rendering.
type State struct {
	PRs         []gh.PR
	Selected    int
	FilterQuery string
	SortField   string // "age" | "status" | "repo"
	Scope       string // "ALL" | "mine" | "review"
	FocusPane   int    // 0 = table pane, 1 = detail pane
	ShowHelp    bool
	Loading     bool
	LastUpdated string
	RepoCount   int
	Width       int
	Height      int
}

// Layout renders the full terminal screen for a given State.
// Implementing this interface is all that's needed to add a new layout.
type Layout interface {
	// Name returns the identifier used in config ("twopane", "split", etc.).
	Name() string
	// View renders the full screen and returns it as a string.
	View(s State) string
}
