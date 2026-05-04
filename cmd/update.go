package cmd

import (
	"github.com/spf13/cobra"

	"github.com/jvcorredor/bytheway/internal/store"
)

var updateCmd = &cobra.Command{
	Use:               "update <fragment> <new description>",
	Short:             "Replace the first body line of a task",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: completeFragment(store.FilterAll),
	RunE: func(cmd *cobra.Command, args []string) error {
		fragment, newDescription := args[0], args[1]
		s, err := newStore()
		if err != nil {
			return err
		}
		t, err := s.Update(fragment, newDescription)
		if err != nil {
			return handleResolveError(cmd, s, fragment, err)
		}
		cmd.Printf("updated %s\n", t.ID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
