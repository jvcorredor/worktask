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

type Filter int

const (
	FilterOpen Filter = iota
	FilterClosed
	FilterAll
)

type Store struct {
	TasksDir string
	Now      func() time.Time
	NewID    func() string
}

func New(tasksDir string) *Store {
	return &Store{
		TasksDir: tasksDir,
		Now:      time.Now,
		NewID:    id.New,
	}
}

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

type Candidate struct {
	ID          string
	Description string
}

type ErrAmbiguous struct {
	Fragment   string
	Candidates []Candidate
}

func (e *ErrAmbiguous) Error() string {
	return fmt.Sprintf("store: ambiguous fragment %q (%d candidates)", e.Fragment, len(e.Candidates))
}

type ErrNoMatch struct {
	Fragment string
}

func (e *ErrNoMatch) Error() string {
	return fmt.Sprintf("store: no match for %q", e.Fragment)
}

func (s *Store) Get(fragment string) (task.Task, []byte, error) {
	l, err := s.resolve(fragment, []string{"open", "closed"})
	if err != nil {
		return task.Task{}, nil, err
	}
	return l.task, l.raw, nil
}

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

func (s *Store) Path(fragment string) (string, error) {
	l, err := s.resolve(fragment, []string{"open", "closed"})
	if err != nil {
		return "", err
	}
	return l.path, nil
}

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
