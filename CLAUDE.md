# CLAUDE.md

## Documentation

The docs site at <https://jvcorredor.github.io/worktask/> is the canonical source for everything user-visible about `worktask` — subcommands, flags, JSON schema, file format, config keys, and the agentic-usage patterns. The site source lives in `docs/` and is published via `.github/workflows/docs.yml` on every push to `main` that touches `docs/**`.

`README.md` is a stub: tagline, install command, and a link to the docs site. Do not grow it back into a long-form reference. Anything user-visible belongs on the docs site.

Any PR that changes user-visible CLI surface — subcommands, flags, JSON schema, file format, or config keys — must update the corresponding docs page in the same PR. Same-PR is the rule because it is the only thing keeping the docs site from drifting out of sync with the CLI; there is no auto-generated content and no link-checking in CI.

The vendored reference slash command at `docs/src/content/docs/examples/worktask-slash-command.md` is part of that contract. If a JSON schema or error-envelope change requires updating how an agent wraps the CLI, the slash-command file is updated in the same PR as the schema change.

## Agent skills

### Issue tracker

GitHub Issues at `jvcorredor/worktask`, accessed via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical roles using their default label strings (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `CONTEXT.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.
