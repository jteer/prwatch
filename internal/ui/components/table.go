package components

import (
	"fmt"
	"image/color"
	"strings"

	lip "charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	gh "github.com/jteer/prwatch/internal/github"
	"github.com/jteer/prwatch/internal/ui/styles"
)

// Fixed column widths (chars). Title fills the remainder.
const (
	wRepo    = 12
	wNum     = 6
	wAuthor  = 10
	wAge     = 5
	wCI      = 2
	wReviews = 10
	wLabels  = 16
	wStatus  = 16
	colGap   = 1
)

// fixedWidth totals all fixed columns + inter-column gaps, optionally with labels.
func fixedWidth(showLabels bool) int {
	w := wRepo + wNum + wAuthor + wAge + wCI + wReviews + wStatus
	gaps := 7
	if showLabels {
		w += wLabels
		gaps++
	}
	return w + gaps
}

// titleWidth computes the title column width and whether the labels column fits.
func titleWidth(termW int) (tw int, showLabels bool) {
	const minTitle = 10
	if tw = termW - fixedWidth(true); tw >= minTitle {
		return tw, true
	}
	if tw = termW - fixedWidth(false); tw >= minTitle {
		return tw, false
	}
	return minTitle, false
}

// LoadingView renders a centered "loading" placeholder in exactly height lines.
func LoadingView(termW, height int) string {
	return lip.NewStyle().
		Background(styles.ColorBg).
		Foreground(styles.ColorMeta).
		Width(termW).Height(height).
		Align(lip.Center).AlignVertical(lip.Center).
		Render("⟳  loading pull requests...")
}

// PRTable renders the PR table (header + rows) in exactly `height` lines.
func PRTable(termW, height int, prs []gh.PR, selected int) string {
	tw, showLabels := titleWidth(termW)
	lines := make([]string, 0, height)

	lines = append(lines, renderHeader(termW, tw, showLabels))
	avail := height - 1

	// Scroll: keep selected visible.
	start := 0
	if selected >= avail {
		start = selected - avail + 1
	}
	end := start + avail
	if end > len(prs) {
		end = len(prs)
	}

	for i := start; i < end; i++ {
		lines = append(lines, renderRow(termW, tw, showLabels, prs[i], i == selected))
	}

	emptyBg := lip.NewStyle().Background(styles.ColorBg)
	for len(lines) < height {
		lines = append(lines, fillLine(termW, emptyBg))
	}
	return strings.Join(lines[:height], "\n")
}

func renderHeader(termW, tw int, showLabels bool) string {
	bg := styles.ColorHeader
	st := lip.NewStyle().Background(bg).Foreground(styles.ColorMeta).Bold(true)
	sp := st.Render(strings.Repeat(" ", colGap))

	cells := []string{
		styledCell(wRepo, "REPO", st),
		styledCell(wNum, "#", st),
		styledCell(tw, "TITLE", st),
		styledCell(wAuthor, "AUTHOR", st),
		styledCell(wAge, "AGE", st),
		styledCell(wCI, "CI", st),
		styledCell(wReviews, "REVIEWS", st),
	}
	if showLabels {
		cells = append(cells, styledCell(wLabels, "LABELS", st))
	}
	cells = append(cells, styledCell(wStatus, "STATUS", st))
	return joinCells(cells, sp, termW, st.Bold(false))
}

func renderRow(termW, tw int, showLabels bool, p gh.PR, sel bool) string {
	bg := urgencyBg(p.Urgency, sel)
	base := lip.NewStyle().Background(bg).Foreground(urgencyFG(p.Urgency, sel))
	if p.Urgency == gh.UrgencyDim {
		base = base.Faint(true)
	}
	sp := base.Render(strings.Repeat(" ", colGap))

	cursor := " "
	if sel {
		cursor = lip.NewStyle().Background(bg).Foreground(styles.ColorLink).Render("▍")
	}
	titleContent := cursor + trunc(p.Title, tw-1)
	titleCell := base.Width(tw).Render(titleContent)

	ciSt := ciStyle(p.CI).Background(bg)

	cells := []string{
		styledCell(wRepo, p.Repo, base.Foreground(styles.ColorMeta2).Bold(true)),
		styledCell(wNum, fmt.Sprintf("#%d", p.Number), base.Foreground(styles.ColorLink)),
		titleCell,
		styledCell(wAuthor, p.Author, base.Foreground(styles.ColorMeta)),
		styledCell(wAge, p.Age(), base.Foreground(styles.ColorMeta)),
		styledCell(wCI, p.CI.Glyph(), ciSt),
		styledCell(wReviews, p.Reviews.String(), base),
	}
	if showLabels {
		cells = append(cells, styledCell(wLabels, p.LabelString(), base.Foreground(styles.ColorMeta).Faint(true)))
	}
	cells = append(cells, styledCell(wStatus, p.Status, statusStyle(p.Urgency).Background(bg)))

	return joinCells(cells, sp, termW, base)
}

// joinCells concatenates cells with a styled spacer, then pads to termW with bg.
// Every character has an explicit background — no ANSI-reset gaps.
func joinCells(cells []string, sp string, termW int, padSt lip.Style) string {
	var b strings.Builder
	spW := ansi.StringWidth(sp)
	used := 0

	for i, c := range cells {
		if i > 0 {
			b.WriteString(sp)
			used += spW
		}
		b.WriteString(c)
		used += ansi.StringWidth(c)
	}

	remaining := termW - used
	if remaining > 0 {
		b.WriteString(padSt.Render(strings.Repeat(" ", remaining)))
	}
	return b.String()
}

// fillLine returns a full-width line using bg's background.
func fillLine(termW int, st lip.Style) string {
	return st.Width(termW).Render("")
}

// styledCell renders content in exactly w chars with style st, with explicit background.
func styledCell(w int, s string, st lip.Style) string {
	return st.Width(w).Render(trunc(s, w))
}

func urgencyFG(u gh.Urgency, sel bool) color.Color {
	if sel {
		return styles.ColorFG
	}
	switch u {
	case gh.UrgencyRed:
		return styles.ColorUrgent
	case gh.UrgencyAmber:
		return styles.ColorWarning
	case gh.UrgencyGreen:
		return styles.ColorGood
	case gh.UrgencyDim:
		return styles.ColorDim
	default:
		return styles.ColorFG
	}
}

func urgencyBg(u gh.Urgency, sel bool) color.Color {
	if sel {
		return styles.ColorSelected
	}
	switch u {
	case gh.UrgencyRed:
		return styles.ColorUrgentBg
	case gh.UrgencyAmber:
		return styles.ColorWarningBg
	case gh.UrgencyGreen:
		return styles.ColorGoodBg
	default:
		return styles.ColorBg
	}
}

func ciStyle(s gh.CIState) lip.Style {
	switch s {
	case gh.CIPass:
		return styles.CIPass
	case gh.CIFail:
		return styles.CIFail
	case gh.CIPending:
		return styles.CIPend
	default:
		return styles.MetaText
	}
}

func statusStyle(u gh.Urgency) lip.Style {
	switch u {
	case gh.UrgencyRed:
		return styles.StatusUrgent
	case gh.UrgencyAmber:
		return styles.StatusWarning
	case gh.UrgencyGreen:
		return styles.StatusGood
	case gh.UrgencyDim:
		return styles.StatusDim
	default:
		return styles.StatusNeutral
	}
}
