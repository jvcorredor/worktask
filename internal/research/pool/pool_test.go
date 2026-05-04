package pool

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jvcorredor/bytheway/internal/research/runner"
	"github.com/jvcorredor/bytheway/internal/task"
)

func TestRun_emptyInputClosesEventsAndReturnsZeroSummary(t *testing.T) {
	res := Run(context.Background(), nil, nil, Opts{Concurrency: 3, Timeout: time.Minute})

	for ev := range res.Events {
		t.Errorf("unexpected event on empty input: %+v", ev)
	}

	summary := <-res.Done
	if len(summary.Results) != 0 {
		t.Errorf("Results = %d; want 0", len(summary.Results))
	}
	if summary.Totals != (Totals{}) {
		t.Errorf("Totals = %+v; want zero value", summary.Totals)
	}
	if summary.Started.IsZero() {
		t.Errorf("Started must be set even on empty batch")
	}
	if summary.Finished.IsZero() {
		t.Errorf("Finished must be set even on empty batch")
	}
	if summary.BatchID == "" {
		t.Errorf("BatchID must be set")
	}
}

func TestRun_singleSuccessEmitsStartedAndDoneEvents(t *testing.T) {
	tk := taskWith("aaaaaaaa", "task one\nbody\n")
	item := Item{
		Task:    tk,
		Prompt:  "rendered",
		LogPath: "/tmp/aaaaaaaa.jsonl",
	}
	runFn := stubRunFn(map[string]runner.Result{
		"aaaaaaaa": {Status: "findings", Summary: "found context", Tokens: 1234},
	})

	res := Run(context.Background(), []Item{item}, runFn, Opts{Concurrency: 1, Timeout: time.Second})

	var events []Event
	for ev := range res.Events {
		events = append(events, ev)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events; want 2 (started + done): %+v", len(events), events)
	}
	if events[0].Type != "task_started" || events[0].Task.ID != "aaaaaaaa" {
		t.Errorf("event[0] = %+v; want task_started for aaaaaaaa", events[0])
	}
	if events[0].Started.IsZero() {
		t.Errorf("task_started must carry a non-zero Started time")
	}
	if events[1].Type != "task_done" || events[1].Result.Status != "findings" {
		t.Errorf("event[1] = %+v; want task_done with findings", events[1])
	}
	if events[1].Result.Summary != "found context" {
		t.Errorf("event[1].Result.Summary = %q; want %q", events[1].Result.Summary, "found context")
	}

	summary := <-res.Done
	if len(summary.Results) != 1 {
		t.Fatalf("len(Results) = %d; want 1", len(summary.Results))
	}
	out := summary.Results[0]
	if out.ID != "aaaaaaaa" {
		t.Errorf("Results[0].ID = %q; want aaaaaaaa", out.ID)
	}
	if out.Description != "task one" {
		t.Errorf("Results[0].Description = %q; want %q", out.Description, "task one")
	}
	if out.Status != "findings" {
		t.Errorf("Results[0].Status = %q; want findings", out.Status)
	}
	if out.Summary != "found context" {
		t.Errorf("Results[0].Summary = %q; want %q", out.Summary, "found context")
	}
	if out.LogPath != "/tmp/aaaaaaaa.jsonl" {
		t.Errorf("Results[0].LogPath = %q; want /tmp/aaaaaaaa.jsonl", out.LogPath)
	}
	if out.Tokens != 1234 {
		t.Errorf("Results[0].Tokens = %d; want 1234", out.Tokens)
	}
	if summary.Totals != (Totals{Findings: 1}) {
		t.Errorf("Totals = %+v; want {Findings: 1}", summary.Totals)
	}
}

func TestRun_concurrencyBoundLimitsInFlightWorkers(t *testing.T) {
	const n = 10
	const limit = 3

	items := make([]Item, n)
	for i := range items {
		items[i] = Item{
			Task:    taskWith(strID(i), "task\n"),
			Prompt:  "p",
			LogPath: "/tmp/x",
		}
	}

	var (
		mu       sync.Mutex
		inFlight int
		maxSeen  int
	)
	runFn := func(_ context.Context, _ Item) (runner.Result, error) {
		mu.Lock()
		inFlight++
		if inFlight > maxSeen {
			maxSeen = inFlight
		}
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		mu.Lock()
		inFlight--
		mu.Unlock()
		return runner.Result{Status: "findings", Summary: "ok"}, nil
	}

	res := Run(context.Background(), items, runFn, Opts{Concurrency: limit, Timeout: time.Second})
	for range res.Events {
	}
	summary := <-res.Done

	if maxSeen > limit {
		t.Errorf("max concurrent in-flight = %d; want <= %d", maxSeen, limit)
	}
	if maxSeen < limit {
		t.Errorf("max concurrent in-flight = %d; want = %d (pool should saturate the bound)", maxSeen, limit)
	}
	if len(summary.Results) != n {
		t.Errorf("len(Results) = %d; want %d", len(summary.Results), n)
	}
	if summary.Totals.Findings != n {
		t.Errorf("Totals.Findings = %d; want %d", summary.Totals.Findings, n)
	}
}

func TestRun_perTaskTimeoutMarksHangAsFailed(t *testing.T) {
	item := Item{Task: taskWith("aaaaaaaa", "task\n"), Prompt: "p", LogPath: "/tmp/x"}

	runFn := func(ctx context.Context, _ Item) (runner.Result, error) {
		<-ctx.Done()
		return runner.Result{}, ctx.Err()
	}

	start := time.Now()
	res := Run(context.Background(), []Item{item}, runFn, Opts{Concurrency: 1, Timeout: 50 * time.Millisecond})

	var failed Event
	for ev := range res.Events {
		if ev.Type == "task_failed" {
			failed = ev
		}
	}
	summary := <-res.Done
	elapsed := time.Since(start)

	if elapsed > 500*time.Millisecond {
		t.Errorf("elapsed = %v; want pool to bail out close to the 50ms timeout", elapsed)
	}
	if failed.Type != "task_failed" {
		t.Fatalf("never got task_failed event; got summary %+v", summary)
	}
	if failed.Err == nil || !errors.Is(failed.Err, context.DeadlineExceeded) {
		t.Errorf("Err = %v; want context.DeadlineExceeded", failed.Err)
	}
	if summary.Totals.Failed != 1 {
		t.Errorf("Totals.Failed = %d; want 1", summary.Totals.Failed)
	}
	if len(summary.Results) != 1 || summary.Results[0].Status != "failed" {
		t.Errorf("Results = %+v; want one failed outcome", summary.Results)
	}
}

func TestRun_parentCancelDrainsInFlightAndStillEmitsSummary(t *testing.T) {
	const n = 5
	items := make([]Item, n)
	for i := range items {
		items[i] = Item{Task: taskWith(strID(i), "task\n"), Prompt: "p", LogPath: "/tmp/x"}
	}

	runFn := func(ctx context.Context, _ Item) (runner.Result, error) {
		<-ctx.Done()
		return runner.Result{}, ctx.Err()
	}

	ctx, cancel := context.WithCancel(context.Background())
	res := Run(ctx, items, runFn, Opts{Concurrency: 2, Timeout: time.Hour})

	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	for range res.Events {
	}
	summary := <-res.Done

	if summary.Finished.IsZero() {
		t.Errorf("Summary.Finished must be set after cancel")
	}
	if summary.Totals.Findings != 0 {
		t.Errorf("Totals.Findings = %d; want 0 (nothing should have completed cleanly)", summary.Totals.Findings)
	}
	for _, r := range summary.Results {
		if r.Status != "failed" {
			t.Errorf("outcome %s status = %q; want failed under cancel", r.ID, r.Status)
		}
		if r.Error == "" {
			t.Errorf("outcome %s missing error message", r.ID)
		}
	}
}

func TestRun_partialFailureReportsEachOutcomeCorrectly(t *testing.T) {
	items := []Item{
		{Task: taskWith("a0000000", "ok findings\n"), Prompt: "p", LogPath: "/tmp/a"},
		{Task: taskWith("b0000000", "needs clarify\n"), Prompt: "p", LogPath: "/tmp/b"},
		{Task: taskWith("c0000000", "explicit failed status\n"), Prompt: "p", LogPath: "/tmp/c"},
		{Task: taskWith("d0000000", "exec error\n"), Prompt: "p", LogPath: "/tmp/d"},
	}

	runFn := func(_ context.Context, item Item) (runner.Result, error) {
		switch item.Task.ID {
		case "a0000000":
			return runner.Result{Status: "findings", Summary: "ok"}, nil
		case "b0000000":
			return runner.Result{Status: "clarify", Summary: "what is X"}, nil
		case "c0000000":
			return runner.Result{Status: "failed", Summary: "agent self-reported failure"}, nil
		case "d0000000":
			return runner.Result{}, errors.New("claude exited 1")
		}
		t.Fatalf("unexpected item %s", item.Task.ID)
		return runner.Result{}, nil
	}

	res := Run(context.Background(), items, runFn, Opts{Concurrency: 2, Timeout: time.Second})
	for range res.Events {
	}
	summary := <-res.Done

	want := Totals{Findings: 1, Clarify: 1, Failed: 2}
	if summary.Totals != want {
		t.Errorf("Totals = %+v; want %+v", summary.Totals, want)
	}

	byID := map[string]TaskOutcome{}
	for _, r := range summary.Results {
		byID[r.ID] = r
	}
	if byID["a0000000"].Status != "findings" {
		t.Errorf("a status = %q", byID["a0000000"].Status)
	}
	if byID["b0000000"].Status != "clarify" {
		t.Errorf("b status = %q", byID["b0000000"].Status)
	}
	if byID["c0000000"].Status != "failed" {
		t.Errorf("c status = %q (agent-reported)", byID["c0000000"].Status)
	}
	if byID["d0000000"].Status != "failed" {
		t.Errorf("d status = %q (exec error)", byID["d0000000"].Status)
	}
	if byID["d0000000"].Error == "" {
		t.Errorf("d outcome missing exec error message")
	}
}

func strID(i int) string {
	return string([]byte{byte('a' + i%26), byte('a' + (i/26)%26), 'b', 'c', 'd', 'e', 'f', '0'})
}

// stubRunFn returns a RunFunc that maps task IDs to fixed results.
func stubRunFn(byID map[string]runner.Result) RunFunc {
	return func(_ context.Context, item Item) (runner.Result, error) {
		return byID[item.Task.ID], nil
	}
}

func taskWith(id, body string) task.Task {
	return task.Task{ID: id, Body: body}
}
