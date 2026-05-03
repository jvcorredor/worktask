package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jvcorredor/worktask/internal/task"
)

// TestListCmd_humanPipeModeIsPlain is the end-to-end pipe-mode test:
// when stdout is a non-TTY writer (here, *bytes.Buffer), `worktask list`
// must emit the plain `id  YYYY-MM-DD HH:MM  description\n` rows with
// no header and no ANSI escapes — the contract callers rely on for
// piping into grep, awk, and other text tools.
func TestListCmd_humanPipeModeIsPlain(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	openDir := filepath.Join(tmp, "worktask", "open")
	if err := os.MkdirAll(openDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	tasks := []task.Task{
		{
			ID:      "11111111",
			Created: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC),
			Body:    "first task\n",
		},
		{
			ID:      "22222222",
			Created: time.Date(2026, 4, 29, 10, 30, 0, 0, time.UTC),
			Body:    "second task with a longer description that would otherwise be truncated in styled mode\n",
		},
	}
	for _, tk := range tasks {
		raw, err := task.Encode(tk)
		if err != nil {
			t.Fatalf("task.Encode: %v", err)
		}
		name := tk.ID + "-" + "task.md"
		if err := os.WriteFile(filepath.Join(openDir, name), raw, 0o644); err != nil {
			t.Fatalf("write task file: %v", err)
		}
	}

	prevFormat := format
	format = formatHuman
	t.Cleanup(func() { format = prevFormat })

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"list"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute list: %v (stderr=%s)", err, stderr.String())
	}

	want := "11111111  2026-04-29 09:00  first task\n" +
		"22222222  2026-04-29 10:30  second task with a longer description that would otherwise be truncated in styled mode\n"

	if stdout.String() != want {
		t.Errorf("list pipe-mode output does not match plain rows.\n--- got ---\n%s\n--- want ---\n%s", stdout.String(), want)
	}
	if bytes.ContainsRune(stdout.Bytes(), '\x1b') {
		t.Errorf("list pipe-mode output must not contain ANSI escapes, got %q", stdout.String())
	}
}
