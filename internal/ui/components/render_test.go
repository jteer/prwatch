package components

import "testing"

func TestTrunc(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		n     int
		want  string
	}{
		{"empty string", "", 5, ""},
		{"n zero", "hello", 0, ""},
		{"n negative", "hello", -1, ""},
		{"fits exactly", "hello", 5, "hello"},
		{"fits under limit", "hi", 5, "hi"},
		{"truncated with ellipsis", "hello world", 6, "hello…"},
		{"n=1 returns single rune", "hello", 1, "h"},
		{"unicode multibyte — fits", "héllo", 5, "héllo"},
		{"unicode multibyte — truncated", "héllo world", 4, "hél…"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := trunc(tc.s, tc.n); got != tc.want {
				t.Errorf("trunc(%q, %d) = %q, want %q", tc.s, tc.n, got, tc.want)
			}
		})
	}
}
