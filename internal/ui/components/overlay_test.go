package components

import (
	"strings"
	"testing"
)

func TestPadToWidth(t *testing.T) {
	tests := []struct {
		name string
		s    string
		w    int
		want string
	}{
		{"exact width", "hello", 5, "hello"},
		{"shorter — padded", "hi", 5, "hi   "},
		{"longer — untouched", "toolong", 4, "toolong"},
		{"empty string", "", 3, "   "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := padToWidth(tc.s, tc.w); got != tc.want {
				t.Errorf("padToWidth(%q, %d) = %q, want %q", tc.s, tc.w, got, tc.want)
			}
		})
	}
}

func TestSpliceLine(t *testing.T) {
	tests := []struct {
		name   string
		base   string
		over   string
		x      int
		termW  int
		wantAt int    // position where 'over' content starts
		wantContains string
	}{
		{
			name:         "splice at start",
			base:         "0123456789",
			over:         "AB",
			x:            0,
			termW:        10,
			wantContains: "AB",
		},
		{
			name:         "splice in middle",
			base:         "0123456789",
			over:         "XY",
			x:            3,
			termW:        10,
			wantContains: "XY",
		},
		{
			name:         "splice preserves left",
			base:         "abcdefghij",
			over:         "ZZ",
			x:            2,
			termW:        10,
			wantContains: "ab",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := spliceLine(tc.base, tc.over, tc.x, tc.termW)
			if !strings.Contains(got, tc.wantContains) {
				t.Errorf("spliceLine output %q missing %q", got, tc.wantContains)
			}
		})
	}
}

func TestOverlayCenterLineCount(t *testing.T) {
	base := strings.Repeat("x", 20) + "\n" +
		strings.Repeat("x", 20) + "\n" +
		strings.Repeat("x", 20) + "\n" +
		strings.Repeat("x", 20) + "\n" +
		strings.Repeat("x", 20)

	card := "CARD\nLINE"

	out := OverlayCenter(base, card, 20, 5)
	lines := strings.Split(out, "\n")
	if len(lines) != 5 {
		t.Errorf("OverlayCenter line count = %d, want 5", len(lines))
	}
}

func TestOverlayCenterCardContent(t *testing.T) {
	base := strings.Repeat(strings.Repeat(".", 30)+"\n", 9)
	base = strings.TrimRight(base, "\n")
	card := "HELLO"

	out := OverlayCenter(base, card, 30, 9)
	if !strings.Contains(out, "HELLO") {
		t.Errorf("OverlayCenter output missing card content 'HELLO'")
	}
}

func TestOverlayCenterSmallerThanCard(t *testing.T) {
	base := strings.Repeat("x", 5) + "\n" + strings.Repeat("x", 5)
	card := "VERYLONGCARD"

	out := OverlayCenter(base, card, 5, 2)
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Errorf("line count = %d, want 2", len(lines))
	}
}
