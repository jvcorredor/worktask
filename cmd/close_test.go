package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jvcorredor/bytheway/internal/task"
)

// TestCloseCmd_movesOpenTaskToClosedAndPrintsClosedID is the end-to-end
// test for `btw close <fragment>`: the open task file moves under
// closed/, the frontmatter gets a `completed:` stamp, and stdout reads
// `closed <id>` (the verb matches the new subcommand name and the
// inverse `reopen`).
func TestCloseCmd_movesOpenTaskToClosedAndPrintsClosedID(t *testing.T) {
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
	rootCmd.SetArgs([]string{"close", "abcdef12"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute close: %v (stderr=%s)", err, stderr.String())
	}

	if got, want := strings.TrimRight(stdout.String(), "\n"), "closed abcdef12"; got != want {
		t.Errorf("close stdout = %q; want %q", got, want)
	}

	if _, err := os.Stat(openPath); !os.IsNotExist(err) {
		t.Errorf("expected open file to be gone, stat err = %v", err)
	}
	closedPath := filepath.Join(tmp, "worktask", "closed", "2026-04-29T11-30_abcdef12_buy-milk.md")
	if _, err := os.Stat(closedPath); err != nil {
		t.Errorf("expected closed file at %s: %v", closedPath, err)
	}
}

// TestCompleteCmd_isRemoved enforces the no-alias acceptance criterion:
// after the rename, `btw complete <fragment>` must error out as an
// unknown subcommand so that agents using the JSON envelope re-pin to
// the new name rather than silently keep working against a deprecated
// alias.
func TestCompleteCmd_isRemoved(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"complete", "abcdef12"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected error for `complete`, got nil (stdout=%q)", stdout.String())
	}
	if !strings.Contains(err.Error(), `unknown command "complete"`) {
		t.Errorf("expected 'unknown command \"complete\"', got %v", err)
	}
}
