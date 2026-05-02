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
