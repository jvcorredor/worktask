# CLAUDE.md

## Documentation

The docs site at <https://jvcorredor.github.io/bytheway/> is the canonical source for everything user-visible about `btw` — subcommands, flags, JSON schema, file format, config keys, and the agentic-usage patterns. The site source lives in `docs/` and is published via `.github/workflows/docs.yml` on every push to `main` that touches `docs/**`.

`README.md` is a stub: tagline, install command, and a link to the docs site. Do not grow it back into a long-form reference. Anything user-visible belongs on the docs site.

Any PR that changes user-visible CLI surface — subcommands, flags, JSON schema, file format, or config keys — must update the corresponding docs page in the same PR. Same-PR is the rule because it is the only thing keeping the docs site from drifting out of sync with the CLI; there is no auto-generated content and no link-checking in CI.

The vendored reference slash command at `docs/src/content/docs/examples/btw-slash-command.md` is part of that contract. If a JSON schema or error-envelope change requires updating how an agent wraps the CLI, the slash-command file is updated in the same PR as the schema change.

## Local commands

This repo uses `just` as a task runner; install it once with `brew install just` (or your platform's equivalent). Beyond that the only dev dependencies are Go and Yarn.

Run `just --list` to discover the available recipes — every recipe carries a one-line description.

Run `just ci` before opening a Go PR; it runs `fmt-check`, `vet`, `test`, and `test-install` locally so failures surface before pushing.

## Go source-level docs

Exported Go symbols carry doc comments in the standard Go convention: a complete sentence that begins with the identifier name. Every package has a package doc comment that explains what it does and any contracts (on-disk format, stable schema, side effects).

`go doc <pkg>` is the preferred way to get a package overview — it reads the package and symbol comments straight from source, so it stays current as the code changes.

`golangci-lint run` enforces this via `revive`'s `exported` rule and `staticcheck`'s `ST1000`. Run it before opening a PR; CI runs the same gate. This rule is about internal-developer docs; user-visible CLI surface continues to live on the docs site per the rule above.

## Agent skills

### Issue tracker

GitHub Issues at `jvcorredor/bytheway`, accessed via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical roles using their default label strings (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `CONTEXT.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.
