#!/bin/sh
# btw installer — POSIX, no bashisms.
#
# Usage:  curl -fsSL https://jvcorredor.github.io/btw/install.sh | sh
#
# Env vars:
#   INSTALL_DIR   destination directory (default: $HOME/.local/bin)
#   VERSION       release tag without leading "v" (default: latest)

set -eu

REPO='jvcorredor/bytheway'
RELEASES_BASE="https://github.com/${REPO}/releases/download"
LATEST_API="https://api.github.com/repos/${REPO}/releases/latest"

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

die() {
    printf 'install.sh: %s\n' "$*" >&2
    exit 1
}

resolve_version() {
    if [ -n "${VERSION:-}" ]; then
        printf '%s\n' "${VERSION#v}"
        return
    fi
    tag=$(curl -fsSL "$LATEST_API" \
        | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
        | head -n1)
    [ -n "$tag" ] || die "could not resolve latest release tag from $LATEST_API"
    printf '%s\n' "${tag#v}"
}

detect_platform() {
    kernel=$(uname -s)
    machine=$(uname -m)
    case "$kernel" in
        Linux) os=linux ;;
        Darwin) os=darwin ;;
        *) die "unsupported OS: $kernel" ;;
    esac
    case "$machine" in
        x86_64|amd64) arch=amd64 ;;
        arm64|aarch64) arch=arm64 ;;
        *) die "unsupported architecture: $machine" ;;
    esac
    printf '%s_%s\n' "$os" "$arch"
}

sha256_hash() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    else
        shasum -a 256 "$1" | awk '{print $1}'
    fi
}

installed_version() {
    bin="$1"
    [ -x "$bin" ] || return 0
    "$bin" version 2>/dev/null | awk 'NR==1 {print $2}' | sed 's/^v//'
}

print_path_hint_if_needed() {
    case ":$PATH:" in
        *":$INSTALL_DIR:"*) return 0 ;;
    esac
    case "$(basename "${SHELL:-/bin/sh}")" in
        fish)
            printf '%s is not on $PATH. Add it with:\n    fish_add_path %s\n' \
                "$INSTALL_DIR" "$INSTALL_DIR" ;;
        *)
            printf '%s is not on $PATH. Add it with:\n    export PATH="%s:$PATH"\n' \
                "$INSTALL_DIR" "$INSTALL_DIR" ;;
    esac
}

main() {
    version=$(resolve_version)
    platform=$(detect_platform)
    tag="v${version}"
    tarball="bytheway_${version}_${platform}.tar.gz"
    tarball_url="${RELEASES_BASE}/${tag}/${tarball}"
    checksums_url="${RELEASES_BASE}/${tag}/checksums.txt"

    install_path="$INSTALL_DIR/btw"
    existing=$(installed_version "$install_path")

    workdir=$(mktemp -d "${TMPDIR:-/tmp}/btw-install.XXXXXX")
    trap 'rm -rf "$workdir"' EXIT

    printf 'Downloading %s\n' "$tarball"
    curl -fsSL "$tarball_url" -o "$workdir/$tarball" \
        || die "failed to download $tarball_url"
    curl -fsSL "$checksums_url" -o "$workdir/checksums.txt" \
        || die "failed to download $checksums_url"

    expected=$(awk -v f="$tarball" '$2 == f { print $1; exit }' "$workdir/checksums.txt")
    [ -n "$expected" ] || die "no checksum for $tarball in checksums.txt"
    actual=$(sha256_hash "$workdir/$tarball")
    [ "$expected" = "$actual" ] || die "checksum mismatch for $tarball: expected $expected, got $actual"

    tar -xzf "$workdir/$tarball" -C "$workdir" btw \
        || die "failed to extract btw from $tarball"

    mkdir -p "$INSTALL_DIR"
    if [ -n "$existing" ] && [ "$existing" != "$version" ]; then
        printf 'Upgrading btw from %s to %s\n' "$existing" "$version"
    fi
    mv "$workdir/btw" "$install_path"
    chmod +x "$install_path"

    printf 'Installed btw %s to %s\n' "$version" "$install_path"

    print_path_hint_if_needed
}

main "$@"
