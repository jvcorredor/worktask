// Package writeback applies a research run's side effects: appends a
// timestamped Research section to the task body, sets the research-meta
// frontmatter on findings/clarify (not failed), and appends a worklog line.
package writeback

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jvcorredor/bytheway/internal/research/runner"
	"github.com/jvcorredor/bytheway/internal/store"
)

// Input is the side-effect payload for one research run.
type Input struct {
	Fragment       string
	Result         runner.Result
	RunStart       time.Time
	LogPath        string // absolute path to the stream-json log
	LogPathRel     string // path relative to the tasks dir, persisted in frontmatter
	WorkingLogPath string
}

// Apply commits a research run's outputs. On failed status, no body section,
// no frontmatter mutation, no worklog entry — the task is left eligible for
// retry on the next run.
func Apply(s *store.Store, in Input) error {
	if in.Result.Status == "failed" {
		return nil
	}

	section := fmt.Sprintf("\n## Research %s\n\n%s", in.RunStart.UTC().Format(time.RFC3339), strings.TrimRight(in.Result.Body, "\n"))
	if !strings.HasSuffix(section, "\n") {
		section += "\n"
	}
	if _, err := s.Append(in.Fragment, section); err != nil {
		return fmt.Errorf("writeback: append body section: %w", err)
	}

	if _, err := s.SetResearchMeta(in.Fragment, in.RunStart, in.LogPathRel); err != nil {
		return fmt.Errorf("writeback: set research meta: %w", err)
	}

	if err := appendWorklog(in.WorkingLogPath, in.RunStart, in.Result.Summary); err != nil {
		return fmt.Errorf("writeback: append worklog: %w", err)
	}
	return nil
}

func appendWorklog(path string, when time.Time, summary string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	line := fmt.Sprintf("%s: research: %s\n", when.Local().Format("2006-01-02 15:04"), summary)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = f.WriteString(line)
	return err
}
