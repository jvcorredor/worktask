# ADR 0002: Research coordinator

## Status

Accepted

## Context

`cmd/research.go` is the apex of the research subsystem: 287 lines that pull
in nine packages (`config`, `render`, `research/log`, `research/pool`,
`research/prompt`, `research/runner`, `research/writeback`, `store`, plus
several stdlib) and stitch them together by hand. Two parallel pipelines
live in the file — `runSingleResearch` and `runBatchResearch` — and they
duplicate four steps each: prompt-template loading, log-path
computation (absolute + relative), `runner.Run` + `writeback.Apply`
invocation, and rendering.

The duplication is not the worst part. The worst part is `itemMeta`: a
sidecar map keyed by task ID, defined inside `cmd/`, that exists solely
because `pool.Item.LogPath` carries the relative path (for display in
`pool.TaskOutcome`) while `runner.RunInput.LogPath` needs the absolute
path. The closure that bridges pool and runner reaches into the sidecar
on every dispatch. The construct is a workaround caused by orchestration
state living in `cmd/`, above the layer that should own it.

Tracking issue [#85](https://github.com/jvcorredor/bytheway/issues/85)
asked for a research spike: investigate whether a `ResearchCoordinator`
module — with injected dependencies and a clean test surface — can
absorb the duplication and dissolve `itemMeta`.

The research subsystem already has the right leaf packages. Each is
single-purpose and tested at its own level: `prompt` loads and renders,
`log` computes filenames, `runner` spawns the headless agent and parses
its output, `pool` runs items in bounded parallel and emits events,
`writeback` applies side effects to the store. What is missing is an
apex that composes them.

## Decision

Introduce a top-level `internal/research` package — package name
`research` — that exposes a `Coordinator` with two methods, `RunOne` and
`RunBatch`. The leaf subpackages stay where they are. `cmd/research.go`
shrinks to a thin shell that handles flag parsing, fragment resolution,
listing/partitioning, signal wiring, and JSON rendering of events and
outcomes.

### Layering

```
cmd/research.go         (flags, store lookups, signal handling, render)
        │
        ▼
internal/research       (Coordinator: composes the leaves)
        │
        ├─► internal/research/prompt
        ├─► internal/research/log
        ├─► internal/research/runner
        ├─► internal/research/pool
        └─► internal/research/writeback
                                 │
                                 ▼
                          internal/store
```

The empty top-level `internal/research/` directory becomes the apex of
the layered cake. Children name pieces; the parent names the whole
domain.

### Coordinator interface

The `Coordinator` is a struct with collaborators wired at construction
and two methods that take pre-resolved inputs.

`RunOne(ctx, t task.Task) (Outcome, error)` runs research on one
already-resolved task and returns an `Outcome` shaped to match
`render.ResearchRun` field-for-field.

`RunBatch(ctx, tasks []task.Task, skipped int, opts BatchOpts)
(<-chan Event, <-chan Summary, error)` mirrors the existing `pool.Run`
shape: events stream as work happens; one `Summary` lands on `Done` once
the events channel closes. The skipped count is passed through verbatim
to `Summary.Totals.Skipped`.

`cmd/` is the input boundary. It resolves fragments to `task.Task`
(handling user-friendly "did you mean?" hints itself), lists open tasks
and partitions via `partitionResearchCandidates`, owns
`signal.NotifyContext`, and pre-resolves the model + extra-tools list
from flag overrides + config. The Coordinator never touches the
`config`, `prompt` (other than `prompt.Render`), or `render` packages.

### Injection seams

Three things are injected; everything else is a free-function call.

`RunFunc func(ctx, runner.RunInput) (runner.Result, error)` is the
runner-shaped seam. Production wires `runner.Run` with `runner.OSExec{}`;
tests pass a pure in-memory func that returns canned `runner.Result`
values. We inject one level *above* `runner.Exec` (the seam suggested by
the spike issue) because `runner.Run` itself does file I/O —
`os.MkdirAll`, `os.Create`, scanner-buffered stream-json writes — and
injecting at that level keeps the Coordinator's tests pure in-memory.
`runner.Run` is unit-tested at its own level; the Coordinator's tests
verify wiring, not log-file scanning. This precedent already exists in
`pool`, which injects `pool.RunFunc` for exactly the same reason.

`*store.Store` is injected as a concrete pointer. Tests construct one
under `t.TempDir()`, matching the pattern in `internal/store/store_test.go`.
No narrow store interface is introduced — `store` is a one-implementation
package and YAGNI applies until a second backend ever shows up.

`Clock func() time.Time` is an optional `Option` (defaulting to
`time.Now().UTC`) so per-task `runStart` timestamps — which determine
log filenames, frontmatter `last_researched`, and worklog lines — are
deterministic in tests.

The prompt template is loaded *by `cmd/`* and passed to `New(...)` as a
string. The Coordinator imports neither the `prompt` package's `Load`
nor the `config` package; it calls `prompt.Render(template, t)` per
task. Tests pass a literal template string with no filesystem touch.

### Outcome and event types are owned by the research package

Coordinator returns `research.Outcome`, `research.Event`, and
`research.Summary` — distinct types defined in the new package, not type
aliases for or re-exports of `pool.Event` / `pool.TaskOutcome` /
`pool.Summary`. The internal goroutine that drains pool's events
translates pool's types into research's types before emitting.

Reasoning: the entire purpose of putting `research` on top of `pool`
is to give `cmd/` a single orchestration package to depend on. If
`cmd/` reads `pool.Event` off a `research.Coordinator` channel, the
new layer is leaky — pool is still part of cmd/'s API surface. Distinct
types make the boundary real. The cost is one struct copy per event
per task, which is negligible.

The fields of `research.Event` mirror what `render.ResearchEvent`
actually consumes — `Type`, `ID`, `Description`, `Started`, `Status`,
`Summary`, `Err`. `LogPath` and `Tokens` live on `Outcome` and
`Summary.Results[]`, where the render layer expects them, but not on
the streamed `Event` (which today's render line schema doesn't carry).

### itemMeta dissolves

The Coordinator builds a private `prepared` map (keyed by task ID) at
the top of `RunBatch`. Each entry holds the precomputed `pool.Item`,
the absolute log path, and the per-task `runStart`. The closure passed
into `pool.Run` looks up its entry by `item.Task.ID` — the same
structural lookup the cmd-level `itemMeta` did, but no longer crossing
a package boundary. The map is now legitimate "precomputed per-task
inputs" owned by the layer that needs them; the workaround status
disappears.

`pool.Item` is left untouched. Adding `LogPathAbs` to `pool.Item` to
remove the lookup entirely would re-couple `pool` to filesystem-path
concerns, against its current "pure orchestration; does not know about
claude, prompts, log paths, or write-back side effects" docstring. The
re-coupling is reversible later if it ever proves worthwhile.

### Writeback runs inside the worker closure

`writeback.Apply` is invoked from inside the `RunFunc` closure each
pool worker calls — after `RunFunc` returns and before the closure
returns its `runner.Result` back to pool. Concurrency is safe: each
task's `s.Append` and `s.SetResearchMeta` write to that task's own
file (different tasks → different files), and `appendWorklog` uses
`O_APPEND` on a shared file with sub-PIPE_BUF lines, which POSIX
guarantees atomic across processes.

Doing writeback inside the closure (rather than in a separate
event-pump goroutine that consumes `pool.Event`s) means the closure
already has `prepared[item.Task.ID]` in scope — no second map lookup
in a separate goroutine. It also means that by the time pool emits
`task_done`, the side effects are committed: the event represents a
fully-applied result, not a "we're about to write back" promise.

A writeback error becomes a `failed` outcome whose `Summary` is the
agent's original `result.Summary` and whose `Error` is the writeback
error message. Status is `failed`. This is a small intentional
behavior change from today's batch path, which silently swallows
writeback errors via `_ = writeback.Apply(...)`. See Consequences.

### Per-task render failures don't abort the batch

If `prompt.Render` fails for any task during the prep loop, the
Coordinator emits a synthetic `task_failed` event for that task,
records a failed outcome in the summary, and continues with the rest
of the batch. The current behavior is to return early from
`runBatchResearch` and abort the whole sweep — also flagged as an
intentional change in Consequences.

## Consequences

- `cmd/research.go` shrinks dramatically. Most of what it does today
  becomes a Coordinator concern; what stays is genuinely cmd-shaped:
  flag parsing, store lookups, signal wiring, JSON rendering.
- `itemMeta` is no longer a workaround. The same map structure exists
  one layer down, where it represents normal precomputed state owned by
  the orchestrator that needs it.
- The `internal/research` package becomes the only research-facing
  import for `cmd/`. The leaf subpackages stay public-within-internal so
  other consumers (none today) can still wire them directly if needed.
- **Behavior change: writeback errors surface as failed outcomes.**
  The current batch path silently swallows them; the new path turns
  them into `failed` results so a "research run reported success but
  the task body wasn't updated" silently doesn't happen. Single-task
  callers (`btw research <fragment>`) see the writeback error
  reflected in the JSON line's `error` field.
- **Behavior change: per-task render failures no longer abort the
  batch.** A malformed body that breaks the template for one task no
  longer poisons the other 49. The failing task lands in the summary
  as a failed outcome with the render error.
- The Coordinator's tests stay pure in-memory because `RunFunc` is the
  injection seam. `runner.Run` continues to be tested at its own level.
- The leaf packages — `prompt`, `log`, `runner`, `pool`, `writeback` —
  are unchanged. The spike does not touch their interfaces.

## References

- Spike issue: [#85](https://github.com/jvcorredor/bytheway/issues/85)
- Prototype branch: `bytheway-85-research-coordinator-spike` —
  contains the Coordinator package and three indicative tests
  (`TestRunOne_findings_appliesWriteback`,
  `TestRunOne_failed_skipsWriteback`,
  `TestRunBatch_mixedOutcomes_emitsEventsAndAppliesWritebackPerStatus`).
  The branch does *not* yet rewrite `cmd/research.go`; that
  extraction is a follow-up that consumes this prototype.
