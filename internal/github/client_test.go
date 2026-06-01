package github

import (
	"encoding/json"
	"testing"
	"time"
)

func newTestClient(username string, team []string) *Client {
	return NewClient("tok", username, []string{"owner/repo"}, team)
}

// -- runState -----------------------------------------------------------------

func TestRunState(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		conclusion string
		want       CIState
	}{
		{"in progress", "IN_PROGRESS", "", CIPending},
		{"queued", "QUEUED", "", CIPending},
		{"completed success", "COMPLETED", "SUCCESS", CIPass},
		{"completed failure", "COMPLETED", "FAILURE", CIFail},
		{"completed cancelled", "COMPLETED", "CANCELLED", CIFail},
		{"completed timed out", "COMPLETED", "TIMED_OUT", CIFail},
		{"completed unknown conclusion", "COMPLETED", "SKIPPED", CIPending},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := runState(tc.status, tc.conclusion); got != tc.want {
				t.Errorf("runState(%q, %q) = %v, want %v", tc.status, tc.conclusion, got, tc.want)
			}
		})
	}
}

// -- deriveStatus -------------------------------------------------------------

func TestDeriveStatus(t *testing.T) {
	c := newTestClient("alice", nil)

	tests := []struct {
		name string
		n    rawPR
		rev  ReviewSummary
		want string
	}{
		{
			name: "draft",
			n:    rawPR{IsDraft: true},
			want: "Draft",
		},
		{
			name: "approved decision",
			n:    rawPR{ReviewDecision: "APPROVED"},
			want: "Approved",
		},
		{
			name: "changes requested decision",
			n:    rawPR{ReviewDecision: "CHANGES_REQUESTED"},
			want: "Changes Req.",
		},
		{
			name: "review required — reviewer is current user",
			n: rawPR{
				ReviewDecision: "REVIEW_REQUIRED",
				ReviewRequests: struct {
					Nodes []struct {
						RequestedReviewer struct {
							Login string `json:"login"`
						} `json:"requestedReviewer"`
					} `json:"nodes"`
				}{
					Nodes: []struct {
						RequestedReviewer struct {
							Login string `json:"login"`
						} `json:"requestedReviewer"`
					}{{RequestedReviewer: struct{ Login string `json:"login"` }{Login: "alice"}}},
				},
			},
			want: "Review You",
		},
		{
			name: "review required — other reviewer",
			n: rawPR{
				ReviewDecision: "REVIEW_REQUIRED",
				ReviewRequests: struct {
					Nodes []struct {
						RequestedReviewer struct {
							Login string `json:"login"`
						} `json:"requestedReviewer"`
					} `json:"nodes"`
				}{
					Nodes: []struct {
						RequestedReviewer struct {
							Login string `json:"login"`
						} `json:"requestedReviewer"`
					}{{RequestedReviewer: struct{ Login string `json:"login"` }{Login: "bob"}}},
				},
			},
			want: "Needs Review",
		},
		{
			name: "no decision but approved count > 0",
			rev:  ReviewSummary{Approved: 1},
			want: "Approved",
		},
		{
			name: "no decision no approvals",
			want: "Needs Review",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := c.deriveStatus(tc.n, tc.rev); got != tc.want {
				t.Errorf("deriveStatus() = %q, want %q", got, tc.want)
			}
		})
	}
}

// -- urgency ------------------------------------------------------------------

func TestUrgency(t *testing.T) {
	c := newTestClient("alice", nil)
	now := time.Now()

	tests := []struct {
		name string
		pr   PR
		want Urgency
	}{
		{
			name: "draft → dim",
			pr:   PR{IsDraft: true},
			want: UrgencyDim,
		},
		{
			name: "author owns PR with rejection → red",
			pr: PR{
				Author:  "alice",
				Reviews: ReviewSummary{Rejected: 1},
			},
			want: UrgencyRed,
		},
		{
			name: "alice is pending reviewer → red",
			pr: PR{
				Reviewers: []Reviewer{{Login: "alice", Decision: "PENDING"}},
			},
			want: UrgencyRed,
		},
		{
			name: "approved → green",
			pr:   PR{Status: "Approved"},
			want: UrgencyGreen,
		},
		{
			name: "CI fail → amber",
			pr:   PR{CI: CIFail, CreatedAt: now},
			want: UrgencyAmber,
		},
		{
			name: "stale (>5d) → amber",
			pr:   PR{CreatedAt: now.Add(-6 * 24 * time.Hour)},
			want: UrgencyAmber,
		},
		{
			name: "normal PR → none",
			pr:   PR{CreatedAt: now, CI: CIPass},
			want: UrgencyNone,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := c.urgency(&tc.pr); got != tc.want {
				t.Errorf("urgency() = %v, want %v", got, tc.want)
			}
		})
	}
}

// -- parseRepoPage ------------------------------------------------------------

func TestParseRepoPage(t *testing.T) {
	c := newTestClient("alice", []string{"alice"})

	raw := json.RawMessage(`{
		"name": "repo",
		"pullRequests": {
			"pageInfo": {"hasNextPage": false, "endCursor": ""},
			"nodes": [
				{
					"number": 42,
					"title": "Test PR",
					"isDraft": false,
					"url": "https://github.com/owner/repo/pull/42",
					"body": "description",
					"createdAt": "2024-01-01T00:00:00Z",
					"updatedAt": "2024-01-02T00:00:00Z",
					"author": {"login": "bob"},
					"baseRefName": "main",
					"milestone": null,
					"labels": {"nodes": [{"name": "bug"}]},
					"reviewDecision": "APPROVED",
					"reviewRequests": {"nodes": []},
					"reviews": {"nodes": [{"author": {"login": "alice"}, "state": "APPROVED"}]},
					"commits": {"nodes": []}
				}
			]
		}
	}`)

	prs, cursor, hasMore, err := c.parseRepoPage("owner/repo", raw)
	if err != nil {
		t.Fatalf("parseRepoPage error: %v", err)
	}
	if hasMore {
		t.Error("hasMore = true, want false")
	}
	if cursor != "" {
		t.Errorf("cursor = %q, want empty", cursor)
	}
	if len(prs) != 1 {
		t.Fatalf("len(prs) = %d, want 1", len(prs))
	}

	pr := prs[0]
	if pr.Number != 42 {
		t.Errorf("Number = %d, want 42", pr.Number)
	}
	if pr.Title != "Test PR" {
		t.Errorf("Title = %q, want %q", pr.Title, "Test PR")
	}
	if pr.Author != "bob" {
		t.Errorf("Author = %q, want %q", pr.Author, "bob")
	}
	if pr.Repo != "owner/repo" {
		t.Errorf("Repo = %q, want %q", pr.Repo, "owner/repo")
	}
	if len(pr.Labels) != 1 || pr.Labels[0] != "bug" {
		t.Errorf("Labels = %v, want [bug]", pr.Labels)
	}
	if pr.Reviews.Approved != 1 {
		t.Errorf("Reviews.Approved = %d, want 1", pr.Reviews.Approved)
	}
}

func TestParseRepoPagePagination(t *testing.T) {
	c := newTestClient("alice", nil)

	raw := json.RawMessage(`{
		"name": "repo",
		"pullRequests": {
			"pageInfo": {"hasNextPage": true, "endCursor": "cursor123"},
			"nodes": []
		}
	}`)

	_, cursor, hasMore, err := c.parseRepoPage("owner/repo", raw)
	if err != nil {
		t.Fatalf("parseRepoPage error: %v", err)
	}
	if !hasMore {
		t.Error("hasMore = false, want true")
	}
	if cursor != "cursor123" {
		t.Errorf("cursor = %q, want %q", cursor, "cursor123")
	}
}

func TestParseRepoPageInvalidJSON(t *testing.T) {
	c := newTestClient("alice", nil)
	_, _, _, err := c.parseRepoPage("owner/repo", json.RawMessage(`{bad json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// -- NewClient / repo helpers -------------------------------------------------

func TestNewClient(t *testing.T) {
	repos := []string{"owner/repo1", "invalid", "org/repo2"}
	c := NewClient("tok", "alice", repos, []string{"alice", "bob"})

	if c.RepoCount() != 2 {
		t.Errorf("RepoCount() = %d, want 2 (invalid repo skipped)", c.RepoCount())
	}
	if c.RepoFull(0) != "owner/repo1" {
		t.Errorf("RepoFull(0) = %q, want %q", c.RepoFull(0), "owner/repo1")
	}
	if c.RepoFull(1) != "org/repo2" {
		t.Errorf("RepoFull(1) = %q, want %q", c.RepoFull(1), "org/repo2")
	}
	if c.RepoFull(-1) != "" {
		t.Errorf("RepoFull(-1) should return empty")
	}
	if c.RepoFull(99) != "" {
		t.Errorf("RepoFull(99) should return empty")
	}

	got := c.Repos()
	if len(got) != 2 {
		t.Fatalf("Repos() len = %d, want 2", len(got))
	}
}

// -- CI rollup in parsePR -----------------------------------------------------

func TestParsePRCIState(t *testing.T) {
	c := newTestClient("alice", nil)

	tests := []struct {
		name     string
		rollup   string
		wantCI   CIState
	}{
		{"success", "SUCCESS", CIPass},
		{"failure", "FAILURE", CIFail},
		{"error", "ERROR", CIFail},
		{"pending", "PENDING", CIPending},
		{"expected", "EXPECTED", CIPending},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n := rawPR{
				CreatedAt: "2024-01-01T00:00:00Z",
				UpdatedAt: "2024-01-01T00:00:00Z",
				Commits: struct {
					Nodes []struct {
						Commit struct {
							StatusCheckRollup *struct{ State string `json:"state"` } `json:"statusCheckRollup"`
							CheckSuites       struct {
								Nodes []struct {
									CheckRuns struct {
										Nodes []struct {
											Name       string `json:"name"`
											Status     string `json:"status"`
											Conclusion string `json:"conclusion"`
										} `json:"nodes"`
									} `json:"checkRuns"`
								} `json:"nodes"`
							} `json:"checkSuites"`
						} `json:"commit"`
					} `json:"nodes"`
				}{
					Nodes: []struct {
						Commit struct {
							StatusCheckRollup *struct{ State string `json:"state"` } `json:"statusCheckRollup"`
							CheckSuites       struct {
								Nodes []struct {
									CheckRuns struct {
										Nodes []struct {
											Name       string `json:"name"`
											Status     string `json:"status"`
											Conclusion string `json:"conclusion"`
										} `json:"nodes"`
									} `json:"checkRuns"`
								} `json:"nodes"`
							} `json:"checkSuites"`
						} `json:"commit"`
					}{{Commit: struct {
						StatusCheckRollup *struct{ State string `json:"state"` } `json:"statusCheckRollup"`
						CheckSuites       struct {
							Nodes []struct {
								CheckRuns struct {
									Nodes []struct {
										Name       string `json:"name"`
										Status     string `json:"status"`
										Conclusion string `json:"conclusion"`
									} `json:"nodes"`
								} `json:"checkRuns"`
							} `json:"nodes"`
						} `json:"checkSuites"`
					}{StatusCheckRollup: &struct{ State string `json:"state"` }{State: tc.rollup}}}},
				},
			}

			pr := c.parsePR("owner/repo", n)
			if pr.CI != tc.wantCI {
				t.Errorf("CI = %v, want %v", pr.CI, tc.wantCI)
			}
		})
	}
}
