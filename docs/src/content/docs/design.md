---
title: Design notes
description: Why bytheway is shaped the way it is — out-of-scope items, internal package layout, and the migration tool.
---

`btw` is shaped around two readers: an LLM agent calling a thin slash command, and a human at a terminal. The agent gets stable JSON output with an explicit `error` discriminator so it can branch without parsing prose. The human gets one task per markdown file so `cat`, `grep`, `rg`, and `vim` work directly against the data directory without going through the binary. Most decisions below fall out of one of those two readers.

## Out of scope

By design, none of the following are in the tool. Add later only if a real need shows up; YAGNI for now.

- Tags, priority, due dates
- Cross-task relationships
- Full-text search
- Multi-machine sync
- Concurrent-write locking
- A `delete` subcommand
- Batch `close --all`
- Shell completion
- TUI or `fzf` picker

## Internal layout

```
cmd/         cobra subcommands, one file each, all thin
internal/
  config/    XDG-aware Load(), tasks_dir + editor
  id/        8-char hex from crypto/rand
  slug/      Generate(description, maxLen)
  task/      Task struct, Encode/Decode (frontmatter + body)
  store/     filesystem repository, match resolution, ErrAmbiguous/ErrNoMatch
  render/    human + JSON formatters, stable JSON schema
migrate/     one-shot WORKING.md → per-task-files port
main.go      entrypoint
```

The `cmd/` files are thin: each one parses arguments, calls into `internal/store`, and hands the result to `internal/render`. All match-resolution logic lives in `store`; all formatting (including the stable JSON schema documented in [Agentic usage](./agentic.md)) lives in `render`. Adding a subcommand is a new file in `cmd/` plus whatever store method it needs.

## Migration tool

`migrate/` is a separate `main` package that ports legacy `[task <id>]: ...` lines out of a single `WORKING.md` file into per-task files. It preserves original IDs and timestamps, leaves the freeform worklog narrative in place, and is intended to run once and then be ignored.

```
go run ./migrate /path/to/WORKING.md
```

It is folded into this page rather than promoted to its own nav entry because it is one-shot historical content — once the legacy `WORKING.md` has been ported, the binary has no further job.
