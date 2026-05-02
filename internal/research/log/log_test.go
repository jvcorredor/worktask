package log

import (
	"path/filepath"
	"testing"
	"time"
)

func TestPath_isDeterministicUnderTasksDir(t *testing.T) {
	tasksDir := "/var/data/worktask"
	hash := "abcdef12"
	runStart := time.Date(2026, 5, 1, 14, 30, 5, 0, time.UTC)

	got := Path(tasksDir, hash, runStart)
	want := filepath.Join(tasksDir, "research-logs", "abcdef12_2026-05-01T14-30-05.jsonl")
	if got != want {
		t.Errorf("Path = %q; want %q", got, want)
	}
}

func TestDir_isResearchLogsUnderTasksDir(t *testing.T) {
	got := Dir("/var/data/worktask")
	want := filepath.Join("/var/data/worktask", "research-logs")
	if got != want {
		t.Errorf("Dir = %q; want %q", got, want)
	}
}
