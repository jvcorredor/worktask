package cmd

import (
	"github.com/spf13/cobra"
)

var closeCmd = &cobra.Command{
	Use:   "close <fragment>",
	Short: "Move a task from open to closed and stamp completed",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fragment := args[0]
		s, err := newStore()
		if err != nil {
			return err
		}
		t, err := s.Close(fragment)
		if err != nil {
			return handleResolveError(cmd, s, fragment, err)
		}
		cmd.Printf("closed %s\n", t.ID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(closeCmd)
}
