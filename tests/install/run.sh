#!/bin/sh
# POSIX test harness for docs/public/install.sh.
#
# Each case runs in an isolated sandbox dir. The sandbox provides:
#   - $home_dir            a fresh $HOME (so ~/.local/bin is empty per case)
#   - $bin_shim            a directory prepended to $PATH containing curl/uname shims
#   - $fixtures            a directory mirroring the GitHub release URL space
#                          (the curl shim resolves URLs to files under here)
#
# The shim maps:
#   https://api.github.com/repos/jvcorredor/worktask/releases/latest
#       -> $fixtures/latest.json
#   https://github.com/jvcorredor/worktask/releases/download/<rest>
#       -> $fixtures/<rest>
#
# Cases assemble the fixture tree they need, invoke install.sh under HOME=$home_dir
# and the shimmed PATH, and assert on filesystem state / output / exit code.

set -eu

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
INSTALL_SH="$REPO_ROOT/docs/public/install.sh"

failures=0
total=0

sha256_of() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    else
        shasum -a 256 "$1" | awk '{print $1}'
    fi
}

make_sandbox() {
    sandbox="$(mktemp -d "${TMPDIR:-/tmp}/wt-install-test.XXXXXX")"
    fixtures="$sandbox/fixtures"
    home_dir="$sandbox/home"
    bin_shim="$sandbox/bin"
    mkdir -p "$fixtures" "$home_dir" "$bin_shim"
}

cleanup_sandbox() {
    [ -n "${sandbox:-}" ] && rm -rf "$sandbox"
}

# Build a fake release tarball + checksums.txt under $fixtures/<tag>/.
# Args: tag (e.g. v0.2.0), os (linux|darwin), arch (amd64|arm64), [version_for_binary]
# The fake binary is a shell script that prints "worktask <version>" on `version`.
build_fixture_release() {
    tag="$1"; os="$2"; arch="$3"
    version="${4:-${tag#v}}"
    rel_dir="$fixtures/$tag"
    mkdir -p "$rel_dir"
    build_dir="$sandbox/build.$$.$tag.$os.$arch"
    mkdir -p "$build_dir"
    cat >"$build_dir/worktask" <<EOF
#!/bin/sh
case "\$1" in
    version|--version|-v) echo "worktask $version" ;;
    *) echo "worktask $version" ;;
esac
EOF
    chmod +x "$build_dir/worktask"
    tarball="worktask_${version}_${os}_${arch}.tar.gz"
    tar -czf "$rel_dir/$tarball" -C "$build_dir" worktask
    # Append to checksums.txt (one release may have multiple platforms).
    (
        cd "$rel_dir"
        printf '%s  %s\n' "$(sha256_of "$tarball")" "$tarball" >> checksums.txt
    )
    rm -rf "$build_dir"
}

# Write a /latest API fixture pinning to a given tag.
fixture_latest_tag() {
    tag="$1"
    cat >"$fixtures/latest.json" <<EOF
{"tag_name":"$tag","name":"$tag"}
EOF
}

install_curl_shim() {
    cat >"$bin_shim/curl" <<'SHIM'
#!/bin/sh
set -eu
out=
url=
while [ $# -gt 0 ]; do
    case "$1" in
        -o|--output) out="$2"; shift 2 ;;
        --) shift; break ;;
        http*://*) url="$1"; shift ;;
        -*) shift ;;
        *) shift ;;
    esac
done
[ -n "$url" ] || { echo "curl-shim: no URL among args" >&2; exit 22; }
[ -n "${WT_CURL_LOG:-}" ] && printf '%s\n' "$url" >>"$WT_CURL_LOG"
src=
case "$url" in
    https://api.github.com/repos/jvcorredor/worktask/releases/latest)
        src="$WT_FIXTURES/latest.json" ;;
    https://github.com/jvcorredor/worktask/releases/download/*)
        rest="${url#https://github.com/jvcorredor/worktask/releases/download/}"
        src="$WT_FIXTURES/$rest" ;;
    *)
        echo "curl-shim: unhandled url $url" >&2; exit 22 ;;
esac
[ -f "$src" ] || { echo "curl-shim: missing fixture $src (url=$url)" >&2; exit 22; }
if [ -n "$out" ]; then
    cp "$src" "$out"
else
    cat "$src"
fi
SHIM
    chmod +x "$bin_shim/curl"
}

# Force platform reported by the script.  Args: kernel (Linux/Darwin), machine (x86_64/arm64/aarch64).
install_uname_shim() {
    cat >"$bin_shim/uname" <<SHIM
#!/bin/sh
kernel='$1'
machine='$2'
case "\$1" in
    -s) printf '%s\n' "\$kernel" ;;
    -m) printf '%s\n' "\$machine" ;;
    *) /usr/bin/uname "\$@" ;;
esac
SHIM
    chmod +x "$bin_shim/uname"
}

# Run install.sh in the sandbox. Extra args become a `env` prefix (KEY=VALUE ...).
run_install() {
    env -i \
        HOME="$home_dir" \
        PATH="$bin_shim:/usr/bin:/bin" \
        WT_FIXTURES="$fixtures" \
        WT_CURL_LOG="$sandbox/curl.log" \
        "$@" \
        sh "$INSTALL_SH"
}

# Run install.sh expecting failure; prints stdout/stderr to current FDs.
run_install_expect_fail() {
    if run_install "$@"; then
        echo "expected install.sh to fail but it succeeded" >&2
        return 1
    fi
}

# === Cases ===

case_tracer_bullet_default_install() {
    install_curl_shim
    install_uname_shim Linux x86_64
    build_fixture_release v0.2.0 linux amd64
    fixture_latest_tag v0.2.0
    run_install
    [ -x "$home_dir/.local/bin/worktask" ] || { echo "binary not installed"; return 1; }
    out="$("$home_dir/.local/bin/worktask" version)"
    [ "$out" = "worktask 0.2.0" ] || { echo "version mismatch: $out"; return 1; }
}

case_install_dir_override() {
    install_curl_shim
    install_uname_shim Linux x86_64
    build_fixture_release v0.2.0 linux amd64
    fixture_latest_tag v0.2.0
    custom="$sandbox/custom/bin"
    run_install INSTALL_DIR="$custom"
    [ -x "$custom/worktask" ] || { echo "binary not in custom dir"; return 1; }
    [ ! -e "$home_dir/.local/bin/worktask" ] || { echo "binary leaked into default dir"; return 1; }
}

case_version_pin_skips_latest_lookup() {
    install_curl_shim
    install_uname_shim Linux x86_64
    build_fixture_release v0.1.0 linux amd64
    # Intentionally no latest.json — VERSION must skip the /latest call.
    run_install VERSION=0.1.0
    out="$("$home_dir/.local/bin/worktask" version)"
    [ "$out" = "worktask 0.1.0" ] || { echo "wrong version installed: $out"; return 1; }
}

case_version_pin_accepts_v_prefix() {
    install_curl_shim
    install_uname_shim Linux x86_64
    build_fixture_release v0.1.0 linux amd64
    run_install VERSION=v0.1.0
    out="$("$home_dir/.local/bin/worktask" version)"
    [ "$out" = "worktask 0.1.0" ] || { echo "wrong version installed: $out"; return 1; }
}

case_default_version_calls_latest_api() {
    install_curl_shim
    install_uname_shim Linux x86_64
    build_fixture_release v0.2.0 linux amd64
    fixture_latest_tag v0.2.0
    run_install
    grep -qx 'https://api.github.com/repos/jvcorredor/worktask/releases/latest' \
        "$sandbox/curl.log" \
        || { echo "expected /releases/latest in curl log:"; cat "$sandbox/curl.log"; return 1; }
}

case_version_pin_does_not_call_latest_api() {
    install_curl_shim
    install_uname_shim Linux x86_64
    build_fixture_release v0.1.0 linux amd64
    run_install VERSION=0.1.0
    if grep -qx 'https://api.github.com/repos/jvcorredor/worktask/releases/latest' \
        "$sandbox/curl.log"; then
        echo "VERSION pin should not call /releases/latest, but it did:"
        cat "$sandbox/curl.log"
        return 1
    fi
}

case_checksum_mismatch_aborts() {
    install_curl_shim
    install_uname_shim Linux x86_64
    build_fixture_release v0.2.0 linux amd64
    fixture_latest_tag v0.2.0
    # Corrupt the tarball after checksums.txt was written, so SHA won't match.
    printf 'corrupted' >>"$fixtures/v0.2.0/worktask_0.2.0_linux_amd64.tar.gz"
    if run_install; then
        echo "install.sh should have failed on checksum mismatch"
        return 1
    fi
    [ ! -e "$home_dir/.local/bin/worktask" ] \
        || { echo "binary was installed despite checksum mismatch"; return 1; }
}

case_upgrade_message_when_binary_present() {
    install_curl_shim
    install_uname_shim Linux x86_64
    build_fixture_release v0.1.0 linux amd64
    build_fixture_release v0.2.0 linux amd64
    fixture_latest_tag v0.2.0

    run_install VERSION=0.1.0

    log="$sandbox/upgrade.log"
    run_install >"$log" 2>&1
    grep -E -q 'pgrading.*0\.1\.0.*0\.2\.0' "$log" \
        || { echo "missing upgrade message; install output was:"; cat "$log"; return 1; }
}

case_no_upgrade_message_on_first_install() {
    install_curl_shim
    install_uname_shim Linux x86_64
    build_fixture_release v0.2.0 linux amd64
    fixture_latest_tag v0.2.0

    log="$sandbox/first.log"
    run_install >"$log" 2>&1
    if grep -E -i -q 'upgrad' "$log"; then
        echo "first install should not say 'upgrading'; output was:"
        cat "$log"
        return 1
    fi
}

case_path_snippet_when_install_dir_off_path() {
    install_curl_shim
    install_uname_shim Linux x86_64
    build_fixture_release v0.2.0 linux amd64
    fixture_latest_tag v0.2.0

    log="$sandbox/path.log"
    # Default INSTALL_DIR ($HOME/.local/bin) is not on the sandboxed PATH.
    run_install SHELL=/bin/bash >"$log" 2>&1
    grep -q "$home_dir/.local/bin" "$log" \
        || { echo "PATH snippet did not mention install dir; output:"; cat "$log"; return 1; }
    grep -q 'export PATH' "$log" \
        || { echo "expected bash 'export PATH' snippet; output:"; cat "$log"; return 1; }
}

case_no_path_snippet_when_install_dir_on_path() {
    install_curl_shim
    install_uname_shim Linux x86_64
    build_fixture_release v0.2.0 linux amd64
    fixture_latest_tag v0.2.0
    custom="$sandbox/onpath/bin"
    mkdir -p "$custom"

    log="$sandbox/path.log"
    # Run install with $custom prepended to PATH so it is already discoverable.
    env -i \
        HOME="$home_dir" \
        PATH="$custom:$bin_shim:/usr/bin:/bin" \
        WT_FIXTURES="$fixtures" \
        WT_CURL_LOG="$sandbox/curl.log" \
        SHELL=/bin/bash \
        INSTALL_DIR="$custom" \
        sh "$INSTALL_SH" >"$log" 2>&1
    if grep -q 'export PATH' "$log" || grep -q 'fish_add_path' "$log"; then
        echo "should not print PATH snippet when install dir is already on PATH; output:"
        cat "$log"
        return 1
    fi
}

case_path_snippet_fish_shell() {
    install_curl_shim
    install_uname_shim Linux x86_64
    build_fixture_release v0.2.0 linux amd64
    fixture_latest_tag v0.2.0

    log="$sandbox/fish.log"
    run_install SHELL=/usr/local/bin/fish >"$log" 2>&1
    grep -q 'fish_add_path' "$log" \
        || { echo "expected fish_add_path snippet; output:"; cat "$log"; return 1; }
    if grep -q 'export PATH' "$log"; then
        echo "fish shell should not print bash-style export; output:"
        cat "$log"
        return 1
    fi
}

case_platform_linux_arm64() {
    install_curl_shim
    install_uname_shim Linux aarch64
    build_fixture_release v0.2.0 linux arm64
    fixture_latest_tag v0.2.0
    run_install
    [ -x "$home_dir/.local/bin/worktask" ] || { echo "binary not installed"; return 1; }
}

case_platform_darwin_amd64() {
    install_curl_shim
    install_uname_shim Darwin x86_64
    build_fixture_release v0.2.0 darwin amd64
    fixture_latest_tag v0.2.0
    run_install
    [ -x "$home_dir/.local/bin/worktask" ] || { echo "binary not installed"; return 1; }
}

case_platform_darwin_arm64() {
    install_curl_shim
    install_uname_shim Darwin arm64
    build_fixture_release v0.2.0 darwin arm64
    fixture_latest_tag v0.2.0
    run_install
    [ -x "$home_dir/.local/bin/worktask" ] || { echo "binary not installed"; return 1; }
}

case_platform_unsupported_os_aborts() {
    install_curl_shim
    install_uname_shim FreeBSD x86_64
    fixture_latest_tag v0.2.0
    log="$sandbox/err.log"
    if run_install >"$log" 2>&1; then
        echo "expected failure on unsupported OS; output:"
        cat "$log"
        return 1
    fi
    grep -E -i -q 'unsupported|FreeBSD' "$log" \
        || { echo "error message did not mention unsupported OS:"; cat "$log"; return 1; }
}

case_platform_unsupported_arch_aborts() {
    install_curl_shim
    install_uname_shim Linux ppc64le
    fixture_latest_tag v0.2.0
    log="$sandbox/err.log"
    if run_install >"$log" 2>&1; then
        echo "expected failure on unsupported arch; output:"
        cat "$log"
        return 1
    fi
    grep -E -i -q 'unsupported|ppc64' "$log" \
        || { echo "error message did not mention unsupported arch:"; cat "$log"; return 1; }
}

case_posix_parse_sh() {
    sh -n "$INSTALL_SH" || { echo "sh -n parse failed"; return 1; }
}

case_posix_parse_dash() {
    if ! command -v dash >/dev/null 2>&1; then
        echo "skipping: dash not installed"
        return 0
    fi
    dash -n "$INSTALL_SH" || { echo "dash -n parse failed"; return 1; }
}

case_install_does_not_edit_rc_files() {
    install_curl_shim
    install_uname_shim Linux x86_64
    build_fixture_release v0.2.0 linux amd64
    fixture_latest_tag v0.2.0
    # Pre-create rc files so we can assert untouched.
    : >"$home_dir/.bashrc"
    : >"$home_dir/.zshrc"
    : >"$home_dir/.profile"
    run_install SHELL=/bin/bash >/dev/null 2>&1
    [ ! -s "$home_dir/.bashrc" ] || { echo ".bashrc was modified"; return 1; }
    [ ! -s "$home_dir/.zshrc" ] || { echo ".zshrc was modified"; return 1; }
    [ ! -s "$home_dir/.profile" ] || { echo ".profile was modified"; return 1; }
}

# === Driver ===

run_case() {
    name="$1"
    total=$((total + 1))
    make_sandbox
    out_file="$sandbox/out"
    err_file="$sandbox/err"
    rc=0
    ( "case_$name" ) >"$out_file" 2>"$err_file" || rc=$?
    if [ "$rc" -eq 0 ]; then
        printf 'ok  %s\n' "$name"
    else
        failures=$((failures + 1))
        printf 'FAIL %s (rc=%d)\n' "$name" "$rc"
        echo '--- stdout ---'
        cat "$out_file" || true
        echo '--- stderr ---'
        cat "$err_file" || true
        echo '--------------'
    fi
    cleanup_sandbox
}

CASES='
posix_parse_sh
posix_parse_dash
tracer_bullet_default_install
install_dir_override
version_pin_skips_latest_lookup
version_pin_accepts_v_prefix
default_version_calls_latest_api
version_pin_does_not_call_latest_api
checksum_mismatch_aborts
upgrade_message_when_binary_present
no_upgrade_message_on_first_install
path_snippet_when_install_dir_off_path
no_path_snippet_when_install_dir_on_path
path_snippet_fish_shell
platform_linux_arm64
platform_darwin_amd64
platform_darwin_arm64
platform_unsupported_os_aborts
platform_unsupported_arch_aborts
install_does_not_edit_rc_files
'

for name in $CASES; do
    run_case "$name"
done

if [ "$failures" -ne 0 ]; then
    printf '\n%d/%d cases failed\n' "$failures" "$total" >&2
    exit 1
fi
printf '\nall %d cases passed\n' "$total"
