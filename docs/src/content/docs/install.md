---
title: Install
description: Install worktask from a pre-built binary or from source, with environment-variable overrides, $PATH setup, and checksum verification.
---

worktask supports two install methods: a pre-built binary served from this docs site (no Go toolchain required), and `go install` from source.

## Pre-built binary

```
curl -fsSL https://jvcorredor.github.io/worktask/install.sh | sh
```

The script downloads the platform-appropriate tarball and `checksums.txt` from the matching [GitHub Release](https://github.com/jvcorredor/worktask/releases), verifies the SHA256 of the tarball against the manifest, and extracts the `worktask` binary into the install directory. It does not edit any shell rc files.

### Environment-variable overrides

| Variable      | Default                  | Notes                                                                        |
|---------------|--------------------------|------------------------------------------------------------------------------|
| `INSTALL_DIR` | `$HOME/.local/bin`       | Destination directory. Use `/usr/local/bin` for a system-wide install.       |
| `VERSION`     | latest GitHub release    | Pin to a specific release tag, e.g. `VERSION=0.2.0` (leading `v` optional).  |

```
INSTALL_DIR=/usr/local/bin curl -fsSL https://jvcorredor.github.io/worktask/install.sh | sudo sh
VERSION=0.2.0 curl -fsSL https://jvcorredor.github.io/worktask/install.sh | sh
```

### Supported platforms

`linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`. Other platforms abort with a clear error; build from source instead.

### `$PATH` setup

If the install directory is not on your `$PATH`, the script prints a one-line snippet you can paste into your shell config — `export PATH="$HOME/.local/bin:$PATH"` for bash/zsh/sh, `fish_add_path $HOME/.local/bin` for fish. The script never edits rc files itself.

### Checksum verification

The script downloads `checksums.txt` from the same release and compares the SHA256 of the tarball against the entry for the platform-specific filename. A mismatch aborts with a non-zero exit and a clear error; nothing is installed in that case.

### Re-running the script

If a `worktask` binary already lives at the destination, the script reads its version with `worktask version` and prints `Upgrading worktask from X to Y` when the resolved version differs.

### macOS quarantine note

Tarballs downloaded through a browser get the `com.apple.quarantine` extended attribute, which makes Gatekeeper block the binary on first run. The install script downloads via `curl` and is unaffected. If you fetched the tarball manually instead, clear the attribute once before running:

```
xattr -dr com.apple.quarantine ./worktask
```

## From source (`go install`)

```
go install github.com/jvcorredor/worktask@latest
```

Requires a Go toolchain. The binary lands in `$GOBIN` (or `$GOPATH/bin`) as `worktask`. Pin a specific version with `@v0.2.0`. Module-mode `go install` does the same SHA verification as any other Go module download.
