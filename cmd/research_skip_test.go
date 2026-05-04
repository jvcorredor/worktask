package cmd

import (
	"testing"
	"time"

	"github.com/jvcorredor/bytheway/internal/task"
)

func TestPartitionResearchCandidates(t *testing.T) {
	now := time.Date(2026, 5, 1, 14, 0, 0, 0, time.UTC)

	never := task.Task{ID: "11111111", Body: "never researched\n"}
	fresh := task.Task{
		ID:             "22222222",
		Body:           "researched today\n",
		LastResearched: now.Add(-2 * time.Hour),
	}
	stale := task.Task{
		ID:             "33333333",
		Body:           "researched a month ago\n",
		LastResearched: now.AddDate(0, -1, 0),
	}

	cases := []struct {
		name           string
		flags          skipFlags
		input          []task.Task
		wantToResearch []string
		wantSkipped    []string
	}{
		{
			name:           "default skips tasks that already have last_researched",
			flags:          skipFlags{},
			input:          []task.Task{never, fresh, stale},
			wantToResearch: []string{"11111111"},
			wantSkipped:    []string{"22222222", "33333333"},
		},
		{
			name:           "--all forces every open task through, regardless of last_researched",
			flags:          skipFlags{all: true},
			input:          []task.Task{never, fresh, stale},
			wantToResearch: []string{"11111111", "22222222", "33333333"},
			wantSkipped:    nil,
		},
		{
			name:           "--stale=14d re-researches anything older than 14 days, leaves fresh ones skipped",
			flags:          skipFlags{stale: 14 * 24 * time.Hour},
			input:          []task.Task{never, fresh, stale},
			wantToResearch: []string{"11111111", "33333333"},
			wantSkipped:    []string{"22222222"},
		},
		{
			name:           "--all overrides --stale (--all wins)",
			flags:          skipFlags{all: true, stale: 14 * 24 * time.Hour},
			input:          []task.Task{never, fresh, stale},
			wantToResearch: []string{"11111111", "22222222", "33333333"},
			wantSkipped:    nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			toResearch, skipped := partitionResearchCandidates(tc.input, tc.flags, now)
			if !idsEqual(toResearch, tc.wantToResearch) {
				t.Errorf("toResearch IDs = %v; want %v", ids(toResearch), tc.wantToResearch)
			}
			if !idsEqual(skipped, tc.wantSkipped) {
				t.Errorf("skipped IDs = %v; want %v", ids(skipped), tc.wantSkipped)
			}
		})
	}
}

func ids(ts []task.Task) []string {
	out := make([]string, 0, len(ts))
	for _, t := range ts {
		out = append(out, t.ID)
	}
	return out
}

func idsEqual(got []task.Task, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i, t := range got {
		if t.ID != want[i] {
			return false
		}
	}
	return true
}
