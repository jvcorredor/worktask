package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jvcorredor/bytheway/internal/task"
)

func TestAddCmd_withTagsCreatesTaggedTask(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	prevTags := addTags
	addTags = nil
	t.Cleanup(func() { addTags = prevTags })

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"add", "--tag", "bug", "--tag", "urgent", "fix login"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute add: %v (stderr=%s)", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "added") {
		t.Errorf("stdout = %q, want 'added ...'", stdout.String())
	}

	openDir := filepath.Join(tmp, "worktask", "open")
	entries, err := os.ReadDir(openDir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("no task files found in open dir")
	}

	data, err := os.ReadFile(filepath.Join(openDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("read task file: %v", err)
	}
	got, err := task.Decode(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "bug" || got.Tags[1] != "urgent" {
		t.Errorf("tags = %v, want [bug urgent]", got.Tags)
	}
}

func TestAddCmd_rejectsInvalidTag(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	prevTags := addTags
	addTags = nil
	t.Cleanup(func() { addTags = prevTags })

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"add", "--tag", "bug!", "fix login"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected error for invalid tag, got success")
	}
}

func TestAddCmd_normalizesTagCase(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	prevTags := addTags
	addTags = nil
	t.Cleanup(func() { addTags = prevTags })

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"add", "--tag", "BUG", "fix login"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute add: %v (stderr=%s)", err, stderr.String())
	}

	openDir := filepath.Join(tmp, "worktask", "open")
	entries, err := os.ReadDir(openDir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(openDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	got, err := task.Decode(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "bug" {
		t.Errorf("tags = %v, want [bug] (normalized)", got.Tags)
	}
}

func TestAddCmd_noTagsCreatesTaskWithoutTags(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	prevTags := addTags
	addTags = nil
	t.Cleanup(func() { addTags = prevTags })

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"add", "fix login"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute add: %v (stderr=%s)", err, stderr.String())
	}

	openDir := filepath.Join(tmp, "worktask", "open")
	entries, err := os.ReadDir(openDir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(openDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	got, err := task.Decode(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Tags) != 0 {
		t.Errorf("tags = %v, want empty", got.Tags)
	}
}
