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
