---
title: Releases
description: How worktask versions and ships releases — Conventional Commits, pre-1.0 policy, and where to find the artifacts.
---

Release notes, source archives, and platform-specific tarballs for every version live on the [GitHub Releases page](https://github.com/jvcorredor/worktask/releases). The latest release is also what `curl ... | sh` installs by default; see [Install](./install.md).

## Conventional Commits

Every commit on `main` follows [Conventional Commits](https://www.conventionalcommits.org/). The release pipeline runs on every push to `main` and decides whether to cut a tag based on the commit type:

| Commit type                                         | Effect on version          |
|-----------------------------------------------------|----------------------------|
| `feat:`                                             | bumps minor                |
| `fix:`                                              | bumps patch                |
| `feat!:` / `fix!:` / footer `BREAKING CHANGE:`      | breaking change (see below)|
| `chore:`, `docs:`, `refactor:`, `test:`, `ci:`, `build:` | no release                 |

Release notes are generated from the commit log and published only to the GitHub Releases page; no `CHANGELOG.md` is committed back to the repo.

## Pre-1.0 versioning policy

While the major version is `0`, breaking changes bump the **minor** version rather than jumping to `1.0.0`. So `feat!:` on `0.2.0` cuts `0.3.0`, not `1.0.0`. Once `1.0.0` is cut, breaking changes resume bumping major as usual.

This keeps the pre-1.0 surface free to evolve without prematurely committing to a stable contract, while still signalling breakage clearly in release notes and version diffs.

## Pinning a version

`curl ... | sh` installs the latest release. To pin:

```
VERSION=0.2.0 curl -fsSL https://jvcorredor.github.io/worktask/install.sh | sh
```

Or with `go install`:

```
go install github.com/jvcorredor/worktask@v0.2.0
```
