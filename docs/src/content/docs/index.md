---
title: worktask
description: A Go CLI for task management designed primarily for LLM-agent consumption (Claude Code), usable as a normal terminal CLI.
---

A Go CLI for task management designed primarily for LLM-agent consumption (Claude Code), usable as a normal terminal CLI.

Match resolution, ID generation, slug generation, and formatting all live in the CLI so the agent never has to read or rewrite task files itself. Tasks are plain markdown — `cat`, `grep`, `rg`, and `vim` work directly against the data directory without going through the binary.

## Install

```
go install github.com/jvcorredor/worktask@latest
```

The binary lands in `$GOBIN` as `worktask`. Alias to `wt` if you want a shorter handle.

## Where to go next

- [CLI reference](./cli.md) — every subcommand, flags, examples, and match-resolution behavior.
- [File & storage format](./storage.md) — data directory layout and the on-disk shape of a task file.
- [Configuration](./config.md) — `tasks_dir`, `editor`, XDG paths, and resolution order.
- [Agentic usage](./agentic.md) — JSON schema, error envelopes, and the slash-command dispatch pattern.
- [Design notes](./design.md) — out-of-scope items, internal package layout, and the migration tool.
