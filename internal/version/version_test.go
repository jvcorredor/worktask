package version

import (
	"runtime/debug"
	"testing"
)

// resetForTest restores package state to what it would be at process start
// for a binary built without ldflags and outside `go install`. Tests should
// defer it via t.Cleanup so they do not leak state into siblings.
func resetForTest(t *testing.T) {
	t.Helper()
	prevV, prevC, prevD := Version, Commit, Date
	prevBI := readBuildInfo
	t.Cleanup(func() {
		Version, Commit, Date = prevV, prevC, prevD
		readBuildInfo = prevBI
	})
	Version = ""
	Commit = ""
	Date = ""
	readBuildInfo = func() (*debug.BuildInfo, bool) { return nil, false }
}

func TestGet_baselineReturnsSentinels(t *testing.T) {
	resetForTest(t)

	got := Get()

	if got.Version != "dev" {
		t.Errorf("Version = %q; want %q", got.Version, "dev")
	}
	if got.Commit != "unknown" {
		t.Errorf("Commit = %q; want %q", got.Commit, "unknown")
	}
	if got.Date != "unknown" {
		t.Errorf("Date = %q; want %q", got.Date, "unknown")
	}
	if got.Source != "default" {
		t.Errorf("Source = %q; want %q", got.Source, "default")
	}
}

func TestGet_buildInfoFallback(t *testing.T) {
	resetForTest(t)
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "v0.4.2"},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "cafef00dcafef00dcafef00dcafef00dcafef00d"},
				{Key: "vcs.time", Value: "2026-04-30T08:15:00Z"},
			},
		}, true
	}

	got := Get()

	if got.Version != "v0.4.2" {
		t.Errorf("Version = %q; want %q", got.Version, "v0.4.2")
	}
	if got.Commit != "cafef00dcafef00dcafef00dcafef00dcafef00d" {
		t.Errorf("Commit = %q; want vcs.revision value", got.Commit)
	}
	if got.Date != "2026-04-30T08:15:00Z" {
		t.Errorf("Date = %q; want vcs.time value", got.Date)
	}
	if got.Source != "buildinfo" {
		t.Errorf("Source = %q; want %q", got.Source, "buildinfo")
	}
}

func TestGet_buildInfoWithoutVCSSettings(t *testing.T) {
	resetForTest(t)
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "v0.4.2"}}, true
	}

	got := Get()

	if got.Version != "v0.4.2" {
		t.Errorf("Version = %q; want %q", got.Version, "v0.4.2")
	}
	if got.Commit != "unknown" {
		t.Errorf("Commit = %q; want %q when vcs settings are absent", got.Commit, "unknown")
	}
	if got.Date != "unknown" {
		t.Errorf("Date = %q; want %q when vcs settings are absent", got.Date, "unknown")
	}
	if got.Source != "buildinfo" {
		t.Errorf("Source = %q; want %q", got.Source, "buildinfo")
	}
}

func TestGet_buildInfoMainVersionDevlIsTreatedAsDefault(t *testing.T) {
	// `go run` and bare `go build` produce BuildInfo whose Main.Version is
	// "(devel)". That is not a useful version string — the package treats
	// it as if no buildinfo were available so the "dev" sentinel takes
	// over instead of being surfaced to the user.
	resetForTest(t)
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true
	}

	got := Get()

	if got.Version != "dev" || got.Source != "default" {
		t.Errorf("Get() with Main.Version=(devel) = %+v; want Version=dev Source=default", got)
	}
}

func TestGet_ldflagsValuesWin(t *testing.T) {
	resetForTest(t)
	Version = "v1.2.3"
	Commit = "deadbeef"
	Date = "2026-05-03T10:00:00Z"

	// Even if ReadBuildInfo would return something, ldflags must win.
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "v0.0.1"}}, true
	}

	got := Get()

	if got.Version != "v1.2.3" {
		t.Errorf("Version = %q; want %q", got.Version, "v1.2.3")
	}
	if got.Commit != "deadbeef" {
		t.Errorf("Commit = %q; want %q", got.Commit, "deadbeef")
	}
	if got.Date != "2026-05-03T10:00:00Z" {
		t.Errorf("Date = %q; want %q", got.Date, "2026-05-03T10:00:00Z")
	}
	if got.Source != "ldflags" {
		t.Errorf("Source = %q; want %q", got.Source, "ldflags")
	}
}
