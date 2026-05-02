package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jvcorredor/worktask/internal/store"
)

func TestRun_TracerBullet(t *testing.T) {
	dataDir := t.TempDir()
	workDir := t.TempDir()
	workingPath := filepath.Join(workDir, "WORKING.md")

	input, err := os.ReadFile("testdata/simple_input.md")
	if err != nil {
		t.Fatalf("read input fixture: %v", err)
	}
	if err := os.WriteFile(workingPath, input, 0o644); err != nil {
		t.Fatalf("write working: %v", err)
	}
	expected, err := os.ReadFile("testdata/simple_expected.md")
	if err != nil {
		t.Fatalf("read expected fixture: %v", err)
	}

	s := store.New(dataDir)
	if err := Run(workingPath, s); err != nil {
		t.Fatalf("Run: %v", err)
	}

	openTasks, err := s.List(store.FilterOpen, 0)
	if err != nil {
		t.Fatalf("List open: %v", err)
	}
	if len(openTasks) != 1 {
		t.Fatalf("expected 1 open task, got %d", len(openTasks))
	}
	if openTasks[0].ID != "ba4d63a9" {
		t.Errorf("open task id = %q, want ba4d63a9", openTasks[0].ID)
	}
	wantOpenCreated := time.Date(2026, 4, 27, 10, 1, 0, 0, time.Local)
	if !openTasks[0].Created.Equal(wantOpenCreated) {
		t.Errorf("open task created = %v, want %v", openTasks[0].Created, wantOpenCreated)
	}
	if !openTasks[0].Completed.IsZero() {
		t.Errorf("open task completed = %v, want zero", openTasks[0].Completed)
	}
	if got, want := openTasks[0].Body, "follow up on aws open support cases\n"; got != want {
		t.Errorf("open task body = %q, want %q", got, want)
	}

	closedTasks, err := s.List(store.FilterClosed, 0)
	if err != nil {
		t.Fatalf("List closed: %v", err)
	}
	if len(closedTasks) != 1 {
		t.Fatalf("expected 1 closed task, got %d", len(closedTasks))
	}
	if closedTasks[0].ID != "4afd3135" {
		t.Errorf("closed task id = %q, want 4afd3135", closedTasks[0].ID)
	}
	wantClosedCreated := time.Date(2026, 4, 22, 18, 40, 0, 0, time.Local)
	if !closedTasks[0].Created.Equal(wantClosedCreated) {
		t.Errorf("closed task created = %v, want %v", closedTasks[0].Created, wantClosedCreated)
	}
	wantCompleted := time.Date(2026, 4, 27, 10, 27, 0, 0, time.Local)
	if !closedTasks[0].Completed.Equal(wantCompleted) {
		t.Errorf("closed task completed = %v, want %v", closedTasks[0].Completed, wantCompleted)
	}
	if got, want := closedTasks[0].Body, "cleanup project-context registry PR\n"; got != want {
		t.Errorf("closed task body = %q, want %q", got, want)
	}

	rewritten, err := os.ReadFile(workingPath)
	if err != nil {
		t.Fatalf("read rewritten: %v", err)
	}
	if string(rewritten) != string(expected) {
		t.Errorf("rewritten WORKING.md mismatch.\n--- got ---\n%s\n--- want ---\n%s", rewritten, expected)
	}
}

func TestRun_RefusesWhenAlreadyMigrated(t *testing.T) {
	dataDir := t.TempDir()
	workDir := t.TempDir()
	workingPath := filepath.Join(workDir, "WORKING.md")

	input, err := os.ReadFile("testdata/simple_input.md")
	if err != nil {
		t.Fatalf("read input fixture: %v", err)
	}
	if err := os.WriteFile(workingPath, input, 0o644); err != nil {
		t.Fatalf("write working: %v", err)
	}

	s := store.New(dataDir)
	if err := Run(workingPath, s); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	afterFirst, err := os.ReadFile(workingPath)
	if err != nil {
		t.Fatalf("read after first: %v", err)
	}

	if err := Run(workingPath, s); err == nil {
		t.Fatal("second Run: expected error, got nil")
	}

	openTasks, err := s.List(store.FilterOpen, 0)
	if err != nil {
		t.Fatalf("List open: %v", err)
	}
	if len(openTasks) != 1 {
		t.Errorf("open tasks after rerun = %d, want 1 (no duplicates)", len(openTasks))
	}
	closedTasks, err := s.List(store.FilterClosed, 0)
	if err != nil {
		t.Fatalf("List closed: %v", err)
	}
	if len(closedTasks) != 1 {
		t.Errorf("closed tasks after rerun = %d, want 1 (no duplicates)", len(closedTasks))
	}

	afterSecond, err := os.ReadFile(workingPath)
	if err != nil {
		t.Fatalf("read after second: %v", err)
	}
	if string(afterSecond) != string(afterFirst) {
		t.Errorf("WORKING.md changed on rerun.\n--- after first ---\n%s\n--- after second ---\n%s", afterFirst, afterSecond)
	}
}

func TestRun_AbsorbsIndentedContinuation(t *testing.T) {
	dataDir := t.TempDir()
	workDir := t.TempDir()
	workingPath := filepath.Join(workDir, "WORKING.md")

	input, err := os.ReadFile("testdata/multiline_input.md")
	if err != nil {
		t.Fatalf("read input fixture: %v", err)
	}
	if err := os.WriteFile(workingPath, input, 0o644); err != nil {
		t.Fatalf("write working: %v", err)
	}
	expected, err := os.ReadFile("testdata/multiline_expected.md")
	if err != nil {
		t.Fatalf("read expected fixture: %v", err)
	}

	s := store.New(dataDir)
	if err := Run(workingPath, s); err != nil {
		t.Fatalf("Run: %v", err)
	}

	openTasks, err := s.List(store.FilterOpen, 0)
	if err != nil {
		t.Fatalf("List open: %v", err)
	}
	if len(openTasks) != 1 {
		t.Fatalf("expected 1 open task, got %d", len(openTasks))
	}
	wantBody := "Multi-line task description\n" +
		"  - bullet one\n" +
		"  - bullet two\n" +
		"\n" +
		"  Continued paragraph after a blank.\n" +
		"  - bullet three\n"
	if got := openTasks[0].Body; got != wantBody {
		t.Errorf("task body mismatch.\n--- got ---\n%q\n--- want ---\n%q", got, wantBody)
	}

	rewritten, err := os.ReadFile(workingPath)
	if err != nil {
		t.Fatalf("read rewritten: %v", err)
	}
	if string(rewritten) != string(expected) {
		t.Errorf("rewritten WORKING.md mismatch.\n--- got ---\n%s\n--- want ---\n%s", rewritten, expected)
	}
}

func TestRun_RefusesWhenFileMissing(t *testing.T) {
	dataDir := t.TempDir()
	workDir := t.TempDir()

	s := store.New(dataDir)
	err := Run(filepath.Join(workDir, "does-not-exist.md"), s)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
