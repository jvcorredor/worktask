package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jvcorredor/worktask/internal/config"
	"github.com/jvcorredor/worktask/internal/store"
)

var researchLogCmd = &cobra.Command{
	Use:   "research-log <fragment>",
	Short: "Print the latest research log path for a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fragment := args[0]
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		s := store.New(cfg.TasksDir)

		t, _, err := s.Get(fragment)
		if err != nil {
			return handleResolveError(cmd, s, fragment, err)
		}
		if t.LastResearchLog == "" {
			return fmt.Errorf("no research log recorded for %s", t.ID)
		}
		path := t.LastResearchLog
		if !filepath.IsAbs(path) {
			path = filepath.Join(cfg.TasksDir, path)
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), path)
		return err
	},
}

func init() {
	rootCmd.AddCommand(researchLogCmd)
}
