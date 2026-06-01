package github

import (
	"testing"
	"time"
)

func TestCIStateGlyph(t *testing.T) {
	tests := []struct {
		state CIState
		want  string
	}{
		{CIPass, "✓"},
		{CIFail, "✗"},
		{CIPending, "⟳"},
		{CIUnknown, "?"},
	}

	for _, tc := range tests {
		if got := tc.state.Glyph(); got != tc.want {
			t.Errorf("CIState(%d).Glyph() = %q, want %q", tc.state, got, tc.want)
		}
	}
}

func TestReviewSummaryString(t *testing.T) {
	tests := []struct {
		name string
		rev  ReviewSummary
		want string
	}{
		{"all zero", ReviewSummary{}, "—"},
		{"approved only", ReviewSummary{Approved: 2}, "2✓"},
		{"rejected only", ReviewSummary{Rejected: 1}, "1✗"},
		{"pending only", ReviewSummary{Pending: 3}, "3⏳"},
		{"all set", ReviewSummary{Approved: 1, Rejected: 1, Pending: 1}, "1✓ 1✗ 1⏳"},
		{"approved and pending", ReviewSummary{Approved: 2, Pending: 1}, "2✓ 1⏳"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.rev.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReviewerInitials(t *testing.T) {
	tests := []struct {
		login string
		want  string
	}{
		{"alice-bob", "AB"},
		{"jared-teer", "JT"},
		{"singlename", "SI"},
		{"ab", "AB"},
		{"a", "A"},
		{"multi-word-name", "MW"},
	}

	for _, tc := range tests {
		t.Run(tc.login, func(t *testing.T) {
			r := Reviewer{Login: tc.login}
			if got := r.Initials(); got != tc.want {
				t.Errorf("Initials() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPRAge(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		createdAt time.Time
		want      string
	}{
		{"minutes ago", now.Add(-30 * time.Minute), "30m"},
		{"hours ago", now.Add(-5 * time.Hour), "5h"},
		{"days ago", now.Add(-3 * 24 * time.Hour), "3d"},
		{"just now (0m)", now.Add(-30 * time.Second), "0m"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := PR{CreatedAt: tc.createdAt}
			if got := p.Age(); got != tc.want {
				t.Errorf("Age() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPRLabelString(t *testing.T) {
	tests := []struct {
		labels []string
		want   string
	}{
		{nil, ""},
		{[]string{"bug"}, "bug"},
		{[]string{"bug", "enhancement"}, "bug enhancement"},
	}

	for _, tc := range tests {
		p := PR{Labels: tc.labels}
		if got := p.LabelString(); got != tc.want {
			t.Errorf("LabelString() = %q, want %q", got, tc.want)
		}
	}
}
