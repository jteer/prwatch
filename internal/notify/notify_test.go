package notify

import "testing"

func TestEscAS(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello world", "hello world"},
		{`say "hello"`, `say \"hello\"`},
		{`path\to\file`, `path\\to\\file`},
		{`"quoted" and \slashed\`, `\"quoted\" and \\slashed\\`},
		{"no specials", "no specials"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := escAS(tc.input); got != tc.want {
			t.Errorf("escAS(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
