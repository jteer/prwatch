// Package components provides stateless rendering functions for each TUI region.
package components

import lip "charm.land/lipgloss/v2"

// trunc truncates s to at most n runes, appending "…" if cut.
func trunc(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n == 1 {
		return string(runes[:1])
	}
	return string(runes[:n-1]) + "…"
}

// cell renders content in a fixed-width column with the given style.
func cell(w int, s string, st lip.Style) string {
	return st.Width(w).Render(trunc(s, w))
}

// fullWidth renders s left-aligned at exactly width w with style st.
func fullWidth(w int, s string, st lip.Style) string {
	return st.Width(w).Render(trunc(s, w))
}
