// Package log computes filesystem paths for research run log files
// under the worktask tasks directory.
package log

import (
	"fmt"
	"path/filepath"
	"time"
)

const subdir = "research-logs"

func Dir(tasksDir string) string {
	return filepath.Join(tasksDir, subdir)
}

func Path(tasksDir, hash string, runStart time.Time) string {
	name := fmt.Sprintf("%s_%s.jsonl", hash, runStart.Format("2006-01-02T15-04-05"))
	return filepath.Join(Dir(tasksDir), name)
}
