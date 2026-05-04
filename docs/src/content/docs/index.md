---
title: Capture and investigate fast inbounds
description: A Go CLI to capture and investigate fast inbounds — Slack threads, hallway asks, leadership questions — and hand them off as entrypoints for agentic work in Claude Code.
---

Bytheway is a Go CLI to capture and investigate fast inbounds — Slack threads, hallway asks, leadership questions — and turn them into entrypoints for agentic work in Claude Code. The funnel runs capture → soak → investigate → hand off; bytheway owns capture and investigation, while shaping and handoff are left to downstream agents.

The CLI is the agent-first surface: match resolution, ID generation, slug generation, and formatting all live in the binary so the agent never has to read or rewrite task files itself. Tasks are plain markdown — `cat`, `grep`, `rg`, and `vim` work directly against the data directory without going through the binary.

## Install

Pre-built binary (no Go toolchain required):

```
curl -fsSL https://jvcorredor.github.io/bytheway/install.sh | sh
```

Installs the latest release to `$HOME/.local/bin/bytheway`. Override the destination with `INSTALL_DIR=/usr/local/bin` or pin the release with `VERSION=0.2.0`. Full reference and verification details live on the [Install](./install.md) page.

From source (requires Go):

```
go install github.com/jvcorredor/bytheway@latest
```

The binary lands in `$GOBIN` as `btw`. Alias to `wt` if you want a shorter handle.

## Where to go next

- [Install](./install.md) — both install methods, env-var overrides, `$PATH` setup, checksum verification, macOS quarantine note.
- [CLI reference](./cli.md) — every subcommand, flags, examples, and match-resolution behavior.
- [File & storage format](./storage.md) — data directory layout and the on-disk shape of a task file.
- [Configuration](./config.md) — `tasks_dir`, `editor`, XDG paths, and resolution order.
- [Agentic usage](./agentic.md) — JSON schema, error envelopes, and the slash-command dispatch pattern.
- [Agentic recipes](./agentic-recipes.md) — capture / soak / investigate / hand off as recipes against existing CLI primitives.
- [Releases](./releases.md) — Conventional Commits, pre-1.0 versioning policy, link to the GitHub Releases page.
- [Design notes](./design.md) — out-of-scope items, internal package layout, and the migration tool.
