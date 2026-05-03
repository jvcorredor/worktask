// Package research is the top-level orchestration layer for headless
// research runs. It composes the leaf subpackages — prompt, log,
// runner, pool, writeback — and exposes a [Coordinator] with two
// methods: [Coordinator.RunOne] for a single task and
// [Coordinator.RunBatch] for a streaming batch.
//
// Inputs are pre-resolved at the cmd boundary: callers hand in the
// already-loaded prompt template, already-resolved [task.Task] values,
// and a runner-shaped [RunFunc] seam so tests stay pure in-memory.
// The package itself imports neither config nor render.
//
// See docs/adr/0002-research-coordinator.md for the design rationale.
package research

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	researchlog "github.com/jvcorredor/worktask/internal/research/log"
	"github.com/jvcorredor/worktask/internal/research/pool"
	"github.com/jvcorredor/worktask/internal/research/prompt"
	"github.com/jvcorredor/worktask/internal/research/runner"
	"github.com/jvcorredor/worktask/internal/research/writeback"
	"github.com/jvcorredor/worktask/internal/store"
	"github.com/jvcorredor/worktask/internal/task"
)

// RunFunc is the runner-shaped seam the [Coordinator] depends on.
// Production wires [runner.Run] with [runner.OSExec]; tests inject a
// func that returns canned [runner.Result] values without touching the
// filesystem.
type RunFunc func(ctx context.Context, in runner.RunInput) (runner.Result, error)

// Config carries the fully-resolved settings the [Coordinator] needs.
// All values are pre-resolved at the cmd boundary — Model and
// ExtraTools have already absorbed flag overrides, and the prompt
// template is loaded from disk and passed to [New] separately.
type Config struct {
	// TasksDir is the absolute path to the worktask tasks directory.
	// Used by [researchlog.Path] to build per-task log filenames.
	TasksDir string
	// WorkingLogPath is the absolute path of the worklog file the
	// writeback layer appends "research:" lines to. Empty disables
	// the worklog append.
	WorkingLogPath string
	// Model is the resolved model id passed to the runner. The cmd
	// layer applies --model > config.research_model precedence before
	// constructing the Coordinator.
	Model string
	// ExtraTools is the resolved, pre-validated extra-tools list passed
	// to the runner. Validation happens in the config package; by the
	// time it lands here it is known-clean.
	ExtraTools []string
}

// Option is a functional option for [New].
type Option func(*Coordinator)

// WithClock injects a clock function. Tests use this to make per-task
// run-start times — and therefore log filenames, frontmatter
// last_researched, and worklog line timestamps — deterministic.
// Defaults to time.Now in UTC.
func WithClock(clock func() time.Time) Option {
	return func(c *Coordinator) { c.clock = clock }
}

// Coordinator orchestrates one research run ([Coordinator.RunOne]) or
// one batch ([Coordinator.RunBatch]). Collaborators are wired at
// construction time. A Coordinator instance is intended to live for
// the duration of one CLI invocation; reuse across invocations is
// safe but unnecessary.
type Coordinator struct {
	store    *store.Store
	run      RunFunc
	cfg      Config
	template string
	clock    func() time.Time
}

// New constructs a Coordinator. The template is the loaded prompt
// template body — the cmd layer is responsible for reading it from
// disk via [prompt.Load] before calling New, so the Coordinator never
// imports the prompt-load path. The error return is reserved for
// future construction-time validation; today New does not fail.
func New(s *store.Store, run RunFunc, cfg Config, template string, opts ...Option) (*Coordinator, error) {
	c := &Coordinator{
		store:    s,
		run:      run,
		cfg:      cfg,
		template: template,
		clock:    func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// Outcome is the result of one [Coordinator.RunOne] call, shaped to
// map field-by-field onto render.ResearchRun at the cmd layer.
type Outcome struct {
	ID          string
	Description string
	Status      string
	Summary     string
	LogPath     string // relative to Config.TasksDir
	Tokens      int
	Error       string // populated on failed status
}

// Event is one streaming progress event emitted by
// [Coordinator.RunBatch]. Field shape matches render.ResearchEvent.
type Event struct {
	Type        string // "task_started" | "task_done" | "task_failed"
	ID          string
	Description string    // populated on task_started
	Started     time.Time // populated on task_started
	Status      string    // populated on task_done
	Summary     string    // populated on task_done
	Err         error     // populated on task_failed
}

// Totals counts per-status outcomes plus the skipped-at-prep count.
type Totals struct {
	Findings, Clarify, Failed, Skipped int
}

// Summary is the final aggregate emitted on the Done channel after
// [Coordinator.RunBatch] drains.
type Summary struct {
	BatchID  string
	Started  time.Time
	Finished time.Time
	Results  []Outcome
	Totals   Totals
}

// BatchOpts configures one [Coordinator.RunBatch] invocation.
type BatchOpts struct {
	// Concurrency caps in-flight workers. Values below 1 are clamped
	// to 1 by the underlying pool.
	Concurrency int
	// Timeout is the per-task hard timeout. Zero disables.
	Timeout time.Duration
}

// RunOne runs research on a single resolved task. The body is rendered
// against the loaded template, the runner is invoked, and on a
// non-failed status writeback is applied (Research section appended,
// research-meta frontmatter set, worklog line appended). The returned
// Outcome maps field-by-field onto render.ResearchRun.
//
// A runner error becomes a failed Outcome carrying the error message;
// a writeback error after a non-failed run also becomes a failed
// Outcome — see ADR 0002 Consequences.
func (c *Coordinator) RunOne(ctx context.Context, t task.Task) (Outcome, error) {
	desc := firstLine(t.Body)

	rendered, err := prompt.Render(c.template, t)
	if err != nil {
		return Outcome{
			ID:          t.ID,
			Description: desc,
			Status:      "failed",
			Summary:     err.Error(),
			Error:       err.Error(),
		}, nil
	}

	runStart := c.clock()
	logPath := researchlog.Path(c.cfg.TasksDir, t.ID, runStart)
	logPathRel := relPath(c.cfg.TasksDir, logPath)

	result, runErr := c.run(ctx, runner.RunInput{
		Prompt:     rendered,
		LogPath:    logPath,
		Model:      c.cfg.Model,
		ExtraTools: c.cfg.ExtraTools,
	})
	if runErr != nil {
		result = runner.Result{
			Status:  "failed",
			Summary: runErr.Error(),
		}
	}

	if wbErr := c.writeBack(t.ID, result, runStart, logPath, logPathRel); wbErr != nil {
		return Outcome{
			ID:          t.ID,
			Description: desc,
			Status:      "failed",
			Summary:     result.Summary,
			LogPath:     logPathRel,
			Tokens:      result.Tokens,
			Error:       wbErr.Error(),
		}, nil
	}

	out := Outcome{
		ID:          t.ID,
		Description: desc,
		Status:      result.Status,
		Summary:     result.Summary,
		LogPath:     logPathRel,
		Tokens:      result.Tokens,
	}
	if result.Status == "failed" {
		out.Error = errString(runErr)
	}
	return out, nil
}

// preparedItem holds per-task inputs precomputed at the top of RunBatch.
// The map of these — keyed by task ID — is the legitimate internal
// state that subsumes the cmd-level itemMeta workaround (see ADR 0002).
type preparedItem struct {
	item       pool.Item
	absLogPath string
	runStart   time.Time
	desc       string
	renderErr  error // when non-nil, the task short-circuits to a failed outcome before pool sees it
}

// writebackFailure captures the agent's original summary and the
// writeback error message for one task whose post-run writeback failed.
// RunBatch consults this map after pool drains so the final Outcome
// preserves the agent's summary in Outcome.Summary while surfacing the
// writeback error via Outcome.Error.
type writebackFailure struct {
	originalSummary string
	originalTokens  int
	errMsg          string
}

// RunBatch runs research over a slice of pre-partitioned open tasks
// and returns two channels: a streaming Events channel that closes
// when work drains, and a Done channel that receives exactly one
// Summary after Events closes. The skipped count is forwarded
// verbatim to Summary.Totals.Skipped.
//
// Per-task failure modes:
//   - prompt.Render failure during prep: a task_failed event is
//     emitted with the render error; the task lands as a failed
//     outcome in the summary; other tasks continue. (Behavior change
//     from the prior cmd-level batch path; see ADR 0002.)
//   - RunFunc error: surfaced as task_failed by the underlying pool.
//   - writeback error after a non-failed run: turned into a failed
//     outcome that preserves the agent's original Summary and surfaces
//     the writeback error message via Outcome.Error. (Behavior change.)
//
// The caller owns ctx lifetime — wire signal.NotifyContext at the
// cmd boundary if SIGINT/SIGTERM cancel is desired.
func (c *Coordinator) RunBatch(ctx context.Context, tasks []task.Task, skipped int, opts BatchOpts) (<-chan Event, <-chan Summary, error) {
	prepared := make(map[string]preparedItem, len(tasks))
	items := make([]pool.Item, 0, len(tasks))
	var prepFailures []preparedItem
	var (
		wbMu       sync.Mutex
		wbFailures = make(map[string]writebackFailure)
	)

	for _, t := range tasks {
		runStart := c.clock()
		logPath := researchlog.Path(c.cfg.TasksDir, t.ID, runStart)
		logPathRel := relPath(c.cfg.TasksDir, logPath)
		desc := firstLine(t.Body)

		rendered, rerr := prompt.Render(c.template, t)
		if rerr != nil {
			prepFailures = append(prepFailures, preparedItem{
				item:       pool.Item{Task: t, LogPath: logPathRel},
				absLogPath: logPath,
				runStart:   runStart,
				desc:       desc,
				renderErr:  rerr,
			})
			continue
		}
		item := pool.Item{
			Task:    t,
			Prompt:  rendered,
			LogPath: logPathRel,
		}
		prepared[t.ID] = preparedItem{
			item:       item,
			absLogPath: logPath,
			runStart:   runStart,
			desc:       desc,
		}
		items = append(items, item)
	}

	runFn := func(ctx context.Context, item pool.Item) (runner.Result, error) {
		p := prepared[item.Task.ID]
		result, runErr := c.run(ctx, runner.RunInput{
			Prompt:     item.Prompt,
			LogPath:    p.absLogPath,
			Model:      c.cfg.Model,
			ExtraTools: c.cfg.ExtraTools,
		})
		if runErr != nil {
			return result, runErr
		}
		if wbErr := c.writeBack(item.Task.ID, result, p.runStart, p.absLogPath, item.LogPath); wbErr != nil {
			wbMu.Lock()
			wbFailures[item.Task.ID] = writebackFailure{
				originalSummary: result.Summary,
				originalTokens:  result.Tokens,
				errMsg:          wbErr.Error(),
			}
			wbMu.Unlock()
			return result, fmt.Errorf("writeback: %w", wbErr)
		}
		return result, nil
	}

	poolRes := pool.Run(ctx, items, runFn, pool.Opts{
		Concurrency: opts.Concurrency,
		Timeout:     opts.Timeout,
	})

	events := make(chan Event)
	done := make(chan Summary, 1)

	go func() {
		defer close(events)
		// Synthetic prep-failure events ride the same channel ahead of
		// pool's events. Order across the boundary is "all prep failures
		// first, then live pool events as they happen."
		for _, p := range prepFailures {
			events <- Event{
				Type:        "task_failed",
				ID:          p.item.Task.ID,
				Description: p.desc,
				Started:     p.runStart,
				Err:         p.renderErr,
			}
		}
		for ev := range poolRes.Events {
			events <- translateEvent(ev)
		}
	}()

	go func() {
		poolSummary := <-poolRes.Done
		s := Summary{
			BatchID:  poolSummary.BatchID,
			Started:  poolSummary.Started,
			Finished: poolSummary.Finished,
			Totals: Totals{
				Findings: poolSummary.Totals.Findings,
				Clarify:  poolSummary.Totals.Clarify,
				Failed:   poolSummary.Totals.Failed,
				Skipped:  skipped,
			},
		}
		for _, p := range prepFailures {
			s.Results = append(s.Results, Outcome{
				ID:          p.item.Task.ID,
				Description: p.desc,
				Status:      "failed",
				Summary:     p.renderErr.Error(),
				LogPath:     p.item.LogPath,
				Error:       p.renderErr.Error(),
			})
			s.Totals.Failed++
		}
		for _, r := range poolSummary.Results {
			out := Outcome{
				ID:          r.ID,
				Description: r.Description,
				Status:      r.Status,
				Summary:     r.Summary,
				LogPath:     r.LogPath,
				Tokens:      r.Tokens,
				Error:       r.Error,
			}
			if wb, ok := wbFailures[r.ID]; ok {
				out.Summary = wb.originalSummary
				out.Tokens = wb.originalTokens
				out.Error = wb.errMsg
			}
			s.Results = append(s.Results, out)
		}
		done <- s
		close(done)
	}()

	return events, done, nil
}

func (c *Coordinator) writeBack(fragment string, result runner.Result, runStart time.Time, logPath, logPathRel string) error {
	return writeback.Apply(c.store, writeback.Input{
		Fragment:       fragment,
		Result:         result,
		RunStart:       runStart,
		LogPath:        logPath,
		LogPathRel:     logPathRel,
		WorkingLogPath: c.cfg.WorkingLogPath,
	})
}

func translateEvent(ev pool.Event) Event {
	e := Event{
		Type:        ev.Type,
		ID:          ev.Task.ID,
		Description: firstLine(ev.Task.Body),
		Started:     ev.Started,
	}
	switch ev.Type {
	case "task_done":
		e.Status = ev.Result.Status
		e.Summary = ev.Result.Summary
	case "task_failed":
		e.Err = ev.Err
	}
	return e
}

func relPath(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return rel
}

func firstLine(body string) string {
	body = strings.TrimRight(body, "\n")
	if i := strings.IndexByte(body, '\n'); i >= 0 {
		return body[:i]
	}
	return body
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
