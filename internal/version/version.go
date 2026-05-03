// Package version reports the worktask binary's build identity.
//
// Resolution order, hidden behind [Get]:
//
//  1. ldflags-injected values written into [Version], [Commit], [Date]
//     by `go build -ldflags="-X ..."`.
//  2. module-proxy metadata returned by [runtime/debug.ReadBuildInfo]
//     for binaries produced by `go install module@version`.
//  3. sentinel defaults (`dev`, `unknown`) for unannotated developer
//     builds.
//
// The [Info.Source] field records which branch produced the result so
// callers and tests can distinguish them.
package version

import "runtime/debug"

// Version, Commit, and Date are the ldflags landing zone inside this
// package. They are intentionally exported so a release build can wire
// them via `-ldflags="-X github.com/jvcorredor/worktask/internal/version.Version=v1.2.3 ..."`.
// Unset, [Get] falls back to module-proxy metadata or sentinel defaults.
var (
	Version string
	Commit  string
	Date    string
)

// readBuildInfo is a package-level indirection over
// [runtime/debug.ReadBuildInfo] so tests can swap in fixtures without
// rebuilding the test binary with ldflags.
var readBuildInfo = debug.ReadBuildInfo

// Info is the resolved version identity for the running binary.
//
// Source records which resolution branch produced the values:
// "ldflags", "buildinfo", or "default". Callers should treat the field
// as part of the contract rather than a debugging aid; the JSON
// rendering of `worktask version --format=json` exposes it.
type Info struct {
	Version string
	Commit  string
	Date    string
	Source  string
}

// Get resolves the binary's version identity using the order documented
// on the package. It never returns an error: an unannotated dev build
// produces ("dev", "unknown", "unknown", "default") rather than failing.
func Get() Info {
	if Version != "" {
		return Info{
			Version: Version,
			Commit:  orUnknown(Commit),
			Date:    orUnknown(Date),
			Source:  "ldflags",
		}
	}
	if info, ok := readBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		commit, date := vcsFromBuildInfo(info)
		return Info{
			Version: info.Main.Version,
			Commit:  commit,
			Date:    date,
			Source:  "buildinfo",
		}
	}
	return Info{
		Version: "dev",
		Commit:  "unknown",
		Date:    "unknown",
		Source:  "default",
	}
}

func vcsFromBuildInfo(info *debug.BuildInfo) (commit, date string) {
	commit, date = "unknown", "unknown"
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if s.Value != "" {
				commit = s.Value
			}
		case "vcs.time":
			if s.Value != "" {
				date = s.Value
			}
		}
	}
	return commit, date
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}
