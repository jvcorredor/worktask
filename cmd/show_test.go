package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jvcorredor/bytheway/internal/render"
	"github.com/jvcorredor/bytheway/internal/task"
)

// writeTaskFile encodes tk and writes it under openDir; tests rely on
// task.Decode reading the id from frontmatter, not the filename.
func writeTaskFile(t *testing.T, openDir string, tk task.Task) {
	t.Helper()
	raw, err := task.Encode(tk)
	if err != nil {
		t.Fatalf("task.Encode: %v", err)
	}
	if err := os.WriteFile(filepath.Join(openDir, tk.ID+".md"), raw, 0o644); err != nil {
		t.Fatalf("write task file: %v", err)
	}
}

// TestShowCmd_humanPipeModePassesRawBytes is the end-to-end pipe-mode test:
// when stdout is a non-TTY writer (here, *bytes.Buffer), `btw show
// <frag>` must emit the raw markdown file (frontmatter included) byte-for-byte
// — the contract callers rely on for piping into editors and other tooling.
func TestShowCmd_humanPipeModePassesRawBytes(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	tk := task.Task{
		ID:      "abcdef12",
		Created: time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC),
		Body:    "buy milk\nremember the brand\n",
	}
	raw, err := task.Encode(tk)
	if err != nil {
		t.Fatalf("task.Encode: %v", err)
	}
	openDir := filepath.Join(tmp, "worktask", "open")
	if err := os.MkdirAll(openDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(openDir, "abcdef12-buy-milk.md"), raw, 0o644); err != nil {
		t.Fatalf("write task file: %v", err)
	}

	prevFormat := format
	format = formatHuman
	t.Cleanup(func() { format = prevFormat })

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"show", "abcdef12"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute show: %v (stderr=%s)", err, stderr.String())
	}
	if !bytes.Equal(stdout.Bytes(), raw) {
		t.Errorf("show pipe-mode output does not match raw file bytes.\n--- got ---\n%s\n--- want ---\n%s", stdout.Bytes(), raw)
	}
}

// TestShowCmd_humanAmbiguousPipeModeIsPlain is the end-to-end pipe-mode
// test for the ambiguous-fragment error block: when stderr is a non-TTY
// writer (here, *bytes.Buffer), the candidate listing must be byte-
// identical to today's `  <id>  <description>\n` rendering with no ANSI
// escapes, and the leading prose line must be unchanged.
func TestShowCmd_humanAmbiguousPipeModeIsPlain(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	openDir := filepath.Join(tmp, "worktask", "open")
	if err := os.MkdirAll(openDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeTaskFile(t, openDir, task.Task{
		ID:      "11111111",
		Created: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC),
		Body:    "buy milk\n",
	})
	writeTaskFile(t, openDir, task.Task{
		ID:      "22222222",
		Created: time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC),
		Body:    "buy milkshake\n",
	})

	prevFormat := format
	format = formatHuman
	t.Cleanup(func() { format = prevFormat })

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"show", "milk"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	err := rootCmd.Execute()
	if !errors.Is(err, errExit) {
		t.Fatalf("expected errExit, got %v", err)
	}

	want := "ambiguous fragment \"milk\". matches:\n" +
		"  11111111  buy milk\n" +
		"  22222222  buy milkshake\n"

	if stderr.String() != want {
		t.Errorf("ambiguous pipe-mode stderr does not match plain rendering.\n--- got ---\n%s\n--- want ---\n%s", stderr.String(), want)
	}
	if bytes.ContainsRune(stderr.Bytes(), '\x1b') {
		t.Errorf("ambiguous pipe-mode stderr must not contain ANSI escapes, got %q", stderr.String())
	}
}

// TestShowCmd_humanNoMatchPipeModeIsPlain is the end-to-end pipe-mode
// test for the no-match error block: when stderr is a non-TTY writer
// (here, *bytes.Buffer), the open-tasks listing must be byte-identical
// to today's plain `<id>  <YYYY-MM-DD HH:MM>  [tags]  <description>\n`
// rendering with no header and no ANSI escapes, and the leading prose
// line must be unchanged.
func TestShowCmd_humanNoMatchPipeModeIsPlain(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	openDir := filepath.Join(tmp, "worktask", "open")
	if err := os.MkdirAll(openDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeTaskFile(t, openDir, task.Task{
		ID:      "11111111",
		Created: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC),
		Body:    "first task\n",
	})
	writeTaskFile(t, openDir, task.Task{
		ID:      "22222222",
		Created: time.Date(2026, 4, 29, 10, 30, 0, 0, time.UTC),
		Body:    "second task\n",
	})

	prevFormat := format
	format = formatHuman
	t.Cleanup(func() { format = prevFormat })

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"show", "zzz"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	err := rootCmd.Execute()
	if !errors.Is(err, errExit) {
		t.Fatalf("expected errExit, got %v", err)
	}

	want := "no match for \"zzz\". open tasks:\n" +
		"11111111  2026-04-29 09:00  []  first task\n" +
		"22222222  2026-04-29 10:30  []  second task\n"

	if stderr.String() != want {
		t.Errorf("no-match pipe-mode stderr does not match plain rendering.\n--- got ---\n%s\n--- want ---\n%s", stderr.String(), want)
	}
	if bytes.ContainsRune(stderr.Bytes(), '\x1b') {
		t.Errorf("no-match pipe-mode stderr must not contain ANSI escapes, got %q", stderr.String())
	}
}

// TestShowCmd_jsonIncludesAbsolutePath asserts that `btw show --format=json`
// emits a top-level `path` field whose value is the absolute filesystem path
// of the resolved task file. The test seeds an open-task file on disk, runs
// `show <id> --format=json`, and parses the stdout JSON to compare `path`
// against the path the test wrote to.
func TestShowCmd_jsonIncludesAbsolutePath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	tk := task.Task{
		ID:      "abcdef12",
		Created: time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC),
		Body:    "buy milk\nremember the brand\n",
	}
	raw, err := task.Encode(tk)
	if err != nil {
		t.Fatalf("task.Encode: %v", err)
	}
	openDir := filepath.Join(tmp, "worktask", "open")
	if err := os.MkdirAll(openDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	wantPath := filepath.Join(openDir, "abcdef12-buy-milk.md")
	if err := os.WriteFile(wantPath, raw, 0o644); err != nil {
		t.Fatalf("write task file: %v", err)
	}

	prevFormat := format
	format = formatJSON
	t.Cleanup(func() { format = prevFormat })

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"show", "abcdef12"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute show: %v (stderr=%s)", err, stderr.String())
	}

	var got struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("parse JSON: %v\nstdout=%s", err, stdout.String())
	}
	if got.Path == "" {
		t.Fatalf("expected `path` field in JSON output, got %s", stdout.String())
	}
	if !filepath.IsAbs(got.Path) {
		t.Errorf("`path` = %q; want an absolute path", got.Path)
	}
	if got.Path != wantPath {
		t.Errorf("`path` = %q; want %q", got.Path, wantPath)
	}
}

// TestShowCmd_humanStyledModePathOwnLine asserts that the styled human
// renderer includes the absolute task path on its own faint line between
// the metadata strip and the glamour-rendered body. The test exercises
// render.HumanShow directly with styled=true because the cobra command
// test harness uses *bytes.Buffer (non-TTY), which always takes the
// pipe-mode path.
func TestShowCmd_humanStyledModePathOwnLine(t *testing.T) {
	tk := task.Task{
		ID:      "abcdef12",
		Created: time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC),
		Body:    "buy milk\n",
	}
	raw, err := task.Encode(tk)
	if err != nil {
		t.Fatalf("task.Encode: %v", err)
	}

	taskPath := "/tasks/open/2026-04-29T11-30_abcdef12_buy-milk.md"

	var buf bytes.Buffer
	if err := render.HumanShow(&buf, tk, raw, taskPath, true); err != nil {
		t.Fatalf("HumanShow styled: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, taskPath) {
		t.Errorf("styled output must contain the path %q, got:\n%s", taskPath, out)
	}

	lines := strings.Split(out, "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines (metadata, path, body), got %d:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[1], taskPath) {
		t.Errorf("second line must contain the path; got line %q", lines[1])
	}
	if strings.Contains(lines[0], taskPath) {
		t.Errorf("metadata strip (first line) must not contain the path; got %q", lines[0])
	}
}
