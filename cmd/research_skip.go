package cmd

import (
	"time"

	"github.com/jvcorredor/bytheway/internal/task"
)

// skipFlags is the subset of `btw research` flags that decide which
// tasks make it into the batch.
type skipFlags struct {
	all   bool
	stale time.Duration
}

// partitionResearchCandidates divides open tasks into ones that should be
// researched on this run vs. ones that are skipped because their
// last_researched is recent enough.
//
// Rules (acceptance criteria for the batch sweep):
//   - default: skip any task with last_researched set
//   - --all:   research every task regardless of last_researched
//   - --stale=DUR: re-research tasks whose last_researched is older than DUR;
//     younger ones are skipped
//   - --all overrides --stale
func partitionResearchCandidates(open []task.Task, flags skipFlags, now time.Time) (toResearch, skipped []task.Task) {
	for _, t := range open {
		switch {
		case flags.all:
			toResearch = append(toResearch, t)
		case t.LastResearched.IsZero():
			toResearch = append(toResearch, t)
		case flags.stale > 0 && now.Sub(t.LastResearched) > flags.stale:
			toResearch = append(toResearch, t)
		default:
			skipped = append(skipped, t)
		}
	}
	return toResearch, skipped
}
