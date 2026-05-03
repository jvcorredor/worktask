// Package cmd assembles the worktask cobra command tree.
//
// The root command lives in this file as rootCmd; each subcommand —
// add, list, show, close, reopen, edit, update, append, research —
// is defined in its own file in this package and attached to rootCmd
// from that file's init(). Per-command flags,
// help strings (Cobra's Short and Long), and Run functions are kept
// next to the command they configure rather than centralised here, so
// the command tree is built up by package initialisation rather than by
// an explicit registry.
//
// The package's only public entry point is [Execute], which main.go
// calls. Subcommand var blocks and init functions are intentionally
// undocumented at the symbol level: per-var doc comments on cobra
// command values are not idiomatic, and the user-facing help text
// already lives in each command's Short and Long fields.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jvcorredor/worktask/internal/config"
	"github.com/jvcorredor/worktask/internal/store"
)

const (
	formatHuman = "human"
	formatJSON  = "json"
)

var format string

var rootCmd = &cobra.Command{
	Use:           "worktask",
	Short:         "Markdown-backed task CLI",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&format, "format", formatHuman, "output format: human|json")
}

var errExit = errors.New("exit")

// Execute runs the worktask CLI: it dispatches the root cobra command,
// prints any error to stderr (suppressing the sentinel used to short-
// circuit a subcommand without an extra error line), and exits the
// process with status 1 on failure. It is the only entry point main.go
// calls and does not return on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if !errors.Is(err, errExit) {
			fmt.Fprintln(os.Stderr, "error:", err)
		}
		os.Exit(1)
	}
}

func newStore() (*store.Store, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return store.New(cfg.TasksDir), nil
}
