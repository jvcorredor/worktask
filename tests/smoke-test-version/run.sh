#!/bin/sh
# Black-box test for smoke-test-version.sh. Runs the script with a fake
# `worktask` binary on PATH and verifies exit codes and stderr for each
# behaviour the smoke-test job depends on.

set -eu

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
script="$REPO_ROOT/.github/scripts/smoke-test-version.sh"

fail=0
case_name=""

start_case() {
    case_name="$1"
}

assert() {
    if [ "$1" = "$2" ]; then
        printf '  ok  %s\n' "$case_name"
    else
        printf '  FAIL %s: expected %s, got %s\n' "$case_name" "$2" "$1" >&2
        fail=1
    fi
}

with_fake_worktask() {
    reported_version="$1"
    workdir=$(mktemp -d "${TMPDIR:-/tmp}/smoke-test.XXXXXX")
    cat >"$workdir/worktask" <<EOF
#!/bin/sh
cat <<JSON
{"version":"$reported_version","commit":"abc","date":"2026-05-03","source":"ldflags"}
JSON
EOF
    chmod +x "$workdir/worktask"
    printf '%s\n' "$workdir"
}

# Case 1: matching version exits 0.
start_case 'matching version exits 0'
shim=$(with_fake_worktask 0.4.0)
PATH="$shim:$PATH" EXPECTED_VERSION=0.4.0 sh "$script" >/dev/null 2>&1
assert "$?" 0
rm -rf "$shim"

# Case 2: mismatched version exits non-zero with both versions in stderr.
start_case 'mismatched version exits non-zero'
shim=$(with_fake_worktask 0.3.9)
err=$(PATH="$shim:$PATH" EXPECTED_VERSION=0.4.0 sh "$script" 2>&1 1>/dev/null) && rc=0 || rc=$?
assert "$rc" 1
case "$err" in
    *0.4.0*0.3.9*|*0.3.9*0.4.0*)
        printf '  ok  mismatch stderr names both versions\n' ;;
    *)
        printf '  FAIL mismatch stderr did not name both versions: %s\n' "$err" >&2
        fail=1 ;;
esac
rm -rf "$shim"

# Case 3: missing EXPECTED_VERSION env exits non-zero.
start_case 'missing EXPECTED_VERSION exits non-zero'
shim=$(with_fake_worktask 0.4.0)
PATH="$shim:$PATH" sh "$script" >/dev/null 2>&1 && rc=0 || rc=$?
[ "$rc" -ne 0 ] && printf '  ok  %s\n' "$case_name" || { printf '  FAIL %s: exited 0\n' "$case_name" >&2; fail=1; }
rm -rf "$shim"

# Case 4: worktask not on PATH exits non-zero.
start_case 'missing worktask binary exits non-zero'
empty=$(mktemp -d "${TMPDIR:-/tmp}/empty.XXXXXX")
PATH="$empty" EXPECTED_VERSION=0.4.0 sh "$script" >/dev/null 2>&1 && rc=0 || rc=$?
[ "$rc" -ne 0 ] && printf '  ok  %s\n' "$case_name" || { printf '  FAIL %s: exited 0\n' "$case_name" >&2; fail=1; }
rm -rf "$empty"

exit "$fail"
