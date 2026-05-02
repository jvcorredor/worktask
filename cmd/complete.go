package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var completeCmd = &cobra.Command{
	Use:   "complete <fragment>",
	Short: "Move a task from open to closed and stamp completed",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fragment := args[0]
		s, err := newStore()
		if err != nil {
			return err
		}
		t, err := s.Complete(fragment)
		if err != nil {
			return handleResolveError(cmd, s, fragment, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "completed %s\n", t.ID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completeCmd)
}
