package ui

import (
	"testing"
	"time"

	"github.com/jteer/prwatch/internal/config"
	gh "github.com/jteer/prwatch/internal/github"
)

// -- changesFor ---------------------------------------------------------------

func TestChangesFor(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	later := base.Add(time.Hour)

	tests := []struct {
		name    string
		pr      gh.PR
		prev    prSnapshot
		want    []string
	}{
		{
			name: "nothing changed",
			pr: gh.PR{
				CommitCount: 3,
				CI:          gh.CIPass,
				Reviews:     gh.ReviewSummary{Approved: 1},
				UpdatedAt:   base,
			},
			prev: prSnapshot{
				commitCount: 3,
				ci:          gh.CIPass,
				reviews:     gh.ReviewSummary{Approved: 1},
				updatedAt:   base,
			},
			want: nil,
		},
		{
			name: "new commits",
			pr:   gh.PR{CommitCount: 4, UpdatedAt: later},
			prev: prSnapshot{commitCount: 3, updatedAt: base},
			want: []string{"* new commits"},
		},
		{
			name: "CI changed to pass",
			pr:   gh.PR{CI: gh.CIPass, UpdatedAt: later},
			prev: prSnapshot{ci: gh.CIPending, updatedAt: base},
			want: []string{"* CI updated"},
		},
		{
			name: "CI changed to unknown — no CI notification (same updatedAt)",
			pr:   gh.PR{CI: gh.CIUnknown, UpdatedAt: base},
			prev: prSnapshot{ci: gh.CIPass, updatedAt: base},
			want: nil,
		},
		{
			name: "new approval",
			pr:   gh.PR{Reviews: gh.ReviewSummary{Approved: 2}, UpdatedAt: later},
			prev: prSnapshot{reviews: gh.ReviewSummary{Approved: 1}, updatedAt: base},
			want: []string{"* new reviews"},
		},
		{
			name: "new rejection",
			pr:   gh.PR{Reviews: gh.ReviewSummary{Rejected: 1}, UpdatedAt: later},
			prev: prSnapshot{reviews: gh.ReviewSummary{Rejected: 0}, updatedAt: base},
			want: []string{"* new reviews"},
		},
		{
			name: "updatedAt newer, no specific changes — new activity",
			pr:   gh.PR{CommitCount: 2, CI: gh.CIPass, UpdatedAt: later},
			prev: prSnapshot{commitCount: 2, ci: gh.CIPass, updatedAt: base},
			want: []string{"* new activity"},
		},
		{
			name: "updatedAt older — no activity notification",
			pr:   gh.PR{CommitCount: 2, CI: gh.CIPass, UpdatedAt: base},
			prev: prSnapshot{commitCount: 2, ci: gh.CIPass, updatedAt: later},
			want: nil,
		},
		{
			name: "multiple specific changes",
			pr:   gh.PR{CommitCount: 5, CI: gh.CIFail, UpdatedAt: later},
			prev: prSnapshot{commitCount: 3, ci: gh.CIPass, updatedAt: base},
			want: []string{"* new commits", "* CI updated"},
		},
		{
			name: "specific change present — no fallback activity appended",
			pr:   gh.PR{CommitCount: 5, CI: gh.CIPass, UpdatedAt: later},
			prev: prSnapshot{commitCount: 3, ci: gh.CIPass, updatedAt: base},
			want: []string{"* new commits"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := changesFor(tc.pr, tc.prev)
			if len(got) != len(tc.want) {
				t.Fatalf("changesFor() = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// -- buildNotifyCmds ----------------------------------------------------------

func TestBuildNotifyCmds(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	later := base.Add(time.Hour)

	changedPR := gh.PR{
		Repo: "owner/repo", Number: 1,
		CommitCount: 5, UpdatedAt: later,
	}
	unchangedPR := gh.PR{
		Repo: "owner/repo", Number: 2,
		CommitCount: 3, UpdatedAt: base,
	}
	newPR := gh.PR{
		Repo: "owner/repo", Number: 3,
		CommitCount: 1, UpdatedAt: later,
	}

	snap := map[string]prSnapshot{
		"owner/repo#1": {commitCount: 3, updatedAt: base},
		"owner/repo#2": {commitCount: 3, updatedAt: base},
		// #3 absent — new PR
	}

	tests := []struct {
		name          string
		notifications bool
		allPRs        []gh.PR
		prevPRs       map[string]prSnapshot
		wantCmds      int
	}{
		{
			name:          "notifications disabled — no cmds",
			notifications: false,
			allPRs:        []gh.PR{changedPR},
			prevPRs:       snap,
			wantCmds:      0,
		},
		{
			name:          "changed PR → one cmd",
			notifications: true,
			allPRs:        []gh.PR{changedPR},
			prevPRs:       snap,
			wantCmds:      1,
		},
		{
			name:          "unchanged PR → no cmd",
			notifications: true,
			allPRs:        []gh.PR{unchangedPR},
			prevPRs:       snap,
			wantCmds:      0,
		},
		{
			name:          "new PR not in snapshot → no cmd",
			notifications: true,
			allPRs:        []gh.PR{newPR},
			prevPRs:       snap,
			wantCmds:      0,
		},
		{
			name:          "mixed — only changed PR fires",
			notifications: true,
			allPRs:        []gh.PR{changedPR, unchangedPR, newPR},
			prevPRs:       snap,
			wantCmds:      1,
		},
		{
			name:          "empty snapshot — no cmds",
			notifications: true,
			allPRs:        []gh.PR{changedPR},
			prevPRs:       map[string]prSnapshot{},
			wantCmds:      0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &AppModel{
				cfg:     &config.Config{Notifications: tc.notifications},
				allPRs:  tc.allPRs,
				prevPRs: tc.prevPRs,
			}
			cmds := m.buildNotifyCmds()
			if len(cmds) != tc.wantCmds {
				t.Errorf("buildNotifyCmds() returned %d cmds, want %d", len(cmds), tc.wantCmds)
			}
		})
	}
}

// -- updateSnapshot -----------------------------------------------------------

func TestUpdateSnapshot(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	m := &AppModel{
		prevPRs: make(map[string]prSnapshot),
		allPRs: []gh.PR{
			{
				Repo:        "owner/repo",
				Number:      1,
				CommitCount: 7,
				CI:          gh.CIPass,
				Reviews:     gh.ReviewSummary{Approved: 2, Rejected: 1},
				UpdatedAt:   base,
			},
			{
				Repo:        "other/repo",
				Number:      99,
				CommitCount: 0,
				CI:          gh.CIUnknown,
				UpdatedAt:   base.Add(time.Minute),
			},
		},
	}

	m.updateSnapshot()

	if len(m.prevPRs) != 2 {
		t.Fatalf("snapshot has %d entries, want 2", len(m.prevPRs))
	}

	s1 := m.prevPRs["owner/repo#1"]
	if s1.commitCount != 7 {
		t.Errorf("commitCount = %d, want 7", s1.commitCount)
	}
	if s1.ci != gh.CIPass {
		t.Errorf("ci = %v, want CIPass", s1.ci)
	}
	if s1.reviews.Approved != 2 || s1.reviews.Rejected != 1 {
		t.Errorf("reviews = %+v, want {Approved:2 Rejected:1}", s1.reviews)
	}
	if !s1.updatedAt.Equal(base) {
		t.Errorf("updatedAt = %v, want %v", s1.updatedAt, base)
	}

	s2 := m.prevPRs["other/repo#99"]
	if s2.commitCount != 0 {
		t.Errorf("commitCount = %d, want 0", s2.commitCount)
	}
}

// -- refreshComplete ----------------------------------------------------------

func TestRefreshComplete(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	later := base.Add(time.Hour)

	pr := gh.PR{
		Repo: "owner/repo", Number: 1,
		CommitCount: 5, UpdatedAt: later,
	}

	t.Run("first load — no notifications, snapshot built", func(t *testing.T) {
		m := &AppModel{
			cfg:         &config.Config{Notifications: true},
			prevPRs:     make(map[string]prSnapshot),
			hasSnapshot: false,
			allPRs:      []gh.PR{pr},
		}
		cmds := m.refreshComplete()
		if len(cmds) != 0 {
			t.Errorf("first load: got %d cmds, want 0", len(cmds))
		}
		if !m.hasSnapshot {
			t.Error("hasSnapshot not set after first load")
		}
		if _, ok := m.prevPRs["owner/repo#1"]; !ok {
			t.Error("snapshot not populated after first load")
		}
	})

	t.Run("subsequent load — changed PR fires notification", func(t *testing.T) {
		m := &AppModel{
			cfg: &config.Config{Notifications: true},
			prevPRs: map[string]prSnapshot{
				"owner/repo#1": {commitCount: 3, updatedAt: base},
			},
			hasSnapshot: true,
			allPRs:      []gh.PR{pr},
		}
		cmds := m.refreshComplete()
		if len(cmds) != 1 {
			t.Errorf("subsequent load: got %d cmds, want 1", len(cmds))
		}
		snap := m.prevPRs["owner/repo#1"]
		if snap.commitCount != 5 {
			t.Errorf("snapshot not updated: commitCount = %d, want 5", snap.commitCount)
		}
	})

	t.Run("subsequent load — notifications off, no cmds", func(t *testing.T) {
		m := &AppModel{
			cfg: &config.Config{Notifications: false},
			prevPRs: map[string]prSnapshot{
				"owner/repo#1": {commitCount: 3, updatedAt: base},
			},
			hasSnapshot: true,
			allPRs:      []gh.PR{pr},
		}
		cmds := m.refreshComplete()
		if len(cmds) != 0 {
			t.Errorf("notifications off: got %d cmds, want 0", len(cmds))
		}
	})
}

// -- prKey --------------------------------------------------------------------

func TestPrKey(t *testing.T) {
	pr := gh.PR{Repo: "golang/go", Number: 42}
	if got := prKey(pr); got != "golang/go#42" {
		t.Errorf("prKey() = %q, want %q", got, "golang/go#42")
	}
}
