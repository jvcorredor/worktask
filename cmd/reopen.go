package cmd

import (
	"github.com/spf13/cobra"

	"github.com/jvcorredor/bytheway/internal/store"
)

var reopenCmd = &cobra.Command{
	Use:               "reopen <fragment>",
	Short:             "Move a task from closed back to open",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeFragment(store.FilterClosed),
	RunE: func(cmd *cobra.Command, args []string) error {
		fragment := args[0]
		s, err := newStore()
		if err != nil {
			return err
		}
		t, err := s.Reopen(fragment)
		if err != nil {
			return handleResolveError(cmd, s, fragment, err)
		}
		cmd.Printf("reopened %s\n", t.ID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(reopenCmd)
}
