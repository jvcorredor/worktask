package cmd

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/jvcorredor/bytheway/internal/store"
	"github.com/jvcorredor/bytheway/internal/task"
)

// completionFixture seeds the standard temp tasks dir layout used by
// the cobra command tests with two open and one closed task and returns
// the closed task's id (the rest are deterministic literals).
func completionFixture(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	openDir := filepath.Join(tmp, "worktask", "open")
	closedDir := filepath.Join(tmp, "worktask", "closed")
	for _, d := range []string{openDir, closedDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	writeTaskFile(t, openDir, task.Task{
		ID:      "open0001",
		Created: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC),
		Body:    "buy milk\nfrom the corner store\n",
	})
	writeTaskFile(t, openDir, task.Task{
		ID:      "open0002",
		Created: time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC),
		Body:    "fix the login bug\n",
	})
	writeTaskFile(t, closedDir, task.Task{
		ID:        "clsd0001",
		Created:   time.Date(2026, 4, 28, 8, 0, 0, 0, time.UTC),
		Completed: time.Date(2026, 4, 29, 11, 0, 0, 0, time.UTC),
		Body:      "ship the release\n",
	})
}

// TestCompleteFragment_filtersByState confirms the filter argument
// scopes which tasks are surfaced as completion candidates: FilterAll
// returns every task, FilterOpen drops closed ones, and FilterClosed
// drops open ones. This is what makes `close <Tab>` show only open
// tasks and `reopen <Tab>` show only closed.
func TestCompleteFragment_filtersByState(t *testing.T) {
	completionFixture(t)

	cases := []struct {
		filter store.Filter
		wantID []string
	}{
		{store.FilterAll, []string{"clsd0001", "open0001", "open0002"}},
		{store.FilterOpen, []string{"open0001", "open0002"}},
		{store.FilterClosed, []string{"clsd0001"}},
	}
	for _, c := range cases {
		results, dir := completeFragment(c.filter)(&cobra.Command{}, nil, "")
		if dir != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("filter=%v: directive=%v, want ShellCompDirectiveNoFileComp", c.filter, dir)
		}
		got := completionIDs(results)
		sort.Strings(got)
		if !equalSlices(got, c.wantID) {
			t.Errorf("filter=%v: ids=%v, want %v", c.filter, got, c.wantID)
		}
	}
}

// TestCompleteFragment_descriptions confirms that each candidate is
// rendered as "<id>\t<first-line-of-body>", which is the format fish
// (and zsh) use to show a description next to each completion in the
// menu. Bodies span multiple lines on disk, so only the first line
// must reach the shell.
func TestCompleteFragment_descriptions(t *testing.T) {
	completionFixture(t)

	results, _ := completeFragment(store.FilterAll)(&cobra.Command{}, nil, "")
	want := map[string]string{
		"open0001": "buy milk",
		"open0002": "fix the login bug",
		"clsd0001": "ship the release",
	}
	for _, r := range results {
		id, desc := splitCandidate(r)
		if w, ok := want[id]; !ok {
			t.Errorf("unexpected candidate id %q (full=%q)", id, r)
		} else if desc != w {
			t.Errorf("candidate %q: description=%q, want %q", id, desc, w)
		}
	}
	if len(results) != len(want) {
		t.Errorf("got %d candidates, want %d", len(results), len(want))
	}
}

// TestCompleteFragment_skipsAfterFirstArg checks the second-arg
// behaviour for two-argument commands like `update <fragment> <text>`:
// once a fragment has been provided, the completion function returns no
// candidates and suppresses filename fallback, so a stray <Tab> on the
// free-form text arg does not flood the user with files from cwd.
func TestCompleteFragment_skipsAfterFirstArg(t *testing.T) {
	completionFixture(t)

	results, dir := completeFragment(store.FilterAll)(&cobra.Command{}, []string{"open0001"}, "")
	if len(results) != 0 {
		t.Errorf("expected no candidates after first arg, got %v", results)
	}
	if dir != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive=%v, want ShellCompDirectiveNoFileComp", dir)
	}
}

func completionIDs(candidates []string) []string {
	out := make([]string, 0, len(candidates))
	for _, c := range candidates {
		id, _ := splitCandidate(c)
		out = append(out, id)
	}
	return out
}

func splitCandidate(s string) (id, desc string) {
	for i := 0; i < len(s); i++ {
		if s[i] == '\t' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
