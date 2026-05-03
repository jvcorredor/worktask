#!/bin/sh
# Asserts that the `worktask` binary on PATH reports the version named
# in EXPECTED_VERSION. Used by the post-release smoke-test job in
# .github/workflows/release.yml: a mismatch fails the workflow loudly so
# the maintainer is alerted within minutes of a broken release.

set -eu

die() {
    printf '::error::%s\n' "$*" >&2
    exit 1
}

[ -n "${EXPECTED_VERSION:-}" ] || die 'EXPECTED_VERSION is unset'

command -v worktask >/dev/null 2>&1 \
    || die 'worktask not found on PATH'

actual=$(worktask version --format=json | jq -r '.version') \
    || die 'failed to read version from worktask version --format=json'

[ "$actual" = "$EXPECTED_VERSION" ] \
    || die "worktask version mismatch: expected $EXPECTED_VERSION, got $actual"

printf 'smoke test passed: worktask version reports %s\n' "$actual"
