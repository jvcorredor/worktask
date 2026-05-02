package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <description>",
	Short: "Create a new open task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := newStore()
		if err != nil {
			return err
		}
		t, err := s.Add(args[0])
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "added %s\n", t.ID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
