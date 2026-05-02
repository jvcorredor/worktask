---
title: Reference slash command
description: Manage tasks via the worktask CLI
argument-hint: <add|list|show|complete|reopen|update|append|edit|research|research-log> [args...]
---

Manage tasks via the `worktask` CLI. Task state lives in per-task markdown files under the configured XDG data directory; this command never reads, parses, or rewrites `WORKING.md` or any task file directly. The CLI owns ID generation, slug generation, match resolution, and formatting.

## Dispatch

Parse `$ARGUMENTS` as `<subcommand> [args...]`. Shell out with `--format=json`:

| Subcommand | Shell |
|---|---|
| `add <description>` | `worktask --format=json add "<description>"` |
| `list` (also `--all`, `--closed`, `--limit N`) | `worktask --format=json list [flags]` |
| `show <frag>` | `worktask --format=json show "<frag>"` |
| `complete <frag>` | `worktask --format=json complete "<frag>"` |
| `reopen <frag>` | `worktask --format=json reopen "<frag>"` |
| `update <frag> <new description>` | `worktask --format=json update "<frag>" "<new description>"` |
| `append <frag> <text>` | `worktask --format=json append "<frag>" "<text>"` |
| `edit <frag>` | see below — do NOT shell out to `worktask edit` |
| `research [<frag>]` | see "research" below — invokes a background bash, returns an ack, reports back when done |
| `research-log <frag>` | `worktask research-log "<frag>"` |

Quote each argument as a single token. `<frag>` may be an ID, ID prefix, or description fragment; the CLI resolves it.

If the subcommand is missing or unrecognized, print the table and stop. Do not guess.

## Error handling

When the CLI exits non-zero, parse stdout as JSON and branch on the `error` field:

- `"error": "ambiguous"` — fragment matched multiple tasks. Show `matches` (each `{id, description}`) and ask the user which one. Do NOT pick. Do NOT retry with a longer fragment.
- `"error": "no_match"` — fragment matched nothing. Show `open_tasks` so the user can pick. Do NOT invent IDs or fabricate matches.
- Any other non-zero exit: surface stderr verbatim.

## Success output

- `list` and `show` return JSON. Render as a readable summary, or pass through.
- `add`, `update`, `append`, `complete`, `reopen` print a single line (e.g. `added 1e3900e4`). Pass through.

## `edit <frag>`

`worktask edit` launches `$EDITOR` via `syscall.Exec`, which is not usable from an agent shell. Instead:

1. Run `worktask --format=json show "<frag>"` to validate the fragment.
2. If ambiguous or no_match, handle per the rules above.
3. On a single match, tell the user: "Run `worktask edit <id>` from your shell to open it in your editor."

## `research [<frag>]`

`research` spawns one or more headless `claude -p` agents that perform read-only research and write findings back to the task body. It can take minutes per task. Run it as a background bash so the parent session stays responsive.

1. Invoke via the `Bash` tool with `run_in_background=true`:
   - Single task: `worktask research "<frag>"`
   - Sweep all open tasks (skip already-researched): `worktask research`
   - Pass through `--all`, `--stale=<dur>`, `--concurrency=N`, `--timeout=<dur>` flags if the user asked for them.
2. Reply to the user with an immediate ack (e.g. "queued research on `<frag>`, I'll report when it lands") and stop. Do NOT block, poll, or sleep.
3. The harness will notify you when the background bash exits. At that point read `BashOutput` for the run.
4. Parse the **last** JSON object on stdout. Two shapes are possible:
   - Single-task form (`{id, description, status, summary, log_path, error?}`): render one line. `status` is `findings`, `clarify`, or `failed`.
   - Batch form (`{batch_id, started, finished, results: [...], totals: {findings, clarify, failed, skipped}}`): render a per-status breakdown. Group results by `status`. Show counts from `totals`. List each result's `id`, `description`, and `summary`. Surface `error` for failed rows.
5. Streaming progress events (one JSON object per line, with an `event` field of `task_started` / `task_done` / `task_failed`) appear on stdout while the run is in flight. They are informational; the final structured summary is the only thing the user needs to see.

If the background bash exits non-zero, surface stderr verbatim alongside whatever partial output landed on stdout.

## User input

$ARGUMENTS
