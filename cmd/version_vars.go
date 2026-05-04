package cmd

import (
	intversion "github.com/jvcorredor/bytheway/internal/version"
)

// Version, Commit, and Date are the cobra-side ldflags landing zone.
// Release builds inject them via
//
//	go build -ldflags="-X github.com/jvcorredor/bytheway/cmd.Version=v1.2.3 \
//	                   -X github.com/jvcorredor/bytheway/cmd.Commit=$SHA \
//	                   -X github.com/jvcorredor/bytheway/cmd.Date=$ISO8601"
//
// They mirror the convention of putting build identity at the binary's
// command package; syncVersion forwards them into internal/version where
// the resolution rules live.
var (
	Version string
	Commit  string
	Date    string
)

func init() {
	syncVersion()
}

// syncVersion copies the cmd-side ldflags landing zone into the
// internal/version package and refreshes rootCmd.Version so cobra's
// built-in --version / -v handling reports the resolved value. It is
// idempotent and safe to call multiple times; tests use it to apply
// new fixture values after rewriting the version vars.
func syncVersion() {
	if Version != "" {
		intversion.Version = Version
	}
	if Commit != "" {
		intversion.Commit = Commit
	}
	if Date != "" {
		intversion.Date = Date
	}
	rootCmd.Version = intversion.Get().Version
}
