package cmd

import (
	"github.com/spf13/cobra"
)

var addTags []string

var addCmd = &cobra.Command{
	Use:   "add <description>",
	Short: "Create a new open task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := newStore()
		if err != nil {
			return err
		}
		t, err := s.Add(args[0], addTags...)
		if err != nil {
			return err
		}
		cmd.Printf("added %s\n", t.ID)
		return nil
	},
}

func init() {
	addCmd.Flags().StringArrayVar(&addTags, "tag", nil, "tag for the task (repeatable)")
	rootCmd.AddCommand(addCmd)
}
