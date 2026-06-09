package ui

import (
	"fmt"
	"image/color"
	"strings"

	lip "charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/jteer/prwatch/internal/config"
	"github.com/jteer/prwatch/internal/ui/styles"
)

type configTab int

const (
	tabRepos configTab = iota
	tabCreds
	tabNotifications
)

var configTabLabels = []string{"◫  Repositories", "⚷  Credentials", "🔔  Notifications"}

// renderConfig renders the full-screen config/settings view.
// Every character carries an explicit background to prevent terminal-color bleed.
func renderConfig(m AppModel) string {
	w := max(m.width, 40)
	h := max(m.height, 10)

	// — Title bar —
	titleBar := cfgBar(w, styles.ColorTitleBg,
		cfgSeg(styles.ColorTitleBg, styles.ColorLink, true, "  prwatch")+
			cfgSeg(styles.ColorTitleBg, styles.ColorMeta2, false, "  · settings"),
		cfgSeg(styles.ColorTitleBg, styles.ColorMeta, false, "esc to go back  "),
	)

	// — Pane head —
	paneHead := cfgBar(w, styles.ColorHeader,
		cfgSeg(styles.ColorHeader, styles.ColorGood, false, "● ")+
			cfgSeg(styles.ColorHeader, styles.ColorMeta2, false, "┤ Settings ├"),
		"",
	)

	// — Body: nav + content —
	navW := 22
	bodyH := h - 4 // title + pane-head + action-bar + 1 spare

	navLines := strings.Split(renderConfigNav(navW, bodyH, m.configTab), "\n")
	contentLines := strings.Split(renderConfigContent(w-navW-1, bodyH, m), "\n")
	sepSt := lip.NewStyle().Background(styles.ColorBg).Foreground(styles.ColorBorder)

	bodyLines := make([]string, bodyH)
	for i := 0; i < bodyH; i++ {
		nl, cl := "", ""
		if i < len(navLines) {
			nl = navLines[i]
		}
		if i < len(contentLines) {
			cl = contentLines[i]
		}
		bodyLines[i] = nl + sepSt.Render("│") + cl
	}
	body := strings.Join(bodyLines, "\n")

	// — Action bar —
	var actionLeft string
	switch m.configTab {
	case tabRepos:
		actionLeft = cfgKey("space", "pause/resume") + "  " + cfgKey("↵", "save")
	case tabCreds:
		actionLeft = cfgKey("↵", "test & save") + "  " + cfgKey("x", "sign out")
	case tabNotifications:
		actionLeft = cfgKey("space", "toggle") + "  " + cfgKey("t", "demo") + "  " + cfgKey("↵", "save")
	}
	noteColor := styles.ColorMeta
	if m.configSaved {
		noteColor = styles.ColorGood
	}
	note := cfgSeg(styles.ColorTitleBg, noteColor, false, "   written to config.yaml")
	actionBar := cfgBar(w, styles.ColorTitleBg, "  "+actionLeft+note, "")

	result := strings.Join([]string{titleBar, paneHead, body, actionBar}, "\n")

	// Pad to screen height
	have := strings.Count(result, "\n") + 1
	bgSt := lip.NewStyle().Background(styles.ColorBg)
	for have < h {
		result += "\n" + bgSt.Width(w).Render("")
		have++
	}
	return result
}

// cfgBar builds a 1-line bar: left-aligned content, right-aligned right,
// gap filled with bg. Every character has an explicit background.
func cfgBar(termW int, bg color.Color, left, right string) string {
	bgSt := lip.NewStyle().Background(bg)
	gap := termW - ansi.StringWidth(left) - ansi.StringWidth(right)
	if gap < 0 {
		gap = 0
	}
	return left + bgSt.Render(strings.Repeat(" ", gap)) + right
}

// cfgSeg renders a string segment with explicit background.
func cfgSeg(bg color.Color, fg color.Color, bold bool, s string) string {
	st := lip.NewStyle().Background(bg).Foreground(fg)
	if bold {
		st = st.Bold(true)
	}
	return st.Render(s)
}

func cfgKey(key, desc string) string {
	return styles.KeyCap.Render(key) + " " +
		lip.NewStyle().Background(styles.ColorTitleBg).Foreground(lip.Color("#aeb6c7")).Render(desc)
}

// renderConfigNav renders the left-side tab navigation.
func renderConfigNav(w, h int, active configTab) string {
	navBg := styles.ColorHeader
	navSt := lip.NewStyle().Background(navBg).Foreground(styles.ColorMeta)
	selBg := styles.ColorSelected

	var lines []string
	lines = append(lines, navSt.Width(w).Render(""))
	for i, label := range configTabLabels {
		if configTab(i) == active {
			selSt := lip.NewStyle().Background(selBg).Foreground(lip.Color("#e7eaf0"))
			bar := cfgSeg(selBg, styles.ColorLink, false, "▍") +
				selSt.Width(w-1).Render(" "+label)
			lines = append(lines, bar)
		} else {
			lines = append(lines, navSt.Width(w).Render("  "+label))
		}
		lines = append(lines, navSt.Width(w).Render(""))
	}
	blank := navSt.Width(w).Render("")
	for len(lines) < h {
		lines = append(lines, blank)
	}
	return strings.Join(lines[:h], "\n")
}

func renderConfigContent(w, h int, m AppModel) string {
	switch m.configTab {
	case tabRepos:
		return renderReposTab(w, h, m)
	case tabCreds:
		return renderCredsTab(w, h, m)
	case tabNotifications:
		return renderNotificationsTab(w, h, m)
	}
	return padContent(w, h)
}

// renderReposTab shows the repo list with pause state and selection.
func renderReposTab(w, h int, m AppModel) string {
	bg := styles.ColorBg
	bgSt := lip.NewStyle().Background(bg)
	meta := bgSt.Foreground(styles.ColorMeta)
	heading := bgSt.Foreground(lip.Color("#e7eaf0")).Bold(true)

	repos := m.client.Repos()

	var lines []string
	lines = append(lines, bgSt.Width(w).Render(""))
	lines = append(lines, heading.Width(w).Render(" Watched repositories"))
	lines = append(lines, meta.Width(w).Render(
		fmt.Sprintf("  %d repos configured · polling every %s", len(repos), m.cfg.RefreshInterval),
	))
	lines = append(lines, bgSt.Width(w).Render(""))

	// Header
	hdrSt := lip.NewStyle().Background(styles.ColorHeader).Foreground(styles.ColorMeta).Faint(true)
	lines = append(lines, hdrSt.Width(w).Render(fmt.Sprintf("  %-32s %-8s %s", "REPOSITORY", "BASE", "STATUS")))

	for i, repo := range repos {
		rowBg := bg
		if i == m.configRepoSel {
			rowBg = styles.ColorSelected
		}
		rowSt := lip.NewStyle().Background(rowBg).Foreground(styles.ColorFG)
		metaSt := lip.NewStyle().Background(rowBg).Foreground(styles.ColorMeta)

		var statusStr string
		if m.pausedRepos[repo] {
			statusStr = lip.NewStyle().Background(rowBg).Foreground(styles.ColorDim).Render("[ ] paused")
		} else {
			statusStr = lip.NewStyle().Background(rowBg).Foreground(styles.ColorGood).Bold(true).Render("[✓] watched")
		}

		// Cursor
		cursor := "  "
		if i == m.configRepoSel {
			cursor = lip.NewStyle().Background(rowBg).Foreground(styles.ColorLink).Render("▍ ")
		}

		repoStr := metaSt.Render(fmt.Sprintf("%-32s", trunc32(repo, 32)))
		baseStr := metaSt.Render(fmt.Sprintf("%-8s", "main"))

		line := cursor + rowSt.Render("") + repoStr + baseStr + statusStr
		// Pad to width
		lineW := ansi.StringWidth(line)
		if lineW < w {
			line += lip.NewStyle().Background(rowBg).Width(w - lineW).Render("")
		}
		lines = append(lines, line)
	}

	lines = append(lines, bgSt.Width(w).Render(""))
	lines = append(lines, meta.Width(w).Render("  + add repo  (org/repo · ↵ to add)"))

	blank := bgSt.Width(w).Render("")
	for len(lines) < h {
		lines = append(lines, blank)
	}
	return strings.Join(lines[:h], "\n")
}

// renderCredsTab shows token source, username, and refresh interval.
// Uses m.tokenSrc (resolved once at startup) — never calls exec in the render path.
func renderCredsTab(w, h int, m AppModel) string {
	bg := styles.ColorBg
	bgSt := lip.NewStyle().Background(bg)
	meta := bgSt.Foreground(styles.ColorMeta)
	heading := bgSt.Foreground(lip.Color("#e7eaf0")).Bold(true)
	good := bgSt.Foreground(styles.ColorGood)
	warn := bgSt.Foreground(styles.ColorUrgent)
	boldKey := bgSt.Foreground(styles.ColorMeta2).Bold(true)

	tokenSrc := m.tokenSrc
	hasToken := tokenSrc != config.SourceNone

	var authLine string
	if hasToken {
		var srcStr string
		switch tokenSrc {
		case config.SourceGHCLI:
			srcStr = "via gh cli  (gh auth login)"
		case config.SourceEnvVar:
			srcStr = "via " + m.cfg.GitHub.TokenEnv
		}
		authLine = good.Render("✓ authenticated") + meta.Render("  "+srcStr)
	} else {
		authLine = warn.Render("✗ no token") +
			meta.Render("  run `gh auth login` or set "+m.cfg.GitHub.TokenEnv)
	}

	var lines []string
	lines = append(lines, bgSt.Width(w).Render(""))
	lines = append(lines, heading.Width(w).Render(" GitHub credentials"))
	lines = append(lines, meta.Width(w).Render("  Token source · username · refresh interval"))
	lines = append(lines, bgSt.Width(w).Render(""))
	lines = append(lines, bgSt.Width(w).Render("  "+authLine))
	lines = append(lines, bgSt.Width(w).Render(""))
	lines = append(lines, boldKey.Render("  Token source")+bgSt.Width(w-14).Render(""))
	switch tokenSrc {
	case config.SourceGHCLI:
		lines = append(lines, bgSt.Width(w).Render("  gh cli  (active session from gh auth login)"))
	case config.SourceEnvVar:
		lines = append(lines, bgSt.Width(w).Render("  env  "+m.cfg.GitHub.TokenEnv+"=***"))
	default:
		lines = append(lines, bgSt.Width(w).Render("  not configured"))
	}
	lines = append(lines, bgSt.Width(w).Render(""))
	lines = append(lines, boldKey.Render("  GitHub username")+bgSt.Width(w-16).Render(""))
	lines = append(lines, bgSt.Width(w).Render("  "+m.cfg.GitHub.Username))
	lines = append(lines, bgSt.Width(w).Render(""))
	lines = append(lines, boldKey.Render("  Refresh interval")+bgSt.Width(w-17).Render(""))
	lines = append(lines, bgSt.Width(w).Render("  "+m.cfg.RefreshInterval))

	blank := bgSt.Width(w).Render("")
	for len(lines) < h {
		lines = append(lines, blank)
	}
	return strings.Join(lines[:h], "\n")
}

func renderNotificationsTab(w, h int, m AppModel) string {
	bg := styles.ColorBg
	bgSt := lip.NewStyle().Background(bg)
	meta := bgSt.Foreground(styles.ColorMeta)
	heading := bgSt.Foreground(lip.Color("#e7eaf0")).Bold(true)
	boldKey := bgSt.Foreground(styles.ColorMeta2).Bold(true)

	var toggleStr string
	if m.cfg.Notifications {
		toggleStr = lip.NewStyle().Background(bg).Foreground(styles.ColorGood).Bold(true).Render("[✓] enabled")
	} else {
		toggleStr = lip.NewStyle().Background(bg).Foreground(styles.ColorDim).Render("[ ] disabled")
	}

	var lines []string
	lines = append(lines, bgSt.Width(w).Render(""))
	lines = append(lines, heading.Width(w).Render(" Notifications"))
	lines = append(lines, meta.Width(w).Render("  macOS system alerts when PRs update"))
	lines = append(lines, bgSt.Width(w).Render(""))
	lines = append(lines, bgSt.Width(w).Render("  "+toggleStr))
	lines = append(lines, bgSt.Width(w).Render(""))
	lines = append(lines, boldKey.Render("  Notifies on")+bgSt.Width(w-12).Render(""))
	lines = append(lines, meta.Width(w).Render("  · new commits"))
	lines = append(lines, meta.Width(w).Render("  · CI updates"))
	lines = append(lines, meta.Width(w).Render("  · new reviews"))
	lines = append(lines, meta.Width(w).Render("  · other activity (comments, labels…)"))
	lines = append(lines, bgSt.Width(w).Render(""))
	lines = append(lines, meta.Width(w).Render("  press [t] to send a demo notification"))

	blank := bgSt.Width(w).Render("")
	for len(lines) < h {
		lines = append(lines, blank)
	}
	return strings.Join(lines[:h], "\n")
}

func padContent(w, h int) string {
	blank := lip.NewStyle().Background(styles.ColorBg).Width(w).Render("")
	lines := make([]string, h)
	for i := range lines {
		lines[i] = blank
	}
	return strings.Join(lines, "\n")
}

func trunc32(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}
