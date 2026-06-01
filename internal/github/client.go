package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jteer/prwatch/internal/logger"
)

const graphqlURL = "https://api.github.com/graphql"

// maxPages caps how many pagination rounds we do per repo to avoid unbounded fetches.
const maxPages = 10

type repoRef struct {
	owner string
	name  string
	full  string // "owner/name"
}

// Client fetches PR data from the GitHub GraphQL API.
type Client struct {
	token    string
	username string
	team     map[string]bool
	repos    []repoRef
	http     *http.Client
}

// NewClient accepts repos as "owner/name" strings (from config.ResolvedRepos).
func NewClient(token, username string, repos, team []string) *Client {
	tm := make(map[string]bool, len(team))
	for _, m := range team {
		tm[m] = true
	}

	rr := make([]repoRef, 0, len(repos))
	for _, r := range repos {
		parts := strings.SplitN(r, "/", 2)
		if len(parts) == 2 {
			rr = append(rr, repoRef{owner: parts[0], name: parts[1], full: r})
		}
	}

	return &Client{
		token:    token,
		username: username,
		team:     tm,
		repos:    rr,
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

// -- GraphQL plumbing ---------------------------------------------------------

type gqlRequest struct {
	Query string `json:"query"`
}

type gqlResponse struct {
	Data   map[string]json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (c *Client) executeQuery(ctx context.Context, query string) (map[string]json.RawMessage, error) {
	body, err := json.Marshal(gqlRequest{Query: query})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", graphqlURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gqlResp gqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, err
	}
	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("graphql: %s", gqlResp.Errors[0].Message)
	}
	return gqlResp.Data, nil
}

// -- FetchPRs with pagination --------------------------------------------------

// RepoCount returns how many repos the client watches.
func (c *Client) RepoCount() int { return len(c.repos) }

// RepoFull returns the "owner/name" for a given repo index.
func (c *Client) RepoFull(idx int) string {
	if idx < 0 || idx >= len(c.repos) {
		return ""
	}
	return c.repos[idx].full
}

// Repos returns all "owner/name" strings in watch order.
func (c *Client) Repos() []string {
	out := make([]string, len(c.repos))
	for i, r := range c.repos {
		out[i] = r.full
	}
	return out
}

// FetchBatch fetches the first page (100 PRs max) for the given repo indices
// in a single GraphQL round-trip. Returns PRs and a cursor map for repos that
// have more pages (keyed by the original repo index).
func (c *Client) FetchBatch(ctx context.Context, activeIdxs []int) (prs []PR, cursors map[int]string, err error) {
	if len(activeIdxs) == 0 {
		return nil, nil, nil
	}

	var sb strings.Builder
	sb.WriteString("query PRWatch {\n")
	for qi, repoIdx := range activeIdxs {
		r := c.repos[repoIdx]
		fmt.Fprintf(&sb, "  r%d: repository(owner: %q, name: %q) { ...PRFrag }\n", qi, r.owner, r.name)
	}
	sb.WriteString("}\n")
	fmt.Fprintf(&sb, prFragment, "")

	data, err := c.executeQuery(ctx, sb.String())
	if err != nil {
		return nil, nil, err
	}

	cursors = make(map[int]string)
	for qi, repoIdx := range activeIdxs {
		raw, ok := data[fmt.Sprintf("r%d", qi)]
		if !ok {
			continue
		}
		repoPRs, cursor, hasMore, parseErr := c.parseRepoPage(c.repos[repoIdx].full, raw)
		if parseErr != nil {
			logger.Logf("[github] parse %s: %v", c.repos[repoIdx].full, parseErr)
			continue
		}
		for i := range repoPRs {
			repoPRs[i].Urgency = c.urgency(&repoPRs[i])
		}
		prs = append(prs, repoPRs...)
		if hasMore {
			cursors[repoIdx] = cursor
		}
	}
	return prs, cursors, nil
}

// FetchPage fetches one pagination page for a single repo.
// Returns PRs and the next cursor (empty = no more pages).
func (c *Client) FetchPage(ctx context.Context, repoIdx int, cursor string) (prs []PR, next string, err error) {
	if repoIdx < 0 || repoIdx >= len(c.repos) {
		return nil, "", fmt.Errorf("repo index %d out of range", repoIdx)
	}
	data, err := c.executeQuery(ctx, c.buildSingleRepoQuery(c.repos[repoIdx], cursor))
	if err != nil {
		return nil, "", err
	}
	raw, ok := data["r0"]
	if !ok {
		return nil, "", nil
	}
	prs, next, _, err = c.parseRepoPage(c.repos[repoIdx].full, raw)
	if err != nil {
		return nil, "", err
	}
	for i := range prs {
		prs[i].Urgency = c.urgency(&prs[i])
	}
	return prs, next, nil
}

// -- Query builders -----------------------------------------------------------

const prFragment = `
fragment PRFrag on Repository {
  name
  pullRequests(states: OPEN, first: 100%s, orderBy: {field: UPDATED_AT, direction: DESC}) {
    pageInfo { hasNextPage endCursor }
    nodes {
      number title isDraft url body
      createdAt updatedAt
      author { login }
      baseRefName
      milestone { title }
      labels(first: 10) { nodes { name } }
      reviewDecision
      reviewRequests(first: 20) {
        nodes { requestedReviewer { ... on User { login } } }
      }
      reviews(last: 20, states: [APPROVED, CHANGES_REQUESTED, COMMENTED]) {
        nodes { author { login } state }
      }
      commits(last: 1) {
        totalCount
        nodes { commit {
          statusCheckRollup { state }
          checkSuites(first: 10) { nodes {
            checkRuns(first: 20) { nodes { name status conclusion } }
          } }
        } }
      }
    }
  }
}`



// buildSingleRepoQuery builds a paginated query for one repo at a given cursor.
func (c *Client) buildSingleRepoQuery(repo repoRef, cursor string) string {
	after := fmt.Sprintf(`, after: %q`, cursor)
	return fmt.Sprintf("query PRPage {\n  r0: repository(owner: %q, name: %q) { ...PRFrag }\n}\n",
		repo.owner, repo.name) +
		fmt.Sprintf(prFragment, after)
}

// -- Raw GraphQL shapes -------------------------------------------------------

type rawRepo struct {
	Name         string `json:"name"`
	PullRequests struct {
		PageInfo struct {
			HasNextPage bool   `json:"hasNextPage"`
			EndCursor   string `json:"endCursor"`
		} `json:"pageInfo"`
		Nodes []rawPR `json:"nodes"`
	} `json:"pullRequests"`
}

type rawPR struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	IsDraft   bool   `json:"isDraft"`
	URL       string `json:"url"`
	Body      string `json:"body"` // raw markdown
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Author    struct {
		Login string `json:"login"`
	} `json:"author"`
	BaseRefName    string                                   `json:"baseRefName"`
	Milestone      *struct{ Title string `json:"title"` } `json:"milestone"`
	Labels         struct{ Nodes []struct{ Name string `json:"name"` } `json:"nodes"` } `json:"labels"`
	ReviewDecision string                                   `json:"reviewDecision"`
	ReviewRequests struct {
		Nodes []struct {
			RequestedReviewer struct {
				Login string `json:"login"`
			} `json:"requestedReviewer"`
		} `json:"nodes"`
	} `json:"reviewRequests"`
	Reviews struct {
		Nodes []struct {
			Author struct{ Login string `json:"login"` } `json:"author"`
			State  string                                `json:"state"`
		} `json:"nodes"`
	} `json:"reviews"`
	Commits struct {
		TotalCount int `json:"totalCount"`
		Nodes      []struct {
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
	} `json:"commits"`
}

// -- Parsing ------------------------------------------------------------------

// parseRepoPage parses one page of PRs and returns (prs, endCursor, hasNextPage, err).
func (c *Client) parseRepoPage(full string, raw json.RawMessage) ([]PR, string, bool, error) {
	var repo rawRepo
	if err := json.Unmarshal(raw, &repo); err != nil {
		return nil, "", false, err
	}
	prs := make([]PR, 0, len(repo.PullRequests.Nodes))
	for _, node := range repo.PullRequests.Nodes {
		prs = append(prs, c.parsePR(full, node))
	}
	pi := repo.PullRequests.PageInfo
	return prs, pi.EndCursor, pi.HasNextPage, nil
}

func (c *Client) parsePR(full string, n rawPR) PR {
	created, _ := time.Parse(time.RFC3339, n.CreatedAt)
	updated, _ := time.Parse(time.RFC3339, n.UpdatedAt)

	labels := make([]string, 0, len(n.Labels.Nodes))
	for _, l := range n.Labels.Nodes {
		labels = append(labels, l.Name)
	}

	milestone := ""
	if n.Milestone != nil {
		milestone = n.Milestone.Title
	}

	latestDecision := make(map[string]string)
	for _, r := range n.Reviews.Nodes {
		latestDecision[r.Author.Login] = r.State
	}

	var rev ReviewSummary
	var reviewers []Reviewer
	seen := make(map[string]bool)

	for login, state := range latestDecision {
		seen[login] = true
		switch state {
		case "APPROVED":
			rev.Approved++
		case "CHANGES_REQUESTED":
			rev.Rejected++
		default:
			rev.Pending++
		}
		reviewers = append(reviewers, Reviewer{Login: login, Decision: state, IsTeam: c.team[login]})
	}
	for _, rr := range n.ReviewRequests.Nodes {
		login := rr.RequestedReviewer.Login
		if !seen[login] {
			seen[login] = true
			rev.Pending++
			reviewers = append(reviewers, Reviewer{Login: login, Decision: "PENDING", IsTeam: c.team[login]})
		}
	}

	ci := CIUnknown
	var checks []Check
	if len(n.Commits.Nodes) > 0 {
		commit := n.Commits.Nodes[0].Commit
		if commit.StatusCheckRollup != nil {
			switch commit.StatusCheckRollup.State {
			case "SUCCESS":
				ci = CIPass
			case "FAILURE", "ERROR":
				ci = CIFail
			default:
				ci = CIPending
			}
		}
		for _, suite := range commit.CheckSuites.Nodes {
			for _, run := range suite.CheckRuns.Nodes {
				checks = append(checks, Check{Name: run.Name, State: runState(run.Status, run.Conclusion)})
			}
		}
	}

	return PR{
		Repo:         full,
		Number:       n.Number,
		Title:        n.Title,
		Author:       n.Author.Login,
		CreatedAt:    created,
		UpdatedAt:    updated,
		URL:          n.URL,
		Body:         n.Body,
		IsDraft:      n.IsDraft,
		BaseBranch:   n.BaseRefName,
		Milestone:    milestone,
		Labels:       labels,
		Reviews:      rev,
		Reviewers:    reviewers,
		Checks:       checks,
		CI:           ci,
		Status:       c.deriveStatus(n, rev),
		CommitCount: n.Commits.TotalCount,
	}
}

func runState(status, conclusion string) CIState {
	if status != "COMPLETED" {
		return CIPending
	}
	switch conclusion {
	case "SUCCESS":
		return CIPass
	case "FAILURE", "CANCELLED", "TIMED_OUT":
		return CIFail
	default:
		return CIPending
	}
}

func (c *Client) deriveStatus(n rawPR, rev ReviewSummary) string {
	if n.IsDraft {
		return "Draft"
	}
	switch n.ReviewDecision {
	case "APPROVED":
		return "Approved"
	case "CHANGES_REQUESTED":
		return "Changes Req."
	case "REVIEW_REQUIRED":
		for _, rr := range n.ReviewRequests.Nodes {
			if rr.RequestedReviewer.Login == c.username {
				return "Review You"
			}
		}
		return "Needs Review"
	}
	if rev.Approved > 0 {
		return "Approved"
	}
	return "Needs Review"
}

func (c *Client) urgency(p *PR) Urgency {
	if p.IsDraft {
		return UrgencyDim
	}
	if p.Author == c.username && p.Reviews.Rejected > 0 {
		return UrgencyRed
	}
	for _, r := range p.Reviewers {
		if r.Login == c.username && r.Decision == "PENDING" {
			return UrgencyRed
		}
	}
	if p.Status == "Approved" {
		return UrgencyGreen
	}
	if p.CI == CIFail || time.Since(p.CreatedAt) > 5*24*time.Hour {
		return UrgencyAmber
	}
	return UrgencyNone
}
