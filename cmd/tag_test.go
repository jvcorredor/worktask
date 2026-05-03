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

// TestTagLsCmd_jsonModeReturnsTagsEnvelope is the end-to-end test for
// `worktask --format=json tag ls`: stdout is the {"tags": [...]} envelope
// with entries sorted alphabetically.
func TestTagLsCmd_jsonModeReturnsTagsEnvelope(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	openDir := filepath.Join(tmp, "worktask", "open")
	if err := os.MkdirAll(openDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	tasks := []task.Task{
		{ID: "11111111", Created: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC), Tags: []string{"infra", "urgent"}, Body: "first\n"},
		{ID: "22222222", Created: time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC), Tags: []string{"infra", "bug"}, Body: "second\n"},
	}
	for _, tk := range tasks {
		raw, err := task.Encode(tk)
		if err != nil {
			t.Fatalf("Encode: %v", err)
		}
		if err := os.WriteFile(filepath.Join(openDir, tk.ID+".md"), raw, 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	prevFormat := format
	format = formatJSON
	t.Cleanup(func() { format = prevFormat })

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"tag", "ls"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute tag ls: %v (stderr=%s)", err, stderr.String())
	}

	want := "{\n  \"tags\": [\n    {\n      \"name\": \"bug\",\n      \"count\": 1\n    },\n    {\n      \"name\": \"infra\",\n      \"count\": 2\n    },\n    {\n      \"name\": \"urgent\",\n      \"count\": 1\n    }\n  ]\n}\n"
	if stdout.String() != want {
		t.Errorf("tag ls JSON output mismatch.\n--- got ---\n%s\n--- want ---\n%s", stdout.String(), want)
	}
}

// TestTagLsCmd_humanModeEmitsSpaceSeparatedLine is the end-to-end test
// for `worktask tag ls`: in human mode, output is a single line of
// "name (count)" entries with no header, no ANSI escapes — the contract
// pipe-mode callers rely on for grep/awk.
func TestTagLsCmd_humanModeEmitsSpaceSeparatedLine(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	openDir := filepath.Join(tmp, "worktask", "open")
	if err := os.MkdirAll(openDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	tasks := []task.Task{
		{ID: "11111111", Created: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC), Tags: []string{"infra", "urgent"}, Body: "first\n"},
		{ID: "22222222", Created: time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC), Tags: []string{"infra", "bug"}, Body: "second\n"},
	}
	for _, tk := range tasks {
		raw, err := task.Encode(tk)
		if err != nil {
			t.Fatalf("Encode: %v", err)
		}
		if err := os.WriteFile(filepath.Join(openDir, tk.ID+".md"), raw, 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	prevFormat := format
	format = formatHuman
	t.Cleanup(func() { format = prevFormat })

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"tag", "ls"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute tag ls: %v (stderr=%s)", err, stderr.String())
	}

	want := "bug (1)  infra (2)  urgent (1)\n"
	if stdout.String() != want {
		t.Errorf("tag ls human output mismatch.\n--- got ---\n%q\n--- want ---\n%q", stdout.String(), want)
	}
	if bytes.ContainsRune(stdout.Bytes(), '\x1b') {
		t.Errorf("tag ls pipe-mode output must not contain ANSI escapes, got %q", stdout.String())
	}
}

// TestTagLsCmd_emptyCorpusEmitsBlankLineHuman pins the empty contract:
// human-mode tag ls on a corpus with no tags emits a single blank line.
func TestTagLsCmd_emptyCorpusEmitsBlankLineHuman(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	prevFormat := format
	format = formatHuman
	t.Cleanup(func() { format = prevFormat })

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"tag", "ls"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute tag ls: %v (stderr=%s)", err, stderr.String())
	}

	if stdout.String() != "\n" {
		t.Errorf("empty tag ls = %q; want %q", stdout.String(), "\n")
	}
}

// TestTagLsCmd_closedFlagFiltersToClosedTagsOnly verifies that --closed
// scopes the scan to closed tasks only — open-only tags do not appear.
func TestTagLsCmd_closedFlagFiltersToClosedTagsOnly(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	openDir := filepath.Join(tmp, "worktask", "open")
	closedDir := filepath.Join(tmp, "worktask", "closed")
	if err := os.MkdirAll(openDir, 0o755); err != nil {
		t.Fatalf("mkdir open: %v", err)
	}
	if err := os.MkdirAll(closedDir, 0o755); err != nil {
		t.Fatalf("mkdir closed: %v", err)
	}

	openTask := task.Task{ID: "11111111", Created: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC), Tags: []string{"open-only"}, Body: "open\n"}
	openRaw, err := task.Encode(openTask)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if err := os.WriteFile(filepath.Join(openDir, "open.md"), openRaw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	closedTask := task.Task{ID: "22222222", Created: time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC), Completed: time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC), Tags: []string{"closed-only"}, Body: "closed\n"}
	closedRaw, err := task.Encode(closedTask)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if err := os.WriteFile(filepath.Join(closedDir, "closed.md"), closedRaw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	prevFormat := format
	format = formatHuman
	t.Cleanup(func() {
		format = prevFormat
		tagLsClosed = false
		tagLsAll = false
	})

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"tag", "ls", "--closed"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute tag ls --closed: %v (stderr=%s)", err, stderr.String())
	}

	got := strings.TrimRight(stdout.String(), "\n")
	if !strings.Contains(got, "closed-only") {
		t.Errorf("--closed output should contain closed-only tag; got %q", got)
	}
	if strings.Contains(got, "open-only") {
		t.Errorf("--closed output must not contain open-only tag; got %q", got)
	}
}

// TestTagLsCmd_allFlagIncludesOpenAndClosed verifies that --all scans
// both open and closed tasks.
func TestTagLsCmd_allFlagIncludesOpenAndClosed(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	openDir := filepath.Join(tmp, "worktask", "open")
	closedDir := filepath.Join(tmp, "worktask", "closed")
	if err := os.MkdirAll(openDir, 0o755); err != nil {
		t.Fatalf("mkdir open: %v", err)
	}
	if err := os.MkdirAll(closedDir, 0o755); err != nil {
		t.Fatalf("mkdir closed: %v", err)
	}

	openTask := task.Task{ID: "11111111", Created: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC), Tags: []string{"open-only"}, Body: "open\n"}
	openRaw, _ := task.Encode(openTask)
	if err := os.WriteFile(filepath.Join(openDir, "open.md"), openRaw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	closedTask := task.Task{ID: "22222222", Created: time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC), Completed: time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC), Tags: []string{"closed-only"}, Body: "closed\n"}
	closedRaw, _ := task.Encode(closedTask)
	if err := os.WriteFile(filepath.Join(closedDir, "closed.md"), closedRaw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	prevFormat := format
	format = formatHuman
	t.Cleanup(func() {
		format = prevFormat
		tagLsClosed = false
		tagLsAll = false
	})

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"tag", "ls", "--all"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute tag ls --all: %v (stderr=%s)", err, stderr.String())
	}

	got := strings.TrimRight(stdout.String(), "\n")
	if !strings.Contains(got, "open-only") {
		t.Errorf("--all output should contain open-only tag; got %q", got)
	}
	if !strings.Contains(got, "closed-only") {
		t.Errorf("--all output should contain closed-only tag; got %q", got)
	}
}

// TestTagRmCmd_removesTagFromOpenTaskAndPrintsRemovedMessage is the
// end-to-end test for `worktask tag rm <fragment> <tag>`.
func TestTagRmCmd_removesTagFromOpenTaskAndPrintsRemovedMessage(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	openDir := filepath.Join(tmp, "worktask", "open")
	if err := os.MkdirAll(openDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	tk := task.Task{
		ID:      "abcdef12",
		Created: time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC),
		Tags:    []string{"urgent", "shopping"},
		Body:    "buy milk\n",
	}
	raw, err := task.Encode(tk)
	if err != nil {
		t.Fatalf("task.Encode: %v", err)
	}
	openPath := filepath.Join(openDir, "2026-04-29T11-30_abcdef12_buy-milk.md")
	if err := os.WriteFile(openPath, raw, 0o644); err != nil {
		t.Fatalf("write task file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"tag", "rm", "abcdef12", "urgent"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute tag rm: %v (stderr=%s)", err, stderr.String())
	}

	if got, want := strings.TrimRight(stdout.String(), "\n"), "removed urgent from abcdef12"; got != want {
		t.Errorf("tag rm stdout = %q; want %q", got, want)
	}

	data, err := os.ReadFile(openPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "---\nid: abcdef12\ncreated: 2026-04-29T11:30:00Z\ntags: [shopping]\n---\nbuy milk\n"
	if string(data) != want {
		t.Errorf("file contents:\n--- got ---\n%s\n--- want ---\n%s", data, want)
	}
}

// TestTagAddCmd_addsTagToOpenTaskAndPrintsTaggedID is the end-to-end
// test for `worktask tag add <fragment> <tag>`.
func TestTagAddCmd_addsTagToOpenTaskAndPrintsTaggedID(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	openDir := filepath.Join(tmp, "worktask", "open")
	if err := os.MkdirAll(openDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	tk := task.Task{
		ID:      "abcdef12",
		Created: time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC),
		Body:    "buy milk\n",
	}
	raw, err := task.Encode(tk)
	if err != nil {
		t.Fatalf("task.Encode: %v", err)
	}
	openPath := filepath.Join(openDir, "2026-04-29T11-30_abcdef12_buy-milk.md")
	if err := os.WriteFile(openPath, raw, 0o644); err != nil {
		t.Fatalf("write task file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"tag", "add", "abcdef12", "urgent"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute tag add: %v (stderr=%s)", err, stderr.String())
	}

	if got, want := strings.TrimRight(stdout.String(), "\n"), "tagged abcdef12 with urgent"; got != want {
		t.Errorf("tag add stdout = %q; want %q", got, want)
	}

	data, err := os.ReadFile(openPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "---\nid: abcdef12\ncreated: 2026-04-29T11:30:00Z\ntags: [urgent]\n---\nbuy milk\n"
	if string(data) != want {
		t.Errorf("file contents:\n--- got ---\n%s\n--- want ---\n%s", data, want)
	}
}
