package github

import (
	"fmt"
	"strings"
	"time"
)

// Urgency controls row color in the table.
type Urgency int

const (
	UrgencyNone  Urgency = iota // neutral
	UrgencyDim                  // draft
	UrgencyGreen                // approved / ready to merge
	UrgencyAmber                // CI failing or stale > 5d
	UrgencyRed                  // changes requested / review owed
)

// CIState is the rolled-up CI status for a PR.
type CIState int

const (
	CIUnknown CIState = iota
	CIPending
	CIPass
	CIFail
)

func (s CIState) Glyph() string {
	switch s {
	case CIPass:
		return "✓"
	case CIFail:
		return "✗"
	case CIPending:
		return "⟳"
	default:
		return "?"
	}
}

// ReviewSummary is the compact roll-up shown in the table.
type ReviewSummary struct {
	Approved int
	Rejected int
	Pending  int
}

func (r ReviewSummary) String() string {
	var parts []string
	if r.Approved > 0 {
		parts = append(parts, fmt.Sprintf("%d✓", r.Approved))
	}
	if r.Rejected > 0 {
		parts = append(parts, fmt.Sprintf("%d✗", r.Rejected))
	}
	if r.Pending > 0 {
		parts = append(parts, fmt.Sprintf("%d⏳", r.Pending))
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, " ")
}

// Reviewer is one entry in the detail pane reviewer list.
type Reviewer struct {
	Login    string
	Decision string // APPROVED, CHANGES_REQUESTED, PENDING
	IsTeam   bool
}

func (r Reviewer) Initials() string {
	parts := strings.Split(r.Login, "-")
	if len(parts) >= 2 {
		return strings.ToUpper(string([]rune(parts[0])[0:1]) + string([]rune(parts[1])[0:1]))
	}
	if len(r.Login) >= 2 {
		return strings.ToUpper(r.Login[:2])
	}
	return strings.ToUpper(r.Login)
}

// Check is a single CI check shown in the detail pane.
type Check struct {
	Name  string
	State CIState
	Note  string
}

// PR is a single pull request with all data needed for both panes.
type PR struct {
	Repo       string
	Number     int
	Title      string
	Author     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	URL        string
	Body       string
	IsDraft    bool
	BaseBranch string
	Milestone  string
	Labels     []string
	Reviews    ReviewSummary
	Reviewers  []Reviewer
	Checks     []Check
	CI         CIState
	Status     string // display string: "Needs Review", "Approved", etc.
	Urgency    Urgency
}

func (p PR) Age() string {
	d := time.Since(p.CreatedAt)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func (p PR) LabelString() string {
	return strings.Join(p.Labels, " ")
}
