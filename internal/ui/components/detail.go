package components

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/glamour/v2"
	lip "charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	gh "github.com/jteer/prwatch/internal/github"
	"github.com/jteer/prwatch/internal/ui/styles"
)

// DetailPanel renders the bottom detail pane for the selected PR in exactly `height` lines.
func DetailPanel(termW, height int, p *gh.PR) string {
	if p == nil {
		return emptyDetail(termW, height)
	}

	bg := styles.ColorDetailBg
	bgSt := lip.NewStyle().Background(bg).Foreground(styles.ColorFG)
	metaSt := lip.NewStyle().Background(bg).Foreground(styles.ColorMeta)
	boldKey := lip.NewStyle().Background(bg).Foreground(styles.ColorMeta2).Bold(true)

	lines := make([]string, 0, height)

	// — Title line —
	titleStr := lip.JoinHorizontal(lip.Left,
		styles.RepoText.Render(p.Repo),
		metaSt.Render(" "),
		styles.LinkText.Render(fmt.Sprintf("#%d", p.Number)),
		metaSt.Render(" · "),
		lip.NewStyle().Background(bg).Foreground(lip.Color("#e7eaf0")).Bold(true).Render(
			trunc(p.Title, termW-len([]rune(p.Repo))-10),
		),
	)
	lines = append(lines, bgSt.Width(termW).Render(titleStr))

	// — Body (glamour markdown, capped to 3 lines) —
	// Strip ANSI from glamour output before rendering with our background.
	// Glamour embeds its own background codes which override our wrapper via
	// inner ANSI resets — stripping them and re-rendering keeps the background clean.
	const bodyBudget = 3
	rawBodyLines := renderMarkdown(p.Body, termW-2, bg)
	for i := 0; i < bodyBudget; i++ {
		if i < len(rawBodyLines) {
			plain := ansi.Strip(rawBodyLines[i])
			lines = append(lines, bgSt.Width(termW).Render(plain))
		} else {
			lines = append(lines, bgSt.Width(termW).Render(""))
		}
	}

	// — Section split: reviewers left, CI right —
	leftW := (termW * 6) / 10
	rightW := termW - leftW - 1

	secH := lip.NewStyle().Background(bg).Foreground(styles.ColorMeta).Faint(true)
	sep := lip.NewStyle().Background(bg).Foreground(styles.ColorBorder).Render("│")

	lines = append(lines, bgSt.Width(termW).Render(
		lip.JoinHorizontal(lip.Left,
			secH.Width(leftW).Render(" REVIEWERS"),
			sep,
			secH.Width(rightW).Render(" CI CHECKS"),
		),
	))

	revLines := buildReviewerLines(p.Reviewers, bg)
	ckLines := buildCheckLines(p.Checks)
	maxRows := max(len(revLines), len(ckLines))

	for i := 0; i < maxRows; i++ {
		lv, rv := "", ""
		if i < len(revLines) {
			lv = revLines[i]
		}
		if i < len(ckLines) {
			rv = ckLines[i]
		}
		row := bgSt.Width(termW).Render(
			lip.JoinHorizontal(lip.Left,
				lip.NewStyle().Background(bg).Width(leftW).Render(lv),
				sep,
				lip.NewStyle().Background(bg).Width(rightW).Render(rv),
			),
		)
		lines = append(lines, row)
	}

	// — Meta —
	labStr := boldKey.Render("labels") + " " + metaSt.Render(trunc(p.LabelString(), 28))
	baseStr := "  " + boldKey.Render("base") + " " + metaSt.Render(p.BaseBranch)
	milStr := ""
	if p.Milestone != "" {
		milStr = "  " + boldKey.Render("milestone") + " " + metaSt.Render(trunc(p.Milestone, 22))
	}
	lines = append(lines, bgSt.Width(termW).Render(labStr+baseStr+milStr))

	updStr := boldKey.Render("updated") + " " + metaSt.Render(p.UpdatedAt.Format("2006-01-02 15:04"))
	lines = append(lines, bgSt.Width(termW).Render(updStr))

	// Pad to height
	empty := bgSt.Width(termW).Render("")
	for len(lines) < height {
		lines = append(lines, empty)
	}

	return strings.Join(lines[:height], "\n")
}

// renderMarkdown renders md as terminal-formatted text using glamour.
// Returns individual display lines, stripped of leading/trailing blank lines.
func renderMarkdown(md string, width int, bg color.Color) []string {
	if md == "" {
		return nil
	}

	// "notty" has no box-drawing borders on code blocks — those would render
	// as near-invisible chars against a dark terminal background.
	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath("notty"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return []string{trunc(md, width)}
	}

	out, err := r.Render(md)
	if err != nil {
		return []string{trunc(md, width)}
	}

	// Split and trim surrounding blank lines added by glamour.
	raw := strings.Split(strings.TrimRight(strings.TrimLeft(out, "\n"), "\n"), "\n")
	result := make([]string, 0, len(raw))
	for _, l := range raw {
		result = append(result, l)
	}
	return result
}

func buildReviewerLines(revs []gh.Reviewer, bg color.Color) []string {
	lines := make([]string, 0, len(revs))
	bgSt := lip.NewStyle().Background(bg)
	for _, r := range revs {
		var decIcon string
		var decColor color.Color
		switch r.Decision {
		case "APPROVED":
			decIcon = "✓"
			decColor = styles.ColorGood
		case "CHANGES_REQUESTED":
			decIcon = "✗"
			decColor = styles.ColorUrgent
		default:
			decIcon = "⏳"
			decColor = styles.ColorWarning
		}

		avatarSt := lip.NewStyle().
			Background(lip.Color("#7f8aa3")).
			Foreground(styles.ColorBg).
			Bold(true)
		if r.IsTeam {
			avatarSt = avatarSt.
				Border(lip.Border{Left: "▏"}, false, false, false, true).
				BorderForeground(styles.ColorLink)
		}
		avatar := avatarSt.Render(r.Initials())
		dec := lip.NewStyle().Background(bg).Foreground(decColor).Bold(true).
			Render(decIcon + " " + r.Decision)
		line := bgSt.Render(" ") + avatar + bgSt.Render(" "+trunc(r.Login, 14)+"  ") + dec
		lines = append(lines, line)
	}
	return lines
}

func buildCheckLines(checks []gh.Check) []string {
	lines := make([]string, 0, len(checks))
	for _, c := range checks {
		glyph := ciStyle(c.State).Render(c.State.Glyph())
		nm := trunc(c.Name, 26)
		note := ""
		if c.Note != "" {
			note = lip.NewStyle().Foreground(styles.ColorUrgent).Render(" · " + trunc(c.Note, 16))
		}
		lines = append(lines, " "+glyph+" "+nm+note)
	}
	return lines
}

func emptyDetail(termW, height int) string {
	return lip.NewStyle().
		Background(styles.ColorDetailBg).
		Foreground(styles.ColorMeta).
		Width(termW).
		Height(height).
		Align(lip.Center).
		AlignVertical(lip.Center).
		Render("no PR selected")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
