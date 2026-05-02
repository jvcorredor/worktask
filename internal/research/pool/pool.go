// Package pool runs research tasks in bounded parallel, emits streaming
// progress events as they execute, and aggregates a final batch Summary.
//
// The pool is pure orchestration: it does not know about claude, prompts,
// log paths, or write-back side effects. Callers pre-build each Item with
// the rendered prompt and chosen log path, inject a RunFunc that executes
// one item, and consume Events as work happens.
package pool

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/jvcorredor/worktask/internal/research/runner"
	"github.com/jvcorredor/worktask/internal/task"
)

// Item is one unit of pool work. All fields are populated by the caller
// before queuing.
type Item struct {
	Task    task.Task
	Prompt  string
	LogPath string
}

// RunFunc executes one Item under the given context. The pool wraps ctx
// with a per-task timeout before calling.
type RunFunc func(ctx context.Context, item Item) (runner.Result, error)

// Opts configures the pool.
type Opts struct {
	Concurrency int
	Timeout     time.Duration
}

// Event is one streaming progress event.
type Event struct {
	Type    string // "task_started" | "task_done" | "task_failed"
	Task    task.Task
	Started time.Time
	Result  runner.Result // populated on task_done
	Err     error         // populated on task_failed
}

// Totals counts per-status outcomes across the batch.
type Totals struct {
	Findings int
	Clarify  int
	Failed   int
	Skipped  int
}

// TaskOutcome is one row of the final per-task summary.
type TaskOutcome struct {
	ID          string
	Description string
	Status      string
	Summary     string
	LogPath     string
	Tokens      int
	Error       string
}

// Summary is the final aggregate emitted after all work drains.
type Summary struct {
	BatchID  string
	Started  time.Time
	Finished time.Time
	Results  []TaskOutcome
	Totals   Totals
}

// Result is what Run hands back to the caller.
type Result struct {
	Events <-chan Event
	Done   <-chan Summary
}

// Run starts the pool. The Events channel closes once all work is done;
// after that, exactly one Summary lands on Done.
func Run(ctx context.Context, items []Item, runFn RunFunc, opts Opts) Result {
	events := make(chan Event)
	done := make(chan Summary, 1)

	go func() {
		started := time.Now().UTC()
		summary := Summary{
			BatchID: started.Format(time.RFC3339),
			Started: started,
		}

		concurrency := opts.Concurrency
		if concurrency < 1 {
			concurrency = 1
		}

		queue := make(chan Item)
		outcomes := make(chan TaskOutcome, len(items))
		var wg sync.WaitGroup
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for item := range queue {
					outcomes <- runOne(ctx, item, runFn, events, opts.Timeout)
				}
			}()
		}

		go func() {
			defer close(queue)
			for _, item := range items {
				queue <- item
			}
		}()

		go func() {
			wg.Wait()
			close(outcomes)
			close(events)
		}()

		for out := range outcomes {
			summary.Results = append(summary.Results, out)
			switch out.Status {
			case "findings":
				summary.Totals.Findings++
			case "clarify":
				summary.Totals.Clarify++
			case "failed":
				summary.Totals.Failed++
			}
		}

		summary.Finished = time.Now().UTC()
		done <- summary
		close(done)
	}()

	return Result{Events: events, Done: done}
}

func runOne(ctx context.Context, item Item, runFn RunFunc, events chan<- Event, timeout time.Duration) TaskOutcome {
	startedAt := time.Now().UTC()
	events <- Event{Type: "task_started", Task: item.Task, Started: startedAt}

	taskCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	result, err := runFn(taskCtx, item)
	if err != nil {
		events <- Event{Type: "task_failed", Task: item.Task, Started: startedAt, Err: err}
		return TaskOutcome{
			ID:          item.Task.ID,
			Description: firstLine(item.Task.Body),
			Status:      "failed",
			Summary:     err.Error(),
			LogPath:     item.LogPath,
			Error:       err.Error(),
		}
	}

	events <- Event{Type: "task_done", Task: item.Task, Started: startedAt, Result: result}
	return TaskOutcome{
		ID:          item.Task.ID,
		Description: firstLine(item.Task.Body),
		Status:      result.Status,
		Summary:     result.Summary,
		LogPath:     item.LogPath,
		Tokens:      result.Tokens,
	}
}

func firstLine(body string) string {
	body = strings.TrimRight(body, "\n")
	if i := strings.IndexByte(body, '\n'); i >= 0 {
		return body[:i]
	}
	return body
}
