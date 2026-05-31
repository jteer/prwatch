package components

import (
	"fmt"
	"image/color"
	"strings"

	lip "charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/jteer/prwatch/internal/ui/layout"
	"github.com/jteer/prwatch/internal/ui/styles"
)

var footerKeys = [][2]string{
	{"j/k", "move"},
	{"/", "filter"},
	{"s", "sort"},
	{"f", "scope"},
	{"o", "open"},
	{"r", "refresh"},
	{"c", "config"},
	{"?", "help"},
	{"q", "quit"},
}

// Footer renders the keybar in exactly 1 line at termW.
// Every character carries an explicit background to prevent ANSI-reset artifacts.
func Footer(termW int, s layout.State) string {
	bg := styles.ColorTitleBg
	bgSt := lip.NewStyle().Background(bg)

	var parts []string
	for _, kd := range footerKeys {
		k := styles.KeyCap.Render(kd[0])
		d := bgSt.Foreground(lip.Color("#aeb6c7")).Render(kd[1])
		parts = append(parts, k+bgSt.Render(" ")+d)
	}
	left := strings.Join(parts, bgSt.Render("  "))

	mode := bgSt.Foreground(styles.ColorGood).Render("⦿ " + s.Scope + " · " + s.LastUpdated)

	return buildBar(termW, bg,
		bgSt.Render("  ")+left,
		mode+bgSt.Render("  "),
	)
}

// FilterBar renders the footer in filter-entry mode.
func FilterBar(termW int, query string) string {
	bg := styles.ColorTitleBg
	bgSt := lip.NewStyle().Background(bg)

	prompt := bgSt.Foreground(styles.ColorLink).Bold(true).Render("/")
	inp := bgSt.Foreground(styles.ColorFG).Render(query)
	cursor := bgSt.Foreground(styles.ColorLink).Render("▍")
	hint := bgSt.Foreground(styles.ColorMeta).Render("  esc to cancel · enter to confirm")
	content := bgSt.Render("  ") + prompt + bgSt.Render(" ") + inp + cursor + hint

	w := ansi.StringWidth(content)
	if w < termW {
		content += bgSt.Width(termW - w).Render("")
	}
	return content
}

// TitleBar renders the top title bar in exactly 1 line.
func TitleBar(termW int, s layout.State) string {
	bg := styles.ColorTitleBg
	bgSt := lip.NewStyle().Background(bg)

	left := bgSt.Foreground(styles.ColorLink).Bold(true).Render("prwatch") +
		bgSt.Foreground(styles.ColorMeta2).Render(
			fmt.Sprintf("  %s  ·  %s", plural(len(s.PRs), "PR"), plural(s.RepoCount, "repo")),
		)

	var right string
	if s.Loading {
		right = bgSt.Foreground(styles.ColorWarning).Render("⟳ refreshing...")
	} else {
		right = bgSt.Foreground(styles.ColorMeta).Render("updated " + s.LastUpdated)
	}

	return buildBar(termW, bg,
		bgSt.Render("  ")+left,
		right+bgSt.Render("  "),
	)
}

// PaneHead renders a pane header line in exactly 1 line.
func PaneHead(termW int, label, right string, focused bool) string {
	bg := styles.ColorHeader
	bgSt := lip.NewStyle().Background(bg)

	var prefix string
	if focused {
		prefix = bgSt.Foreground(styles.ColorGood).Render("●") + bgSt.Render(" ")
	} else {
		prefix = bgSt.Render("  ")
	}

	lbl := bgSt.Foreground(styles.ColorMeta2).Render(label)
	rt := bgSt.Foreground(styles.ColorMeta).Render(right)

	return buildBar(termW, bg,
		prefix+lbl,
		rt,
	)
}

// buildBar constructs a 1-line bar: left-aligned `left`, right-aligned `right`,
// gap filled with bg. Every character has an explicit background.
func buildBar(termW int, bg color.Color, left, right string) string {
	bgSt := lip.NewStyle().Background(bg)
	leftW := ansi.StringWidth(left)
	rightW := ansi.StringWidth(right)
	gap := termW - leftW - rightW
	if gap < 0 {
		gap = 0
		// Truncate right if no room.
		if leftW < termW {
			right = ansi.Truncate(right, termW-leftW, "")
		} else {
			right = ""
		}
	}
	return left + bgSt.Render(strings.Repeat(" ", gap)) + right
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
