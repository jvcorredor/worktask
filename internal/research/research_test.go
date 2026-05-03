package research

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jvcorredor/worktask/internal/research/runner"
	"github.com/jvcorredor/worktask/internal/store"
	"github.com/jvcorredor/worktask/internal/task"
)

const fixedTemplate = "Research the following task:\n\n{{.Body}}"

func TestRunOne_findings_appliesWriteback(t *testing.T) {
	dir := t.TempDir()
	worklog := filepath.Join(dir, "worklog.md")
	s := newStore(t, dir)

	added, err := s.Add("buy milk")
	if err != nil {
		t.Fatalf("seed Add: %v", err)
	}

	clock := fixedClock(2026, 5, 3, 12, 0, 0)
	cfg := Config{TasksDir: dir, WorkingLogPath: worklog, Model: "claude-sonnet-4-6"}

	var gotInput runner.RunInput
	run := func(_ context.Context, in runner.RunInput) (runner.Result, error) {
		gotInput = in
		return runner.Result{
			Status:  "findings",
			Summary: "got context",
			Body:    "Detailed findings here.",
			Tokens:  1234,
		}, nil
	}

	c, err := New(s, run, cfg, fixedTemplate, WithClock(clock.Now))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := c.RunOne(context.Background(), added)
	if err != nil {
		t.Fatalf("RunOne: %v", err)
	}

	if out.Status != "findings" {
		t.Errorf("Outcome.Status = %q; want findings", out.Status)
	}
	if out.Summary != "got context" {
		t.Errorf("Outcome.Summary = %q; want %q", out.Summary, "got context")
	}
	if out.Tokens != 1234 {
		t.Errorf("Outcome.Tokens = %d; want 1234", out.Tokens)
	}
	if out.Error != "" {
		t.Errorf("Outcome.Error = %q; want empty on findings", out.Error)
	}

	wantLogRel := filepath.Join("research-logs", added.ID+"_2026-05-03T12-00-00.jsonl")
	if out.LogPath != wantLogRel {
		t.Errorf("Outcome.LogPath = %q; want %q", out.LogPath, wantLogRel)
	}
	wantPrompt := "Research the following task:\n\nbuy milk\n"
	if gotInput.Prompt != wantPrompt {
		t.Errorf("RunFunc.Prompt = %q; want %q", gotInput.Prompt, wantPrompt)
	}
	if !strings.HasSuffix(gotInput.LogPath, wantLogRel) {
		t.Errorf("RunFunc.LogPath = %q; want suffix %q", gotInput.LogPath, wantLogRel)
	}
	if gotInput.Model != "claude-sonnet-4-6" {
		t.Errorf("RunFunc.Model = %q; want claude-sonnet-4-6", gotInput.Model)
	}

	reread, _, _, err := s.Get(added.ID)
	if err != nil {
		t.Fatalf("Get after writeback: %v", err)
	}
	if !reread.LastResearched.Equal(clock.times[0]) {
		t.Errorf("LastResearched = %v; want %v", reread.LastResearched, clock.times[0])
	}
	if reread.LastResearchLog != wantLogRel {
		t.Errorf("LastResearchLog = %q; want %q", reread.LastResearchLog, wantLogRel)
	}
	if !strings.Contains(reread.Body, "## Research 2026-05-03T12:00:00Z") {
		t.Errorf("body missing Research section header; got:\n%s", reread.Body)
	}
	if !strings.Contains(reread.Body, "Detailed findings here.") {
		t.Errorf("body missing rendered findings body; got:\n%s", reread.Body)
	}

	wlBytes, err := os.ReadFile(worklog)
	if err != nil {
		t.Fatalf("worklog read: %v", err)
	}
	if !strings.Contains(string(wlBytes), "research: got context") {
		t.Errorf("worklog missing summary line; got:\n%s", string(wlBytes))
	}
}

func TestRunOne_failed_skipsWriteback(t *testing.T) {
	dir := t.TempDir()
	worklog := filepath.Join(dir, "worklog.md")
	s := newStore(t, dir)

	added, err := s.Add("debug login")
	if err != nil {
		t.Fatalf("seed Add: %v", err)
	}
	bodyBefore := added.Body
	lastBefore := added.LastResearched

	clock := fixedClock(2026, 5, 3, 12, 0, 0)
	cfg := Config{TasksDir: dir, WorkingLogPath: worklog}

	run := func(_ context.Context, _ runner.RunInput) (runner.Result, error) {
		return runner.Result{}, errors.New("claude exited 1")
	}

	c, err := New(s, run, cfg, fixedTemplate, WithClock(clock.Now))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := c.RunOne(context.Background(), added)
	if err != nil {
		t.Fatalf("RunOne: %v", err)
	}

	if out.Status != "failed" {
		t.Errorf("Outcome.Status = %q; want failed", out.Status)
	}
	if out.Error == "" {
		t.Errorf("Outcome.Error empty; want runner error message")
	}
	if !strings.Contains(out.Summary, "claude exited 1") {
		t.Errorf("Outcome.Summary = %q; want it to contain runner error", out.Summary)
	}

	got, _, _, err := s.Get(added.ID)
	if err != nil {
		t.Fatalf("Get after failed run: %v", err)
	}
	if got.Body != bodyBefore {
		t.Errorf("body changed on failed run:\nbefore=%q\nafter =%q", bodyBefore, got.Body)
	}
	if !got.LastResearched.Equal(lastBefore) {
		t.Errorf("LastResearched changed on failed run: before=%v after=%v", lastBefore, got.LastResearched)
	}
	if got.LastResearchLog != "" {
		t.Errorf("LastResearchLog populated on failed run: %q", got.LastResearchLog)
	}
	if _, err := os.Stat(worklog); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("worklog should not exist after failed run; stat err = %v", err)
	}
}

func TestRunBatch_mixedOutcomes_emitsEventsAndAppliesWritebackPerStatus(t *testing.T) {
	dir := t.TempDir()
	worklog := filepath.Join(dir, "worklog.md")
	s := newStore(t, dir)

	a, err := s.Add("alpha task")
	if err != nil {
		t.Fatalf("seed Add a: %v", err)
	}
	b, err := s.Add("beta task")
	if err != nil {
		t.Fatalf("seed Add b: %v", err)
	}
	bBodyBefore := b.Body

	clock := fixedClock(2026, 5, 3, 12, 0, 0)
	cfg := Config{TasksDir: dir, WorkingLogPath: worklog}

	run := func(_ context.Context, in runner.RunInput) (runner.Result, error) {
		switch {
		case strings.Contains(in.Prompt, "alpha"):
			return runner.Result{
				Status:  "findings",
				Summary: "alpha summary",
				Body:    "alpha body",
				Tokens:  10,
			}, nil
		case strings.Contains(in.Prompt, "beta"):
			return runner.Result{}, errors.New("beta blew up")
		}
		t.Fatalf("unexpected prompt: %q", in.Prompt)
		return runner.Result{}, nil
	}

	c, err := New(s, run, cfg, fixedTemplate, WithClock(clock.Now))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	events, done, err := c.RunBatch(
		context.Background(),
		[]task.Task{a, b},
		2, // arbitrary skipped count
		BatchOpts{Concurrency: 2, Timeout: time.Second},
	)
	if err != nil {
		t.Fatalf("RunBatch: %v", err)
	}

	gotEvents := map[string]map[string]Event{} // id -> type -> event
	for ev := range events {
		if gotEvents[ev.ID] == nil {
			gotEvents[ev.ID] = map[string]Event{}
		}
		gotEvents[ev.ID][ev.Type] = ev
	}
	summary := <-done

	if _, ok := gotEvents[a.ID]["task_started"]; !ok {
		t.Errorf("missing task_started event for alpha (id=%s)", a.ID)
	}
	if e, ok := gotEvents[a.ID]["task_done"]; !ok {
		t.Errorf("missing task_done event for alpha (id=%s)", a.ID)
	} else {
		if e.Status != "findings" {
			t.Errorf("alpha task_done.Status = %q; want findings", e.Status)
		}
		if e.Summary != "alpha summary" {
			t.Errorf("alpha task_done.Summary = %q; want alpha summary", e.Summary)
		}
	}

	if _, ok := gotEvents[b.ID]["task_started"]; !ok {
		t.Errorf("missing task_started event for beta (id=%s)", b.ID)
	}
	if e, ok := gotEvents[b.ID]["task_failed"]; !ok {
		t.Errorf("missing task_failed event for beta (id=%s)", b.ID)
	} else if e.Err == nil || !strings.Contains(e.Err.Error(), "beta blew up") {
		t.Errorf("beta task_failed.Err = %v; want it to mention runner error", e.Err)
	}

	wantTotals := Totals{Findings: 1, Failed: 1, Skipped: 2}
	if summary.Totals != wantTotals {
		t.Errorf("Summary.Totals = %+v; want %+v", summary.Totals, wantTotals)
	}
	if len(summary.Results) != 2 {
		t.Errorf("len(Summary.Results) = %d; want 2", len(summary.Results))
	}

	gotA, _, _, err := s.Get(a.ID)
	if err != nil {
		t.Fatalf("Get alpha: %v", err)
	}
	if !strings.Contains(gotA.Body, "alpha body") {
		t.Errorf("alpha body missing writeback section; got:\n%s", gotA.Body)
	}
	if gotA.LastResearched.IsZero() {
		t.Errorf("alpha LastResearched not set after writeback")
	}

	gotB, _, _, err := s.Get(b.ID)
	if err != nil {
		t.Fatalf("Get beta: %v", err)
	}
	if gotB.Body != bBodyBefore {
		t.Errorf("beta body changed despite failed run:\nbefore=%q\nafter =%q", bBodyBefore, gotB.Body)
	}
	if !gotB.LastResearched.IsZero() {
		t.Errorf("beta LastResearched populated despite failed run: %v", gotB.LastResearched)
	}

	wlBytes, _ := os.ReadFile(worklog)
	if !strings.Contains(string(wlBytes), "research: alpha summary") {
		t.Errorf("worklog missing alpha summary; got:\n%s", string(wlBytes))
	}
	if strings.Contains(string(wlBytes), "research: beta") {
		t.Errorf("worklog should not contain beta entry on failed run; got:\n%s", string(wlBytes))
	}
}

func TestRunOne_writebackError_preservesSummary(t *testing.T) {
	dir := t.TempDir()
	// Create a regular file at the path appendWorklog would treat as a
	// directory parent. os.MkdirAll inside appendWorklog then fails,
	// which forces writeback.Apply to return a non-nil error after the
	// body section and frontmatter have already been written.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("seed blocker file: %v", err)
	}
	worklog := filepath.Join(blocker, "worklog.md")

	s := newStore(t, dir)
	added, err := s.Add("buy milk")
	if err != nil {
		t.Fatalf("seed Add: %v", err)
	}

	clock := fixedClock(2026, 5, 3, 12, 0, 0)
	cfg := Config{TasksDir: dir, WorkingLogPath: worklog}

	const agentSummary = "got context"
	run := func(_ context.Context, _ runner.RunInput) (runner.Result, error) {
		return runner.Result{
			Status:  "findings",
			Summary: agentSummary,
			Body:    "Detailed findings here.",
			Tokens:  1234,
		}, nil
	}

	c, err := New(s, run, cfg, fixedTemplate, WithClock(clock.Now))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := c.RunOne(context.Background(), added)
	if err != nil {
		t.Fatalf("RunOne: %v", err)
	}

	if out.Status != "failed" {
		t.Errorf("Outcome.Status = %q; want failed", out.Status)
	}
	if out.Summary != agentSummary {
		t.Errorf("Outcome.Summary = %q; want %q (agent's original summary preserved)", out.Summary, agentSummary)
	}
	if out.Error == "" {
		t.Errorf("Outcome.Error empty; want writeback error message")
	}
	if !strings.Contains(out.Error, "writeback") {
		t.Errorf("Outcome.Error = %q; want it to identify the writeback failure", out.Error)
	}
}

func TestRunBatch_renderFailure_doesNotAbortBatch(t *testing.T) {
	dir := t.TempDir()
	worklog := filepath.Join(dir, "worklog.md")
	s := newStore(t, dir)

	// Custom template references .Tags[0]; rendering fails at execute
	// time for any task whose Tags slice is empty. badTask has no tags;
	// goodTask has one. The failure is per-task, not per-batch.
	const tmpl = "Research: {{.Body}}{{index .Tags 0}}"

	badTask, err := s.Add("malformed task")
	if err != nil {
		t.Fatalf("seed bad task: %v", err)
	}
	goodTask, err := s.Add("healthy task", "x")
	if err != nil {
		t.Fatalf("seed good task: %v", err)
	}

	clock := fixedClock(2026, 5, 3, 12, 0, 0)
	cfg := Config{TasksDir: dir, WorkingLogPath: worklog}

	var ranIDs []string
	run := func(_ context.Context, in runner.RunInput) (runner.Result, error) {
		ranIDs = append(ranIDs, in.Prompt) // capture which prompts the runner saw
		return runner.Result{
			Status:  "findings",
			Summary: "good summary",
			Body:    "good body",
		}, nil
	}

	c, err := New(s, run, cfg, tmpl, WithClock(clock.Now))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	events, done, err := c.RunBatch(
		context.Background(),
		[]task.Task{badTask, goodTask},
		0,
		BatchOpts{Concurrency: 2, Timeout: time.Second},
	)
	if err != nil {
		t.Fatalf("RunBatch: %v", err)
	}

	gotEvents := map[string]map[string]Event{}
	for ev := range events {
		if gotEvents[ev.ID] == nil {
			gotEvents[ev.ID] = map[string]Event{}
		}
		gotEvents[ev.ID][ev.Type] = ev
	}
	summary := <-done

	failedEv, ok := gotEvents[badTask.ID]["task_failed"]
	if !ok {
		t.Fatalf("missing synthetic task_failed event for bad task (id=%s); got events=%+v", badTask.ID, gotEvents)
	}
	if failedEv.Err == nil {
		t.Errorf("task_failed.Err = nil; want render error")
	}

	if _, ok := gotEvents[goodTask.ID]["task_started"]; !ok {
		t.Errorf("missing task_started for good task (id=%s); other tasks must still run", goodTask.ID)
	}
	if doneEv, ok := gotEvents[goodTask.ID]["task_done"]; !ok {
		t.Errorf("missing task_done for good task (id=%s); other tasks must still run", goodTask.ID)
	} else if doneEv.Status != "findings" {
		t.Errorf("good task_done.Status = %q; want findings", doneEv.Status)
	}

	if len(ranIDs) != 1 {
		t.Errorf("runner invoked %d times; want exactly 1 (only the good task)", len(ranIDs))
	}

	if len(summary.Results) != 2 {
		t.Errorf("len(Summary.Results) = %d; want 2 (one failed + one findings)", len(summary.Results))
	}
	var foundBadFailed bool
	for _, r := range summary.Results {
		if r.ID == badTask.ID {
			foundBadFailed = true
			if r.Status != "failed" {
				t.Errorf("bad task Status = %q; want failed", r.Status)
			}
			if r.Error == "" {
				t.Errorf("bad task Error empty; want render error string")
			}
		}
	}
	if !foundBadFailed {
		t.Errorf("Summary.Results missing failed outcome for bad task")
	}
	if summary.Totals.Failed != 1 {
		t.Errorf("Totals.Failed = %d; want 1", summary.Totals.Failed)
	}
	if summary.Totals.Findings != 1 {
		t.Errorf("Totals.Findings = %d; want 1", summary.Totals.Findings)
	}
}

func TestRunBatch_skippedCount_forwardedVerbatim(t *testing.T) {
	dir := t.TempDir()
	worklog := filepath.Join(dir, "worklog.md")
	s := newStore(t, dir)

	added, err := s.Add("only task")
	if err != nil {
		t.Fatalf("seed Add: %v", err)
	}

	clock := fixedClock(2026, 5, 3, 12, 0, 0)
	cfg := Config{TasksDir: dir, WorkingLogPath: worklog}

	run := func(_ context.Context, _ runner.RunInput) (runner.Result, error) {
		return runner.Result{
			Status:  "findings",
			Summary: "summary",
			Body:    "body",
		}, nil
	}

	c, err := New(s, run, cfg, fixedTemplate, WithClock(clock.Now))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	const skipped = 7
	events, done, err := c.RunBatch(
		context.Background(),
		[]task.Task{added},
		skipped,
		BatchOpts{Concurrency: 1, Timeout: time.Second},
	)
	if err != nil {
		t.Fatalf("RunBatch: %v", err)
	}
	for range events {
	}
	summary := <-done

	if summary.Totals.Skipped != skipped {
		t.Errorf("Summary.Totals.Skipped = %d; want %d (forwarded verbatim from caller)", summary.Totals.Skipped, skipped)
	}
}

// --- helpers ---

func newStore(t *testing.T, dir string) *store.Store {
	t.Helper()
	s := store.New(dir)
	// Pin Now and NewID for stable test output.
	idCounter := 0
	s.NewID = func() string {
		idCounter++
		// Eight-char hex-ish IDs, distinct across calls in one test.
		return string([]byte{
			'a' + byte(idCounter%26),
			'a' + byte((idCounter/26)%26),
			'b', 'c', 'd', 'e', 'f', '0',
		})
	}
	addCounter := 0
	s.Now = func() time.Time {
		addCounter++
		return time.Date(2026, 4, 1, 9, addCounter, 0, 0, time.UTC)
	}
	return s
}

type sequenceClock struct {
	times []time.Time
	idx   int
}

func (c *sequenceClock) Now() time.Time {
	if c.idx >= len(c.times) {
		// Once exhausted, keep returning the last time. Tests should
		// configure enough clock ticks to cover the run.
		return c.times[len(c.times)-1]
	}
	t := c.times[c.idx]
	c.idx++
	return t
}

// fixedClock returns a clock that emits one fixed time. RunOne calls
// the clock once; RunBatch calls it once per task during prep. Tests
// that need distinct per-task times should build a clock with multiple
// entries inline.
func fixedClock(year int, mo time.Month, day, h, m, s int) *sequenceClock {
	return &sequenceClock{
		times: []time.Time{
			time.Date(year, mo, day, h, m, s, 0, time.UTC),
			time.Date(year, mo, day, h, m, s+1, 0, time.UTC),
			time.Date(year, mo, day, h, m, s+2, 0, time.UTC),
			time.Date(year, mo, day, h, m, s+3, 0, time.UTC),
		},
	}
}
