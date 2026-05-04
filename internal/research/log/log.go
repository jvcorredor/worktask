// Package log computes filesystem paths for research run log files
// under the btw tasks directory.
package log

import (
	"fmt"
	"path/filepath"
	"time"
)

const subdir = "research-logs"

// Dir returns the directory under tasksDir where research run logs are
// written. The directory is not created here; callers create it on
// demand before writing.
func Dir(tasksDir string) string {
	return filepath.Join(tasksDir, subdir)
}

// Path returns the absolute path of a research log file for the given
// task hash and run-start time, located under [Dir]. The filename
// embeds runStart in a filesystem-safe RFC-3339-like form so multiple
// runs against the same task do not collide.
func Path(tasksDir, hash string, runStart time.Time) string {
	name := fmt.Sprintf("%s_%s.jsonl", hash, runStart.Format("2006-01-02T15-04-05"))
	return filepath.Join(Dir(tasksDir), name)
}
