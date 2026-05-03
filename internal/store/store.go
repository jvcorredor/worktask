// Package store is the on-disk task repository: it owns the layout under
// the configured tasks directory and is the only package that reads from
// or writes to it.
//
// Directory layout. Open tasks live as files under tasks_dir/open and
// closed tasks under tasks_dir/closed. Filenames follow the
// "<RFC3339-minute>_<id>[_<slug>].md" pattern. Completing a task moves
// its file from open to closed; reopening moves it back. Both directories
// are created lazily on first use.
//
// Resolution. Methods that take a fragment look up a task by ID prefix or
// by description match across the requested subdirectories, returning
// [ErrNoMatch] when nothing matches and [*ErrAmbiguous] when more than
// one task matches.
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jvcorredor/worktask/internal/id"
	"github.com/jvcorredor/worktask/internal/slug"
	"github.com/jvcorredor/worktask/internal/task"
)

const slugMaxLen = 40

// Filter selects which subset of tasks [Store.List] returns.
type Filter int

// Filter values for [Store.List].
const (
	// FilterOpen returns only tasks under the open subdirectory.
	FilterOpen Filter = iota
	// FilterClosed returns only tasks under the closed subdirectory.
	FilterClosed
	// FilterAll returns open tasks followed by closed tasks.
	FilterAll
)

// Store is a handle to the on-disk task repository rooted at TasksDir.
// Now and NewID are injection points for tests; production callers obtain
// a Store with sensible defaults via [New].
type Store struct {
	TasksDir string
	Now      func() time.Time
	NewID    func() string
}

// New returns a [Store] rooted at tasksDir with production defaults:
// time.Now for timestamps and crypto-random ids from internal/id.
func New(tasksDir string) *Store {
	return &Store{
		TasksDir: tasksDir,
		Now:      time.Now,
		NewID:    id.New,
	}
}

// AddPreserved writes t verbatim to disk, choosing open/ when t.Completed
// is the zero value and closed/ otherwise. The slug used in the filename
// is derived from description, not from t.Body. Both subdirectories are
// created if missing. Intended for migrations and tests that need to
// place a task with caller-supplied identity and timestamps.
func (s *Store) AddPreserved(t task.Task, description string) error {
	openDir := filepath.Join(s.TasksDir, "open")
	closedDir := filepath.Join(s.TasksDir, "closed")
	for _, d := range []string{openDir, closedDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("store: mkdir %s: %w", d, err)
		}
	}
	data, err := task.Encode(t)
	if err != nil {
		return fmt.Errorf("store: encode: %w", err)
	}
	dir := openDir
	if !t.Completed.IsZero() {
		dir = closedDir
	}
	path := filepath.Join(dir, filename(t.ID, t.Created, slug.Generate(description, slugMaxLen)))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("store: write %s: %w", path, err)
	}
	return nil
}

// Add creates a new open task with description as its body, assigning a
// fresh ID and the current time as Created. The task file is written
// under tasks_dir/open and the resulting [task.Task] is returned. The
// open and closed subdirectories are created if missing.
func (s *Store) Add(description string) (task.Task, error) {
	created := s.Now()
	t := task.Task{
		ID:      s.NewID(),
		Created: created,
		Body:    description + "\n",
	}

	openDir := filepath.Join(s.TasksDir, "open")
	closedDir := filepath.Join(s.TasksDir, "closed")
	for _, d := range []string{openDir, closedDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return task.Task{}, fmt.Errorf("store: mkdir %s: %w", d, err)
		}
	}

	data, err := task.Encode(t)
	if err != nil {
		return task.Task{}, fmt.Errorf("store: encode: %w", err)
	}

	path := filepath.Join(openDir, filename(t.ID, created, slug.Generate(description, slugMaxLen)))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return task.Task{}, fmt.Errorf("store: write %s: %w", path, err)
	}
	return t, nil
}

// List returns the tasks selected by filter. Open tasks are returned in
// directory order; closed tasks are sorted by Completed descending and
// truncated to closedLimit when closedLimit is positive. When filter is
// [FilterAll], the result is open tasks followed by closed tasks.
// A missing subdirectory is treated as empty rather than as an error.
func (s *Store) List(filter Filter, closedLimit int) ([]task.Task, error) {
	var open, closed []task.Task
	if filter == FilterOpen || filter == FilterAll {
		ts, err := s.loadDir("open")
		if err != nil {
			return nil, err
		}
		open = ts
	}
	if filter == FilterClosed || filter == FilterAll {
		ts, err := s.loadDir("closed")
		if err != nil {
			return nil, err
		}
		sort.Slice(ts, func(i, j int) bool { return ts[j].Completed.Before(ts[i].Completed) })
		if closedLimit > 0 && len(ts) > closedLimit {
			ts = ts[:closedLimit]
		}
		closed = ts
	}
	return append(open, closed...), nil
}

func (s *Store) loadDir(sub string) ([]task.Task, error) {
	dir := filepath.Join(s.TasksDir, sub)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("store: read %s: %w", dir, err)
	}
	tasks := make([]task.Task, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("store: read %s: %w", path, err)
		}
		t, err := task.Decode(data)
		if err != nil {
			return nil, fmt.Errorf("store: decode %s: %w", path, err)
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

type loaded struct {
	task task.Task
	raw  []byte
	path string
}

// Candidate is a one-line summary of a task surfaced when fragment
// resolution is ambiguous.
type Candidate struct {
	ID          string
	Description string
}

// ErrAmbiguous is returned when a fragment resolves to more than one
// task. Candidates lists every task that matched the fragment.
type ErrAmbiguous struct {
	Fragment   string
	Candidates []Candidate
}

// Error implements the error interface for [*ErrAmbiguous].
func (e *ErrAmbiguous) Error() string {
	return fmt.Sprintf("store: ambiguous fragment %q (%d candidates)", e.Fragment, len(e.Candidates))
}

// ErrNoMatch is returned when a fragment matches no task in the
// requested subdirectories.
type ErrNoMatch struct {
	Fragment string
}

// Error implements the error interface for [*ErrNoMatch].
func (e *ErrNoMatch) Error() string {
	return fmt.Sprintf("store: no match for %q", e.Fragment)
}

// Get resolves fragment against both open and closed tasks and returns
// the matching [task.Task] together with the raw bytes of its file.
// Returns [*ErrNoMatch] or [*ErrAmbiguous] when resolution fails.
func (s *Store) Get(fragment string) (task.Task, []byte, error) {
	l, err := s.resolve(fragment, []string{"open", "closed"})
	if err != nil {
		return task.Task{}, nil, err
	}
	return l.task, l.raw, nil
}

// Complete marks the open task identified by fragment as completed at
// the current time, rewrites the file with the new frontmatter, and
// moves it from open/ to closed/. Returns [*ErrNoMatch] or
// [*ErrAmbiguous] when fragment does not resolve to exactly one open
// task.
func (s *Store) Complete(fragment string) (task.Task, error) {
	l, err := s.resolve(fragment, []string{"open"})
	if err != nil {
		return task.Task{}, err
	}
	completed := s.Now()
	t := l.task
	t.Completed = completed
	data, err := task.Encode(t)
	if err != nil {
		return task.Task{}, fmt.Errorf("store: encode: %w", err)
	}
	closedDir := filepath.Join(s.TasksDir, "closed")
	if err := os.MkdirAll(closedDir, 0o755); err != nil {
		return task.Task{}, fmt.Errorf("store: mkdir %s: %w", closedDir, err)
	}
	dst := filepath.Join(closedDir, filepath.Base(l.path))
	if err := os.WriteFile(l.path, data, 0o644); err != nil {
		return task.Task{}, fmt.Errorf("store: write %s: %w", l.path, err)
	}
	if err := os.Rename(l.path, dst); err != nil {
		return task.Task{}, fmt.Errorf("store: rename %s -> %s: %w", l.path, dst, err)
	}
	return t, nil
}

// Reopen clears the Completed timestamp on the closed task identified by
// fragment, rewrites the file, and moves it from closed/ back to open/.
// Returns [*ErrNoMatch] or [*ErrAmbiguous] when fragment does not
// resolve to exactly one closed task.
func (s *Store) Reopen(fragment string) (task.Task, error) {
	l, err := s.resolve(fragment, []string{"closed"})
	if err != nil {
		return task.Task{}, err
	}
	t := l.task
	t.Completed = time.Time{}
	data, err := task.Encode(t)
	if err != nil {
		return task.Task{}, fmt.Errorf("store: encode: %w", err)
	}
	openDir := filepath.Join(s.TasksDir, "open")
	if err := os.MkdirAll(openDir, 0o755); err != nil {
		return task.Task{}, fmt.Errorf("store: mkdir %s: %w", openDir, err)
	}
	dst := filepath.Join(openDir, filepath.Base(l.path))
	if err := os.WriteFile(l.path, data, 0o644); err != nil {
		return task.Task{}, fmt.Errorf("store: write %s: %w", l.path, err)
	}
	if err := os.Rename(l.path, dst); err != nil {
		return task.Task{}, fmt.Errorf("store: rename %s -> %s: %w", l.path, dst, err)
	}
	return t, nil
}

// Update replaces the first body line of the task identified by fragment
// with newDescription and rewrites the file in place. The remainder of
// the body and the open/closed location are preserved. Returns
// [*ErrNoMatch] or [*ErrAmbiguous] when fragment does not resolve.
func (s *Store) Update(fragment, newDescription string) (task.Task, error) {
	l, err := s.resolve(fragment, []string{"open", "closed"})
	if err != nil {
		return task.Task{}, err
	}
	t := l.task
	rest := t.Body
	if i := strings.IndexByte(rest, '\n'); i >= 0 {
		rest = rest[i:]
	} else {
		rest = ""
	}
	t.Body = newDescription + rest
	data, err := task.Encode(t)
	if err != nil {
		return task.Task{}, fmt.Errorf("store: encode: %w", err)
	}
	if err := os.WriteFile(l.path, data, 0o644); err != nil {
		return task.Task{}, fmt.Errorf("store: write %s: %w", l.path, err)
	}
	return t, nil
}

// SetResearchMeta updates the last_researched and last_research_log
// frontmatter fields on the task identified by fragment and rewrites the
// file in place. The task's open/closed location is preserved. Returns
// [*ErrNoMatch] or [*ErrAmbiguous] when fragment does not resolve.
func (s *Store) SetResearchMeta(fragment string, lastResearched time.Time, logPath string) (task.Task, error) {
	l, err := s.resolve(fragment, []string{"open", "closed"})
	if err != nil {
		return task.Task{}, err
	}
	t := l.task
	t.LastResearched = lastResearched
	t.LastResearchLog = logPath
	data, err := task.Encode(t)
	if err != nil {
		return task.Task{}, fmt.Errorf("store: encode: %w", err)
	}
	if err := os.WriteFile(l.path, data, 0o644); err != nil {
		return task.Task{}, fmt.Errorf("store: write %s: %w", l.path, err)
	}
	return t, nil
}

// Path returns the absolute path of the task file identified by
// fragment. The task may be open or closed. Returns [*ErrNoMatch] or
// [*ErrAmbiguous] when fragment does not resolve.
func (s *Store) Path(fragment string) (string, error) {
	l, err := s.resolve(fragment, []string{"open", "closed"})
	if err != nil {
		return "", err
	}
	return l.path, nil
}

// Append concatenates text to the body of the task identified by
// fragment, inserting a separating newline when the existing body does
// not already end with one, and rewrites the file in place. The task's
// open/closed location is preserved. Returns [*ErrNoMatch] or
// [*ErrAmbiguous] when fragment does not resolve.
func (s *Store) Append(fragment, text string) (task.Task, error) {
	l, err := s.resolve(fragment, []string{"open", "closed"})
	if err != nil {
		return task.Task{}, err
	}
	t := l.task
	if t.Body != "" && !strings.HasSuffix(t.Body, "\n") {
		t.Body += "\n"
	}
	t.Body += text
	data, err := task.Encode(t)
	if err != nil {
		return task.Task{}, fmt.Errorf("store: encode: %w", err)
	}
	if err := os.WriteFile(l.path, data, 0o644); err != nil {
		return task.Task{}, fmt.Errorf("store: write %s: %w", l.path, err)
	}
	return t, nil
}

func (s *Store) resolve(fragment string, subdirs []string) (loaded, error) {
	all, err := s.loadAll(subdirs)
	if err != nil {
		return loaded{}, err
	}
	frag := strings.ToLower(fragment)
	strategies := []func(loaded) bool{
		func(l loaded) bool { return l.task.ID == fragment },
		func(l loaded) bool { return strings.HasPrefix(l.task.ID, fragment) },
		func(l loaded) bool { return strings.ToLower(firstLine(l.task.Body)) == frag },
		func(l loaded) bool { return strings.Contains(strings.ToLower(firstLine(l.task.Body)), frag) },
	}
	for _, match := range strategies {
		var hits []loaded
		for _, l := range all {
			if match(l) {
				hits = append(hits, l)
			}
		}
		if len(hits) == 1 {
			return hits[0], nil
		}
		if len(hits) > 1 {
			cands := make([]Candidate, 0, len(hits))
			for _, h := range hits {
				cands = append(cands, Candidate{ID: h.task.ID, Description: firstLine(h.task.Body)})
			}
			return loaded{}, &ErrAmbiguous{Fragment: fragment, Candidates: cands}
		}
	}
	return loaded{}, &ErrNoMatch{Fragment: fragment}
}

func (s *Store) loadAll(subdirs []string) ([]loaded, error) {
	var out []loaded
	for _, sub := range subdirs {
		dir := filepath.Join(s.TasksDir, sub)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("store: read %s: %w", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
				continue
			}
			path := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("store: read %s: %w", path, err)
			}
			t, err := task.Decode(data)
			if err != nil {
				return nil, fmt.Errorf("store: decode %s: %w", path, err)
			}
			out = append(out, loaded{task: t, raw: data, path: path})
		}
	}
	return out, nil
}

func firstLine(body string) string {
	body = strings.TrimRight(body, "\n")
	if i := strings.IndexByte(body, '\n'); i >= 0 {
		return body[:i]
	}
	return body
}

func filename(id string, created time.Time, slug string) string {
	ts := created.Format("2006-01-02T15-04")
	if slug == "" {
		return fmt.Sprintf("%s_%s.md", ts, id)
	}
	return fmt.Sprintf("%s_%s_%s.md", ts, id, slug)
}
