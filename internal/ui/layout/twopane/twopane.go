// Package twopane implements Layout 1: table (70%) + detail (30%).
// Full-row urgency tint, footer keybar. Help overlay composited on top.
package twopane

import (
	"strings"

	gh "github.com/jteer/prwatch/internal/github"
	"github.com/jteer/prwatch/internal/ui/components"
	"github.com/jteer/prwatch/internal/ui/layout"
)

// Layout renders the two-pane view:
//   - Title bar          (1 line)
//   - PR table pane head (1 line)
//   - PR table           (70% of remaining height)
//   - Detail pane head   (1 line)
//   - Detail panel       (30% of remaining height)
//   - Footer keybar      (1 line)
//
// Help overlay is composited on top of the base view so the table shows through.
type Layout struct{}

func New() *Layout { return &Layout{} }

func (l *Layout) Name() string { return "twopane" }

func (l *Layout) View(s layout.State) string {
	w := max(s.Width, 40)
	h := max(s.Height, 10)

	// chrome: title + 2×pane-head + footer
	const chrome = 4
	content := max(h-chrome, 6)
	tableH := max((content*70)/100, 3)
	detailH := max(content-tableH, 2)

	var selPR *gh.PR
	if len(s.PRs) > 0 && s.Selected < len(s.PRs) {
		pr := s.PRs[s.Selected]
		selPR = &pr
	}

	tableHead := components.PaneHead(w,
		"┤ Pull Requests ├",
		"sort: "+s.SortField+"   scope: "+s.Scope,
		s.FocusPane == 0,
	)

	detailRight := ""
	if selPR != nil {
		detailRight = selPR.Repo + " #" + itoa(selPR.Number)
	}
	detailHead := components.PaneHead(w, "┤ Detail ├", detailRight, s.FocusPane == 1)

	var tableContent string
	if s.Loading && len(s.PRs) == 0 {
		tableContent = components.LoadingView(w, tableH)
	} else {
		tableContent = components.PRTable(w, tableH, s.PRs, s.Selected)
	}

	base := strings.Join([]string{
		components.TitleBar(w, s),
		tableHead,
		tableContent,
		detailHead,
		components.DetailPanel(w, detailH, selPR),
		components.Footer(w, s),
	}, "\n")

	// Help card is composited on top so the table remains visible underneath.
	if s.ShowHelp {
		return components.HelpOverlay(base, w, h)
	}

	return base
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}

