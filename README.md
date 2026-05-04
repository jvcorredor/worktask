# bytheway

[![Go Reference](https://pkg.go.dev/badge/github.com/jvcorredor/bytheway.svg)](https://pkg.go.dev/github.com/jvcorredor/bytheway)
[![Go Report Card](https://goreportcard.com/badge/github.com/jvcorredor/bytheway)](https://goreportcard.com/report/github.com/jvcorredor/bytheway)
[![Release](https://img.shields.io/github/v/release/jvcorredor/bytheway?label=release)](https://github.com/jvcorredor/bytheway/releases)
[![Docs](https://img.shields.io/badge/docs-jvcorredor.github.io%2Fbytheway-blue)](https://jvcorredor.github.io/bytheway/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A Go CLI to capture and investigate fast inbounds — Slack threads, hallway asks, leadership questions — and turn them into entrypoints for agentic work in Claude Code.

## Install

### From script

```
curl -fsSL https://jvcorredor.github.io/bytheway/install.sh | sh
```

### From source

```
go install github.com/jvcorredor/bytheway@latest
```

The binary lands in `$HOME/.local/bin` (or `$GOBIN` for `go install`) as `btw`. See the [install reference](https://jvcorredor.github.io/bytheway/install/) for env-var overrides and `$PATH` setup.

## Documentation

Full docs: <https://jvcorredor.github.io/bytheway/>. The docs site is the canonical reference for subcommands, flags, JSON schema, file format, config keys, and agentic-usage patterns.
