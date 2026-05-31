package components

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// OverlayCenter composites cardStr centered on top of baseStr.
// baseStr must be termW×termH (each line separated by \n).
func OverlayCenter(baseStr, cardStr string, termW, termH int) string {
	baseLines := strings.Split(baseStr, "\n")
	cardLines := strings.Split(cardStr, "\n")

	cardH := len(cardLines)
	cardW := 0
	for _, l := range cardLines {
		if w := ansi.StringWidth(l); w > cardW {
			cardW = w
		}
	}

	startY := (termH - cardH) / 2
	startX := (termW - cardW) / 2
	if startY < 0 {
		startY = 0
	}
	if startX < 0 {
		startX = 0
	}

	result := make([]string, termH)
	for i := 0; i < termH; i++ {
		if i < len(baseLines) {
			result[i] = padToWidth(baseLines[i], termW)
		} else {
			result[i] = strings.Repeat(" ", termW)
		}
	}

	for i, cardLine := range cardLines {
		row := startY + i
		if row >= termH {
			break
		}
		result[row] = spliceLine(result[row], cardLine, startX, termW)
	}

	return strings.Join(result, "\n")
}

// padToWidth ensures s is exactly w display chars, padding with spaces.
func padToWidth(s string, w int) string {
	sw := ansi.StringWidth(s)
	if sw < w {
		return s + strings.Repeat(" ", w-sw)
	}
	return s
}

// spliceLine inserts over into base at display column x, replacing overW chars.
func spliceLine(base, over string, x, termW int) string {
	overW := ansi.StringWidth(over)

	left := ansi.Truncate(base, x, "")
	leftW := ansi.StringWidth(left)
	if leftW < x {
		left += strings.Repeat(" ", x-leftW)
	}

	right := ansi.TruncateLeft(base, x+overW, "")

	result := left + over + right
	rw := ansi.StringWidth(result)
	if rw < termW {
		result += strings.Repeat(" ", termW-rw)
	}
	return result
}
