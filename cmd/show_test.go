package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jvcorredor/worktask/internal/task"
)

// TestShowCmd_humanPipeModePassesRawBytes is the end-to-end pipe-mode test:
// when stdout is a non-TTY writer (here, *bytes.Buffer), `worktask show
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
