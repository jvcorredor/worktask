package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/jvcorredor/bytheway/internal/task"
)

func TestAdd_writesFileToOpenDir(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }

	task, err := s.Add("buy milk")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	wantPath := filepath.Join(dir, "open", "2026-04-29T11-30_abcdef12_buy-milk.md")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("expected file at %s: %v", wantPath, err)
	}

	if task.ID != "abcdef12" {
		t.Errorf("task.ID = %q; want %q", task.ID, "abcdef12")
	}

	got, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "---\nid: abcdef12\ncreated: 2026-04-29T11:30:00Z\n---\nbuy milk\n"
	if string(got) != want {
		t.Errorf("file contents:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestAdd_unicodeOnlyDescriptionOmitsSlug(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }

	if _, err := s.Add("🚀"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	wantPath := filepath.Join(dir, "open", "2026-04-29T11-30_abcdef12.md")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("expected file at %s: %v", wantPath, err)
	}
}

func TestList_returnsOpenTasksInChronologicalOrder(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	s.Now = func() time.Time { return time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "11111111" }
	if _, err := s.Add("first"); err != nil {
		t.Fatalf("Add first: %v", err)
	}

	s.Now = func() time.Time { return time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "22222222" }
	if _, err := s.Add("second"); err != nil {
		t.Fatalf("Add second: %v", err)
	}

	tasks, err := s.List(FilterOpen, 0, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("len(tasks) = %d; want 2", len(tasks))
	}
	if tasks[0].ID != "11111111" {
		t.Errorf("tasks[0].ID = %q; want 11111111", tasks[0].ID)
	}
	if tasks[1].ID != "22222222" {
		t.Errorf("tasks[1].ID = %q; want 22222222", tasks[1].ID)
	}
	if tasks[0].Body != "first\n" {
		t.Errorf("tasks[0].Body = %q; want %q", tasks[0].Body, "first\n")
	}
}

func TestGet_exactID(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, raw, gotPath, err := s.Get("abcdef12")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "abcdef12" {
		t.Errorf("got.ID = %q; want %q", got.ID, "abcdef12")
	}
	if got.Body != "buy milk\n" {
		t.Errorf("got.Body = %q; want %q", got.Body, "buy milk\n")
	}
	wantRaw := "---\nid: abcdef12\ncreated: 2026-04-29T11:30:00Z\n---\nbuy milk\n"
	if string(raw) != wantRaw {
		t.Errorf("raw bytes:\n--- got ---\n%s\n--- want ---\n%s", raw, wantRaw)
	}
	wantPath, err := s.Path("abcdef12")
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if gotPath != wantPath {
		t.Errorf("Get path = %q; want %q (must match Store.Path for the same fragment)", gotPath, wantPath)
	}
}

func TestGet_idPrefix(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, _, _, err := s.Get("abcd")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "abcdef12" {
		t.Errorf("got.ID = %q; want %q", got.ID, "abcdef12")
	}
}

func TestGet_descriptionExactCI(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("Buy Milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, _, _, err := s.Get("buy milk")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "abcdef12" {
		t.Errorf("got.ID = %q; want %q", got.ID, "abcdef12")
	}
}

func TestGet_descriptionSubstring(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("Buy Milk Today"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, _, _, err := s.Get("MILK")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "abcdef12" {
		t.Errorf("got.ID = %q; want %q", got.ID, "abcdef12")
	}
}

func TestGet_idExactShortCircuits(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("first task"); err != nil {
		t.Fatalf("Add A: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "11111111" }
	if _, err := s.Add("see abcdef12 for context"); err != nil {
		t.Fatalf("Add B: %v", err)
	}

	got, _, _, err := s.Get("abcdef12")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "abcdef12" {
		t.Errorf("got.ID = %q; want %q (id-exact must short-circuit before description-substring fires)", got.ID, "abcdef12")
	}
}

func TestGet_ambiguousReturnsTypedErrorWithCandidates(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "11111111" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add A: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "22222222" }
	if _, err := s.Add("buy milkshake"); err != nil {
		t.Fatalf("Add B: %v", err)
	}

	_, _, _, err := s.Get("milk")
	if err == nil {
		t.Fatalf("Get: expected error")
	}
	var amb *ErrAmbiguous
	if !errors.As(err, &amb) {
		t.Fatalf("error type = %T (%v); want *ErrAmbiguous", err, err)
	}
	if amb.Fragment != "milk" {
		t.Errorf("Fragment = %q; want %q", amb.Fragment, "milk")
	}
	if len(amb.Candidates) != 2 {
		t.Fatalf("len(Candidates) = %d; want 2", len(amb.Candidates))
	}
	gotIDs := map[string]string{}
	for _, c := range amb.Candidates {
		gotIDs[c.ID] = c.Description
	}
	if gotIDs["11111111"] != "buy milk" {
		t.Errorf("Candidates[11111111].Description = %q; want %q", gotIDs["11111111"], "buy milk")
	}
	if gotIDs["22222222"] != "buy milkshake" {
		t.Errorf("Candidates[22222222].Description = %q; want %q", gotIDs["22222222"], "buy milkshake")
	}
}

func TestGet_ambiguousCandidatesCarryTags(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "11111111" }
	if _, err := s.Add("buy milk", "errand", "shopping"); err != nil {
		t.Fatalf("Add A: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "22222222" }
	if _, err := s.Add("buy milkshake"); err != nil {
		t.Fatalf("Add B: %v", err)
	}

	_, _, _, err := s.Get("milk")
	var amb *ErrAmbiguous
	if !errors.As(err, &amb) {
		t.Fatalf("error type = %T (%v); want *ErrAmbiguous", err, err)
	}
	gotTags := map[string][]string{}
	for _, c := range amb.Candidates {
		gotTags[c.ID] = c.Tags
	}
	wantTagged := []string{"errand", "shopping"}
	if !reflect.DeepEqual(gotTags["11111111"], wantTagged) {
		t.Errorf("Candidates[11111111].Tags = %v; want %v", gotTags["11111111"], wantTagged)
	}
	if got := gotTags["22222222"]; len(got) != 0 {
		t.Errorf("Candidates[22222222].Tags = %v; want empty (no tags on source task)", got)
	}
}

func TestGet_zeroMatchReturnsTypedError(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	_, _, _, err := s.Get("nonexistent-zzz")
	if err == nil {
		t.Fatalf("Get: expected error")
	}
	var nm *ErrNoMatch
	if !errors.As(err, &nm) {
		t.Fatalf("error type = %T (%v); want *ErrNoMatch", err, err)
	}
	if nm.Fragment != "nonexistent-zzz" {
		t.Errorf("Fragment = %q; want %q", nm.Fragment, "nonexistent-zzz")
	}
}

func TestGet_resolvesClosedTasks(t *testing.T) {
	dir := t.TempDir()
	closedDir := filepath.Join(dir, "closed")
	if err := os.MkdirAll(closedDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	created := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	completed := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	closedTask := task.Task{
		ID:        "deadbeef",
		Created:   created,
		Completed: completed,
		Body:      "shipped feature\n",
	}
	data, err := task.Encode(closedTask)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	path := filepath.Join(closedDir, "2026-04-28T12-00_deadbeef_shipped-feature.md")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	s := New(dir)
	got, _, _, err := s.Get("deadbeef")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "deadbeef" {
		t.Errorf("got.ID = %q; want %q", got.ID, "deadbeef")
	}
	if !got.Completed.Equal(completed) {
		t.Errorf("got.Completed = %v; want %v", got.Completed, completed)
	}
}

func TestClose_movesFileFromOpenToClosedAndStampsCompleted(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	completedAt := time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return completedAt }

	got, err := s.Close("abcdef12")
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got.ID != "abcdef12" {
		t.Errorf("got.ID = %q; want %q", got.ID, "abcdef12")
	}
	if !got.Completed.Equal(completedAt) {
		t.Errorf("got.Completed = %v; want %v", got.Completed, completedAt)
	}

	openPath := filepath.Join(dir, "open", "2026-04-29T11-30_abcdef12_buy-milk.md")
	if _, err := os.Stat(openPath); !os.IsNotExist(err) {
		t.Errorf("expected open file to be gone, stat err = %v", err)
	}

	closedPath := filepath.Join(dir, "closed", "2026-04-29T11-30_abcdef12_buy-milk.md")
	data, err := os.ReadFile(closedPath)
	if err != nil {
		t.Fatalf("expected closed file at %s: %v", closedPath, err)
	}
	want := "---\nid: abcdef12\ncreated: 2026-04-29T11:30:00Z\ncompleted: 2026-04-30T09:00:00Z\n---\nbuy milk\n"
	if string(data) != want {
		t.Errorf("closed file contents:\n--- got ---\n%s\n--- want ---\n%s", data, want)
	}
}

func TestReopen_movesFileFromClosedToOpenAndClearsCompleted(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC) }
	if _, err := s.Close("abcdef12"); err != nil {
		t.Fatalf("Close: %v", err)
	}

	got, err := s.Reopen("abcdef12")
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if got.ID != "abcdef12" {
		t.Errorf("got.ID = %q; want %q", got.ID, "abcdef12")
	}
	if !got.Completed.IsZero() {
		t.Errorf("got.Completed = %v; want zero", got.Completed)
	}

	closedPath := filepath.Join(dir, "closed", "2026-04-29T11-30_abcdef12_buy-milk.md")
	if _, err := os.Stat(closedPath); !os.IsNotExist(err) {
		t.Errorf("expected closed file to be gone, stat err = %v", err)
	}

	openPath := filepath.Join(dir, "open", "2026-04-29T11-30_abcdef12_buy-milk.md")
	data, err := os.ReadFile(openPath)
	if err != nil {
		t.Fatalf("expected open file at %s: %v", openPath, err)
	}
	want := "---\nid: abcdef12\ncreated: 2026-04-29T11:30:00Z\n---\nbuy milk\n"
	if string(data) != want {
		t.Errorf("open file contents:\n--- got ---\n%s\n--- want ---\n%s", data, want)
	}
}

func TestClose_fragmentMatchingOnlyClosedReturnsNoMatch(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC) }
	if _, err := s.Close("abcdef12"); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	_, err := s.Close("abcdef12")
	if err == nil {
		t.Fatalf("Close: expected error")
	}
	var nm *ErrNoMatch
	if !errors.As(err, &nm) {
		t.Fatalf("error type = %T (%v); want *ErrNoMatch", err, err)
	}
}

func TestReopen_fragmentMatchingOnlyOpenReturnsNoMatch(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	_, err := s.Reopen("abcdef12")
	if err == nil {
		t.Fatalf("Reopen: expected error")
	}
	var nm *ErrNoMatch
	if !errors.As(err, &nm) {
		t.Fatalf("error type = %T (%v); want *ErrNoMatch", err, err)
	}
}

func TestList_filterClosedSortsByCompletedDescending(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	s.Now = func() time.Time { return time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "11111111" }
	if _, err := s.Add("first"); err != nil {
		t.Fatalf("Add first: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "22222222" }
	if _, err := s.Add("second"); err != nil {
		t.Fatalf("Add second: %v", err)
	}

	s.Now = func() time.Time { return time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC) }
	if _, err := s.Close("11111111"); err != nil {
		t.Fatalf("Close first: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC) }
	if _, err := s.Close("22222222"); err != nil {
		t.Fatalf("Close second: %v", err)
	}

	tasks, err := s.List(FilterClosed, 20, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("len(tasks) = %d; want 2", len(tasks))
	}
	if tasks[0].ID != "22222222" {
		t.Errorf("tasks[0].ID = %q; want 22222222 (most recently completed first)", tasks[0].ID)
	}
	if tasks[1].ID != "11111111" {
		t.Errorf("tasks[1].ID = %q; want 11111111", tasks[1].ID)
	}
}

func TestList_filterClosedHonorsLimit(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	for i := 0; i < 5; i++ {
		hour := 9 + i
		s.Now = func() time.Time { return time.Date(2026, 4, 29, hour, 0, 0, 0, time.UTC) }
		id := fmt.Sprintf("%08d", i)
		s.NewID = func() string { return id }
		if _, err := s.Add(fmt.Sprintf("task %d", i)); err != nil {
			t.Fatalf("Add %d: %v", i, err)
		}
		s.Now = func() time.Time { return time.Date(2026, 4, 30, hour, 0, 0, 0, time.UTC) }
		if _, err := s.Close(id); err != nil {
			t.Fatalf("Close %d: %v", i, err)
		}
	}

	tasks, err := s.List(FilterClosed, 3, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("len(tasks) = %d; want 3", len(tasks))
	}
	if tasks[0].ID != "00000004" {
		t.Errorf("tasks[0].ID = %q; want 00000004 (most recently completed)", tasks[0].ID)
	}
}

func TestList_filterAllReturnsOpenUncappedAndClosedCapped(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	for i := 0; i < 4; i++ {
		hour := 9 + i
		s.Now = func() time.Time { return time.Date(2026, 4, 29, hour, 0, 0, 0, time.UTC) }
		id := fmt.Sprintf("c%07d", i)
		s.NewID = func() string { return id }
		if _, err := s.Add(fmt.Sprintf("closed %d", i)); err != nil {
			t.Fatalf("Add closed %d: %v", i, err)
		}
		s.Now = func() time.Time { return time.Date(2026, 4, 30, hour, 0, 0, 0, time.UTC) }
		if _, err := s.Close(id); err != nil {
			t.Fatalf("Close %d: %v", i, err)
		}
	}
	for i := 0; i < 3; i++ {
		s.Now = func() time.Time { return time.Date(2026, 5, 1, 9+i, 0, 0, 0, time.UTC) }
		s.NewID = func() string { return fmt.Sprintf("o%07d", i) }
		if _, err := s.Add(fmt.Sprintf("open %d", i)); err != nil {
			t.Fatalf("Add open %d: %v", i, err)
		}
	}

	tasks, err := s.List(FilterAll, 2, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	openCount := 0
	closedCount := 0
	for _, tk := range tasks {
		if tk.Completed.IsZero() {
			openCount++
		} else {
			closedCount++
		}
	}
	if openCount != 3 {
		t.Errorf("openCount = %d; want 3 (open never truncated)", openCount)
	}
	if closedCount != 2 {
		t.Errorf("closedCount = %d; want 2 (closed capped at limit)", closedCount)
	}
}

func TestList_filterOpenIgnoresLimit(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	for i := 0; i < 5; i++ {
		s.Now = func() time.Time { return time.Date(2026, 4, 29, 9+i, 0, 0, 0, time.UTC) }
		s.NewID = func() string { return fmt.Sprintf("%08d", i) }
		if _, err := s.Add(fmt.Sprintf("task %d", i)); err != nil {
			t.Fatalf("Add %d: %v", i, err)
		}
	}

	tasks, err := s.List(FilterOpen, 2, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 5 {
		t.Errorf("len(tasks) = %d; want 5 (open never truncated by limit)", len(tasks))
	}
}

func TestList_tagFilterNarrowsToTaggedTasks(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	s.Now = func() time.Time { return time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "11111111" }
	if _, err := s.Add("first", "bug"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "22222222" }
	if _, err := s.Add("second", "infra"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "33333333" }
	if _, err := s.Add("third", "bug", "infra"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.List(FilterOpen, 0, "bug")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d; want 2 (only tasks tagged 'bug')", len(got))
	}
	for _, tk := range got {
		hasBug := false
		for _, tg := range tk.Tags {
			if tg == "bug" {
				hasBug = true
			}
		}
		if !hasBug {
			t.Errorf("returned task %s lacks 'bug' tag (tags=%v)", tk.ID, tk.Tags)
		}
	}
}

func TestList_tagFilterReturnsEmptyForNoMatch(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "11111111" }
	if _, err := s.Add("only", "infra"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.List(FilterOpen, 0, "nonexistent")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len(got) = %d; want 0 (non-matching tag should return empty list, no error)", len(got))
	}
}

func TestList_tagFilterComposesAcrossOpenAndClosed(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	s.Now = func() time.Time { return time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "11111111" }
	if _, err := s.Add("open-bug", "bug"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "22222222" }
	if _, err := s.Add("closed-bug", "bug"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC) }
	if _, err := s.Close("22222222"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "33333333" }
	if _, err := s.Add("infra-only", "infra"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.List(FilterAll, 0, "bug")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len(got) = %d; want 2 (one open + one closed bug)", len(got))
	}
}

func TestList_emptyTagFilterIsNoOp(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	s.Now = func() time.Time { return time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "11111111" }
	if _, err := s.Add("untagged"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "22222222" }
	if _, err := s.Add("tagged", "bug"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.List(FilterOpen, 0, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len(got) = %d; want 2 (empty tag must not filter)", len(got))
	}
}

func TestUpdate_rewritesFirstBodyLinePreservingFrontmatterFilenameAndRest(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	path := filepath.Join(dir, "open", "2026-04-29T11-30_abcdef12_buy-milk.md")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	withExtraBody := string(original) + "context line\nmore context\n"
	if err := os.WriteFile(path, []byte(withExtraBody), 0o644); err != nil {
		t.Fatalf("seed extra body: %v", err)
	}

	got, err := s.Update("abcdef12", "buy oat milk")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.ID != "abcdef12" {
		t.Errorf("got.ID = %q; want %q", got.ID, "abcdef12")
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected filename to be unchanged at %s: %v", path, err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after update: %v", err)
	}
	want := "---\nid: abcdef12\ncreated: 2026-04-29T11:30:00Z\n---\nbuy oat milk\ncontext line\nmore context\n"
	if string(data) != want {
		t.Errorf("file contents:\n--- got ---\n%s\n--- want ---\n%s", data, want)
	}
}

func TestUpdate_zeroMatchReturnsTypedError(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	_, err := s.Update("nonexistent-zzz", "new description")
	if err == nil {
		t.Fatalf("Update: expected error")
	}
	var nm *ErrNoMatch
	if !errors.As(err, &nm) {
		t.Fatalf("error type = %T (%v); want *ErrNoMatch", err, err)
	}
}

func TestAppend_bodyEndingInNewlineGetsTextDirectlyAppended(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if _, err := s.Append("abcdef12", "extra note\n"); err != nil {
		t.Fatalf("Append: %v", err)
	}

	path := filepath.Join(dir, "open", "2026-04-29T11-30_abcdef12_buy-milk.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "---\nid: abcdef12\ncreated: 2026-04-29T11:30:00Z\n---\nbuy milk\nextra note\n"
	if string(data) != want {
		t.Errorf("file contents:\n--- got ---\n%s\n--- want ---\n%s", data, want)
	}
}

func TestAppend_bodyMissingTrailingNewlineGetsLeadingNewlineInjected(t *testing.T) {
	dir := t.TempDir()
	openDir := filepath.Join(dir, "open")
	if err := os.MkdirAll(openDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(openDir, "2026-04-29T11-30_abcdef12_buy-milk.md")
	original := "---\nid: abcdef12\ncreated: 2026-04-29T11:30:00Z\n---\nbuy milk"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	s := New(dir)
	if _, err := s.Append("abcdef12", "extra note\n"); err != nil {
		t.Fatalf("Append: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "---\nid: abcdef12\ncreated: 2026-04-29T11:30:00Z\n---\nbuy milk\nextra note\n"
	if string(data) != want {
		t.Errorf("file contents:\n--- got ---\n%s\n--- want ---\n%s", data, want)
	}
}

func TestAppend_zeroMatchReturnsTypedError(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	_, err := s.Append("nonexistent-zzz", "anything")
	if err == nil {
		t.Fatalf("Append: expected error")
	}
	var nm *ErrNoMatch
	if !errors.As(err, &nm) {
		t.Fatalf("error type = %T (%v); want *ErrNoMatch", err, err)
	}
}

func TestSetResearchMeta_updatesFrontmatterPreservesBody(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := s.Append("abcdef12", "extra body context\n"); err != nil {
		t.Fatalf("Append: %v", err)
	}

	lastResearched := time.Date(2026, 5, 1, 14, 0, 0, 0, time.UTC)
	logPath := "research-logs/abcdef12_2026-05-01T14-00.jsonl"
	if _, err := s.SetResearchMeta("abcdef12", lastResearched, logPath); err != nil {
		t.Fatalf("SetResearchMeta: %v", err)
	}

	path := filepath.Join(dir, "open", "2026-04-29T11-30_abcdef12_buy-milk.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "---\nid: abcdef12\ncreated: 2026-04-29T11:30:00Z\n" +
		"last_researched: 2026-05-01T14:00:00Z\n" +
		"last_research_log: research-logs/abcdef12_2026-05-01T14-00.jsonl\n" +
		"---\nbuy milk\nextra body context\n"
	if string(data) != want {
		t.Errorf("file contents:\n--- got ---\n%s\n--- want ---\n%s", data, want)
	}
}

func TestSetResearchMeta_zeroMatchReturnsTypedError(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	_, err := s.SetResearchMeta("nonexistent-zzz", time.Now(), "ignored")
	if err == nil {
		t.Fatalf("SetResearchMeta: expected error")
	}
	var nm *ErrNoMatch
	if !errors.As(err, &nm) {
		t.Fatalf("error type = %T (%v); want *ErrNoMatch", err, err)
	}
}

func TestPath_returnsResolvedFilePath(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.Path("abcdef12")
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	want := filepath.Join(dir, "open", "2026-04-29T11-30_abcdef12_buy-milk.md")
	if got != want {
		t.Errorf("Path = %q; want %q", got, want)
	}
}

// TestPath_returnsAbsolutePathEvenWhenTasksDirIsRelative pins the
// docstring contract on Store.Path: callers receive a path they can use
// from any working directory, regardless of how the store was
// configured. We chdir into a tempdir, point the store at a relative
// "tasks" sibling, and assert the returned path is absolute.
func TestPath_returnsAbsolutePathEvenWhenTasksDirIsRelative(t *testing.T) {
	root := t.TempDir()
	tasksAbs := filepath.Join(root, "tasks")
	if err := os.MkdirAll(tasksAbs, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	s := New("tasks")
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.Path("abcdef12")
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("Path = %q; want an absolute path (TasksDir was relative)", got)
	}
}

func TestAdd_freshDirCreatesOpenAndClosed(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "fresh")
	s := New(dir)

	if _, err := s.Add("anything"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	for _, sub := range []string{"open", "closed"} {
		info, err := os.Stat(filepath.Join(dir, sub))
		if err != nil {
			t.Errorf("expected %s/ to exist: %v", sub, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s/ to be a directory", sub)
		}
	}
}

func TestAdd_withTagsWritesTagsInFrontmatter(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }

	tk, err := s.Add("fix login", "bug", "urgent")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if len(tk.Tags) != 2 || tk.Tags[0] != "bug" || tk.Tags[1] != "urgent" {
		t.Errorf("task.Tags = %v, want [bug urgent]", tk.Tags)
	}

	path := filepath.Join(dir, "open", "2026-04-29T11-30_abcdef12_fix-login.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "---\nid: abcdef12\ncreated: 2026-04-29T11:30:00Z\ntags: [bug, urgent]\n---\nfix login\n"
	if string(data) != want {
		t.Errorf("file contents:\n--- got ---\n%s\n--- want ---\n%s", data, want)
	}
}

func TestAdd_rejectsInvalidTag(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }

	_, err := s.Add("fix login", "bug!", "urgent")
	if err == nil {
		t.Fatalf("Add with invalid tag: expected error")
	}
}

func TestAdd_normalizesTagCase(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }

	tk, err := s.Add("fix login", "BUG", "Urgent")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if len(tk.Tags) != 2 || tk.Tags[0] != "bug" || tk.Tags[1] != "urgent" {
		t.Errorf("task.Tags = %v, want [bug urgent] (normalized)", tk.Tags)
	}
}

func TestTagsSurviveClose(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("fix login", "bug", "urgent"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	s.Now = func() time.Time { return time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC) }
	got, err := s.Close("abcdef12")
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "bug" || got.Tags[1] != "urgent" {
		t.Errorf("tags after close = %v, want [bug urgent]", got.Tags)
	}
}

func TestTagsSurviveReopen(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("fix login", "bug"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC) }
	if _, err := s.Close("abcdef12"); err != nil {
		t.Fatalf("Close: %v", err)
	}

	got, err := s.Reopen("abcdef12")
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "bug" {
		t.Errorf("tags after reopen = %v, want [bug]", got.Tags)
	}
}

func TestTagsSurviveUpdate(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("fix login", "bug"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.Update("abcdef12", "fix auth login")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "bug" {
		t.Errorf("tags after update = %v, want [bug]", got.Tags)
	}
}

func TestTagsSurviveAppend(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("fix login", "bug"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.Append("abcdef12", "extra note\n")
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "bug" {
		t.Errorf("tags after append = %v, want [bug]", got.Tags)
	}
}

func TestAddTag_addsTagToTaskWithNoTags(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.AddTag("abcdef12", "urgent")
	if err != nil {
		t.Fatalf("AddTag: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "urgent" {
		t.Errorf("tags after AddTag = %v, want [urgent]", got.Tags)
	}

	path := filepath.Join(dir, "open", "2026-04-29T11-30_abcdef12_buy-milk.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "---\nid: abcdef12\ncreated: 2026-04-29T11:30:00Z\ntags: [urgent]\n---\nbuy milk\n"
	if string(data) != want {
		t.Errorf("file contents:\n--- got ---\n%s\n--- want ---\n%s", data, want)
	}
}

func TestAddTag_idempotent(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk", "urgent"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.AddTag("abcdef12", "urgent")
	if err != nil {
		t.Fatalf("AddTag: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "urgent" {
		t.Errorf("tags after idempotent AddTag = %v, want [urgent]", got.Tags)
	}

	path := filepath.Join(dir, "open", "2026-04-29T11-30_abcdef12_buy-milk.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "---\nid: abcdef12\ncreated: 2026-04-29T11:30:00Z\ntags: [urgent]\n---\nbuy milk\n"
	if string(data) != want {
		t.Errorf("file contents:\n--- got ---\n%s\n--- want ---\n%s", data, want)
	}
}

func TestAddTag_normalizesUppercaseTag(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.AddTag("abcdef12", "URGENT")
	if err != nil {
		t.Fatalf("AddTag: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "urgent" {
		t.Errorf("tags after AddTag = %v, want [urgent] (normalized)", got.Tags)
	}
}

func TestAddTag_rejectsInvalidTag(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	_, err := s.AddTag("abcdef12", "bad tag!")
	if err == nil {
		t.Fatalf("AddTag with invalid tag: expected error")
	}
}

func TestAddTag_zeroMatchReturnsTypedError(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	_, err := s.AddTag("nonexistent-zzz", "urgent")
	if err == nil {
		t.Fatalf("AddTag: expected error")
	}
	var nm *ErrNoMatch
	if !errors.As(err, &nm) {
		t.Fatalf("error type = %T (%v); want *ErrNoMatch", err, err)
	}
}

func TestRemoveTag_removesTagAndRewritesFile(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk", "urgent", "shopping"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.RemoveTag("abcdef12", "urgent")
	if err != nil {
		t.Fatalf("RemoveTag: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "shopping" {
		t.Errorf("tags after RemoveTag = %v, want [shopping]", got.Tags)
	}

	path := filepath.Join(dir, "open", "2026-04-29T11-30_abcdef12_buy-milk.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "---\nid: abcdef12\ncreated: 2026-04-29T11:30:00Z\ntags: [shopping]\n---\nbuy milk\n"
	if string(data) != want {
		t.Errorf("file contents:\n--- got ---\n%s\n--- want ---\n%s", data, want)
	}
}

func TestRemoveTag_idempotent(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk", "shopping"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.RemoveTag("abcdef12", "urgent")
	if err != nil {
		t.Fatalf("RemoveTag: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "shopping" {
		t.Errorf("tags after idempotent RemoveTag = %v, want [shopping]", got.Tags)
	}
}

func TestRemoveTag_removingLastTagOmitsTagsLine(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk", "urgent"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.RemoveTag("abcdef12", "urgent")
	if err != nil {
		t.Fatalf("RemoveTag: %v", err)
	}
	if len(got.Tags) != 0 {
		t.Errorf("tags after RemoveTag = %v, want []", got.Tags)
	}

	path := filepath.Join(dir, "open", "2026-04-29T11-30_abcdef12_buy-milk.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "---\nid: abcdef12\ncreated: 2026-04-29T11:30:00Z\n---\nbuy milk\n"
	if string(data) != want {
		t.Errorf("file contents:\n--- got ---\n%s\n--- want ---\n%s", data, want)
	}
}

func TestRemoveTag_zeroMatchReturnsTypedError(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC) }
	s.NewID = func() string { return "abcdef12" }
	if _, err := s.Add("buy milk", "urgent"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	_, err := s.RemoveTag("nonexistent-zzz", "urgent")
	if err == nil {
		t.Fatalf("RemoveTag: expected error")
	}
	var nm *ErrNoMatch
	if !errors.As(err, &nm) {
		t.Fatalf("error type = %T (%v); want *ErrNoMatch", err, err)
	}
}

func TestListTags_emptyCorpusReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	got, err := s.ListTags(FilterOpen)
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("ListTags on empty corpus = %v; want empty", got)
	}
}

func TestListTags_filterScopesWhichTasksAreScanned(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	s.Now = func() time.Time { return time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "11111111" }
	if _, err := s.Add("open task", "open-only", "shared"); err != nil {
		t.Fatalf("Add open: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "22222222" }
	if _, err := s.Add("closed task", "closed-only", "shared"); err != nil {
		t.Fatalf("Add closed: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC) }
	if _, err := s.Close("22222222"); err != nil {
		t.Fatalf("Close: %v", err)
	}

	openTags, err := s.ListTags(FilterOpen)
	if err != nil {
		t.Fatalf("ListTags(FilterOpen): %v", err)
	}
	if got := tagNames(openTags); !equalStrings(got, []string{"open-only", "shared"}) {
		t.Errorf("FilterOpen tag names = %v; want [open-only shared]", got)
	}

	closedTags, err := s.ListTags(FilterClosed)
	if err != nil {
		t.Fatalf("ListTags(FilterClosed): %v", err)
	}
	if got := tagNames(closedTags); !equalStrings(got, []string{"closed-only", "shared"}) {
		t.Errorf("FilterClosed tag names = %v; want [closed-only shared]", got)
	}

	allTags, err := s.ListTags(FilterAll)
	if err != nil {
		t.Fatalf("ListTags(FilterAll): %v", err)
	}
	gotAll := map[string]int{}
	for _, tc := range allTags {
		gotAll[tc.Name] = tc.Count
	}
	wantAll := map[string]int{"open-only": 1, "closed-only": 1, "shared": 2}
	for k, v := range wantAll {
		if gotAll[k] != v {
			t.Errorf("FilterAll count[%q] = %d; want %d (full=%v)", k, gotAll[k], v, allTags)
		}
	}
	if len(allTags) != len(wantAll) {
		t.Errorf("FilterAll len = %d; want %d", len(allTags), len(wantAll))
	}
}

func tagNames(tcs []TagCount) []string {
	out := make([]string, 0, len(tcs))
	for _, tc := range tcs {
		out = append(out, tc.Name)
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestListTags_returnsDistinctTagsWithCountsSortedByName(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	s.Now = func() time.Time { return time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "11111111" }
	if _, err := s.Add("first", "infra", "urgent"); err != nil {
		t.Fatalf("Add first: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "22222222" }
	if _, err := s.Add("second", "infra", "bug"); err != nil {
		t.Fatalf("Add second: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 4, 29, 11, 0, 0, 0, time.UTC) }
	s.NewID = func() string { return "33333333" }
	if _, err := s.Add("third", "infra"); err != nil {
		t.Fatalf("Add third: %v", err)
	}

	got, err := s.ListTags(FilterOpen)
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}

	want := []TagCount{
		{Name: "bug", Count: 1},
		{Name: "infra", Count: 3},
		{Name: "urgent", Count: 1},
	}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d; want %d (got=%v)", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("got[%d] = %v; want %v", i, got[i], w)
		}
	}
}
