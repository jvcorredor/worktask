// Command worktask-migrate is the one-shot migration tool that promotes
// the legacy WORKING.md working log into the per-task file layout under
// the configured tasks directory. The transformation contract is
// documented on [Run].
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jvcorredor/worktask/internal/config"
	"github.com/jvcorredor/worktask/internal/store"
	"github.com/jvcorredor/worktask/internal/task"
)

const tsLayout = "2006-01-02 15:04"

var taskLineRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}) \[task ([0-9a-f]{8})(?: — completed (\d{4}-\d{2}-\d{2} \d{2}:\d{2}))?\]: (.+)$`)

// Run migrates the legacy working-log file at workingMdPath into the
// per-task layout owned by s.
//
// Contract:
//
//   - Every "[task <id>]:" line in the working log becomes a task file
//     under s, preserving the original ID and Created timestamp parsed
//     from the line. Lines marked "— completed <ts>" populate Completed
//     so the task lands in closed/.
//   - Indented and blank lines that follow a task line are folded into
//     that task's body in source order; trailing blanks are returned to
//     the working log instead of being attached to the task.
//   - Non-task lines are preserved in the rewritten working log in their
//     original order.
//   - The rewritten working log is written back to workingMdPath after
//     all task files have been created.
//   - Run is not idempotent: calling it on a working log that contains
//     no task lines returns an error so a second invocation does not
//     silently truncate state.
func Run(workingMdPath string, s *store.Store) error {
	data, err := os.ReadFile(workingMdPath)
	if err != nil {
		return fmt.Errorf("migrate: read %s: %w", workingMdPath, err)
	}
	lines := strings.Split(string(data), "\n")
	keep := make([]string, 0, len(lines))
	type pendingTask struct {
		t    task.Task
		desc string
	}
	var tasks []pendingTask
	i := 0
	for i < len(lines) {
		t, desc, ok := parseTaskLine(lines[i])
		if !ok {
			keep = append(keep, lines[i])
			i++
			continue
		}
		bodyLines := []string{desc}
		j := i + 1
		var pendingBlanks []string
		for j < len(lines) {
			nl := lines[j]
			switch {
			case strings.HasPrefix(nl, " ") || strings.HasPrefix(nl, "\t"):
				bodyLines = append(bodyLines, pendingBlanks...)
				pendingBlanks = nil
				bodyLines = append(bodyLines, nl)
				j++
			case nl == "":
				pendingBlanks = append(pendingBlanks, nl)
				j++
			default:
				goto done
			}
		}
	done:
		t.Body = strings.Join(bodyLines, "\n") + "\n"
		tasks = append(tasks, pendingTask{t, desc})
		keep = append(keep, pendingBlanks...)
		i = j
	}
	if len(tasks) == 0 {
		return fmt.Errorf("migrate: no task lines found in %s (already migrated?)", workingMdPath)
	}
	for _, p := range tasks {
		if err := s.AddPreserved(p.t, p.desc); err != nil {
			return fmt.Errorf("migrate: add task %s: %w", p.t.ID, err)
		}
	}
	if err := os.WriteFile(workingMdPath, []byte(strings.Join(keep, "\n")), 0o644); err != nil {
		return fmt.Errorf("migrate: write %s: %w", workingMdPath, err)
	}
	return nil
}

func parseTaskLine(line string) (task.Task, string, bool) {
	m := taskLineRe.FindStringSubmatch(line)
	if m == nil {
		return task.Task{}, "", false
	}
	created, err := time.ParseInLocation(tsLayout, m[1], time.Local)
	if err != nil {
		return task.Task{}, "", false
	}
	t := task.Task{
		ID:      m[2],
		Created: created,
		Body:    m[4] + "\n",
	}
	if m[3] != "" {
		completed, err := time.ParseInLocation(tsLayout, m[3], time.Local)
		if err != nil {
			return task.Task{}, "", false
		}
		t.Completed = completed
	}
	return t, m[4], true
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: worktask-migrate <WORKING.md>")
		os.Exit(2)
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	s := store.New(cfg.TasksDir)
	if err := Run(os.Args[1], s); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "migrated tasks into %s\n", cfg.TasksDir)
}
