# worktask

[![Go Reference](https://pkg.go.dev/badge/github.com/jvcorredor/worktask.svg)](https://pkg.go.dev/github.com/jvcorredor/worktask)
[![Go Report Card](https://goreportcard.com/badge/github.com/jvcorredor/worktask)](https://goreportcard.com/report/github.com/jvcorredor/worktask)
[![Release](https://img.shields.io/github/v/release/jvcorredor/worktask?label=release)](https://github.com/jvcorredor/worktask/releases)
[![Docs](https://img.shields.io/badge/docs-jvcorredor.github.io%2Fworktask-blue)](https://jvcorredor.github.io/worktask/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A Go CLI for task management designed primarily for LLM-agent consumption (Claude Code), usable as a normal terminal CLI.

## Install

### From script

```
curl -fsSL https://jvcorredor.github.io/worktask/install.sh | sh
```

### From source

```
go install github.com/jvcorredor/worktask@latest                    # from source
```

The binary lands in `$HOME/.local/bin` (or `$GOBIN` for `go install`) as `worktask`. See the [install reference](https://jvcorredor.github.io/worktask/install/) for env-var overrides and `$PATH` setup.

## Documentation

Full docs: <https://jvcorredor.github.io/worktask/>. The docs site is the canonical reference for subcommands, flags, JSON schema, file format, config keys, and agentic-usage patterns.
