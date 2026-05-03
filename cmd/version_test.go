package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	intversion "github.com/jvcorredor/worktask/internal/version"
)

// setVersionForTest seeds the internal/version package's ldflags landing
// zone so version.Get() returns the chosen identity for the duration of
// the test. The ldflags branch wins over runtime/debug.ReadBuildInfo, so
// no buildinfo suppression is needed at this layer.
func setVersionForTest(t *testing.T, v, c, d string) {
	t.Helper()
	prevV, prevC, prevD := intversion.Version, intversion.Commit, intversion.Date
	t.Cleanup(func() {
		intversion.Version = prevV
		intversion.Commit = prevC
		intversion.Date = prevD
	})
	intversion.Version = v
	intversion.Commit = c
	intversion.Date = d
}

func runVersionCmd(t *testing.T, fmtFlag string) (stdout, stderr bytes.Buffer) {
	t.Helper()
	prevFormat := format
	format = fmtFlag
	t.Cleanup(func() { format = prevFormat })

	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"version"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute version: %v (stderr=%s)", err, stderr.String())
	}
	return stdout, stderr
}

// TestRootCmdVersionFlag is the cobra wiring test: after the cmd package
// init runs, rootCmd.Version must be populated from version.Get(), so
// `worktask --version` and `worktask -v` print a meaningful line via
// cobra's built-in version flag handling.
func TestRootCmdVersionFlag(t *testing.T) {
	setVersionForTest(t, "v9.9.9", "abc1234", "2026-05-03T12:00:00Z")
	syncVersion()
	t.Cleanup(syncVersion)

	if rootCmd.Version != "v9.9.9" {
		t.Errorf("rootCmd.Version = %q; want %q", rootCmd.Version, "v9.9.9")
	}

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"--version"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute --version: %v (stderr=%s)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "v9.9.9") {
		t.Errorf("--version output %q should contain v9.9.9", stdout.String())
	}
}

func TestVersionCmd(t *testing.T) {
	cases := []struct {
		name        string
		format      string
		injected    [3]string // version, commit, date
		assertHuman func(t *testing.T, out string)
		assertJSON  func(t *testing.T, out string)
	}{
		{
			name:     "human format prints a single line containing the version",
			format:   formatHuman,
			injected: [3]string{"v1.2.3", "deadbeef", "2026-05-03T10:00:00Z"},
			assertHuman: func(t *testing.T, out string) {
				if !strings.HasSuffix(out, "\n") {
					t.Errorf("human output should end in newline; got %q", out)
				}
				trimmed := strings.TrimRight(out, "\n")
				if strings.Contains(trimmed, "\n") {
					t.Errorf("human output should be a single line; got %q", out)
				}
				if !strings.Contains(trimmed, "v1.2.3") {
					t.Errorf("human output %q should contain version v1.2.3", trimmed)
				}
			},
		},
		{
			name:     "json format emits a well-formed object with documented keys",
			format:   formatJSON,
			injected: [3]string{"v1.2.3", "deadbeef", "2026-05-03T10:00:00Z"},
			assertJSON: func(t *testing.T, out string) {
				var got map[string]any
				if err := json.Unmarshal([]byte(out), &got); err != nil {
					t.Fatalf("json output is not well-formed: %v\n%s", err, out)
				}
				for _, key := range []string{"version", "commit", "date"} {
					if _, ok := got[key]; !ok {
						t.Errorf("json output missing key %q: %v", key, got)
					}
				}
				if got["version"] != "v1.2.3" {
					t.Errorf("json version = %v; want v1.2.3", got["version"])
				}
				if got["commit"] != "deadbeef" {
					t.Errorf("json commit = %v; want deadbeef", got["commit"])
				}
				if got["date"] != "2026-05-03T10:00:00Z" {
					t.Errorf("json date = %v; want 2026-05-03T10:00:00Z", got["date"])
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setVersionForTest(t, tc.injected[0], tc.injected[1], tc.injected[2])

			stdout, _ := runVersionCmd(t, tc.format)

			if tc.assertHuman != nil {
				tc.assertHuman(t, stdout.String())
			}
			if tc.assertJSON != nil {
				tc.assertJSON(t, stdout.String())
			}
		})
	}
}
