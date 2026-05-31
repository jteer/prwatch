package components

import (
	"strings"

	lip "charm.land/lipgloss/v2"

	"github.com/jteer/prwatch/internal/ui/styles"
)

var helpKeys = [][2]string{
	{"j / k", "move up / down"},
	{"↑ / ↓", "move up / down"},
	{"/", "fuzzy filter (repo · title · author)"},
	{"s", "cycle sort (age → status → repo)"},
	{"f", "scope (ALL · mine · review req.)"},
	{"o", "open PR in browser"},
	{"r", "refresh now"},
	{"Tab", "toggle pane focus"},
	{"c", "config / settings"},
	{"g / G", "jump top / bottom"},
	{"?", "toggle this help"},
	{"q", "quit"},
}

// HelpOverlay composites the help card on top of baseView.
// baseView is the fully rendered base layout string.
func HelpOverlay(baseView string, termW, termH int) string {
	keyCap := lip.NewStyle().
		Background(styles.ColorMeta).
		Foreground(styles.ColorBg).
		Bold(true).
		Padding(0, 1)
	desc := lip.NewStyle().
		Background(styles.ColorDetailBg).
		Foreground(styles.ColorFG)

	var rows []string
	for _, kd := range helpKeys {
		row := lip.JoinHorizontal(lip.Left,
			keyCap.Render(kd[0]),
			desc.Render("  "+kd[1]),
		)
		rows = append(rows, row)
	}

	title := lip.NewStyle().
		Background(styles.ColorDetailBg).
		Foreground(lip.Color("#e7eaf0")).
		Bold(true).
		Render("prwatch · keybindings")
	footer := lip.NewStyle().
		Background(styles.ColorDetailBg).
		Foreground(styles.ColorMeta).
		Render("esc or ? to close")

	card := styles.HelpCard.Render(
		title + "\n\n" +
			strings.Join(rows, "\n") +
			"\n\n" + footer,
	)

	return OverlayCenter(baseView, card, termW, termH)
}
