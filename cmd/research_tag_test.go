package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jvcorredor/worktask/internal/task"
)

// TestValidateResearchTag covers the pure helper that normalizes the
// --tag flag value before research uses it: empty stays empty (no filter),
// uppercase is lowercased, and an invalid tag returns an error.
func TestValidateResearchTag(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "empty is a no-op", raw: "", want: ""},
		{name: "lowercase passes through", raw: "infra", want: "infra"},
		{name: "uppercase is normalized", raw: "INFRA", want: "infra"},
		{name: "hyphen is allowed", raw: "infra-core", want: "infra-core"},
		{name: "space rejected", raw: "bad tag", wantErr: true},
		{name: "punctuation rejected", raw: "bug!", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateResearchTag(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("validateResearchTag(%q) = %q, nil; want error", tc.raw, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateResearchTag(%q) returned unexpected error: %v", tc.raw, err)
			}
			if got != tc.want {
				t.Errorf("validateResearchTag(%q) = %q; want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// TestResearchCmd_tagFlagInvalidValueErrors verifies that an invalid
// --tag value (per tag.Validate) is rejected by `worktask research`
// before any work — no claude run, no writeback. Mirrors the
// list-command guarantee.
func TestResearchCmd_tagFlagInvalidValueErrors(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	openDir := filepath.Join(tmp, "worktask", "open")
	if err := os.MkdirAll(openDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	tk := task.Task{
		ID:      "11111111",
		Created: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC),
		Tags:    []string{"infra"},
		Body:    "open task\n",
	}
	raw, err := task.Encode(tk)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if err := os.WriteFile(filepath.Join(openDir, tk.ID+".md"), raw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	t.Cleanup(resetResearchFlags)

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"research", "--tag", "bad tag"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	err = rootCmd.Execute()
	if err == nil {
		t.Fatalf("research --tag 'bad tag': expected error, got nil (stdout=%s)", stdout.String())
	}
	if !strings.Contains(err.Error(), "tag") {
		t.Errorf("error should mention the tag flag; got: %v", err)
	}
}

// TestResearchCmd_tagFlagRejectedInSingleMode verifies that `worktask
// research <fragment> --tag x` errors out clearly rather than silently
// ignoring --tag — surfacing the user mistake before any agent runs.
func TestResearchCmd_tagFlagRejectedInSingleMode(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	openDir := filepath.Join(tmp, "worktask", "open")
	if err := os.MkdirAll(openDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	tk := task.Task{
		ID:      "11111111",
		Created: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC),
		Tags:    []string{"infra"},
		Body:    "open task\n",
	}
	raw, err := task.Encode(tk)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if err := os.WriteFile(filepath.Join(openDir, tk.ID+".md"), raw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	t.Cleanup(resetResearchFlags)

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"research", "11111111", "--tag", "infra"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	err = rootCmd.Execute()
	if err == nil {
		t.Fatalf("research <fragment> --tag: expected error, got nil (stdout=%s)", stdout.String())
	}
	if !strings.Contains(err.Error(), "single-task") && !strings.Contains(err.Error(), "fragment") {
		t.Errorf("error should explain that --tag conflicts with single-task mode; got: %v", err)
	}
}

// TestResearchCmd_tagFlagFiltersBatchToTaggedTasks verifies that the
// --tag value is plumbed through to the open-task listing in batch mode:
// when no open task carries the requested tag, the sweep completes with
// zero results and zero in every total — proving the filter is applied
// before the candidate partitioning. (We assert the empty-match path so
// the test does not need to spawn a real claude runner; if the filter
// were missing, both planted tasks would reach the runner and the test
// would hang or fail.)
func TestResearchCmd_tagFlagFiltersBatchToTaggedTasks(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	openDir := filepath.Join(tmp, "worktask", "open")
	if err := os.MkdirAll(openDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	planted := []task.Task{
		{ID: "11111111", Created: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC), Tags: []string{"infra"}, Body: "first\n"},
		{ID: "22222222", Created: time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC), Tags: []string{"bug"}, Body: "second\n"},
	}
	for _, tk := range planted {
		raw, err := task.Encode(tk)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		if err := os.WriteFile(filepath.Join(openDir, tk.ID+".md"), raw, 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	t.Cleanup(resetResearchFlags)

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"research", "--tag", "DOES-NOT-EXIST"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("research --tag: unexpected error: %v (stderr=%s)", err, stderr.String())
	}

	out := stdout.String()
	// No task should have been researched: the per-task results array is empty.
	if !strings.Contains(out, `"results": []`) {
		t.Errorf("expected empty results array (no task researched), got: %s", out)
	}
	// Totals must reflect zero work done.
	for _, want := range []string{
		`"findings": 0`,
		`"clarify": 0`,
		`"failed": 0`,
		`"skipped": 0`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary should contain %q; full output: %s", want, out)
		}
	}
}

func resetResearchFlags() {
	researchTag = ""
	researchAll = false
	researchStale = 0
	researchConcurrency = defaultResearchConcurrency
	researchTimeout = defaultResearchTimeout
	researchModel = ""
}
