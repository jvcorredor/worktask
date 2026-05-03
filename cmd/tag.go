package cmd

import (
	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Mutate tags on existing tasks",
}

var tagAddCmd = &cobra.Command{
	Use:   "add <fragment> <tag>",
	Short: "Add a tag to an existing task",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		fragment, rawTag := args[0], args[1]
		s, err := newStore()
		if err != nil {
			return err
		}
		t, err := s.AddTag(fragment, rawTag)
		if err != nil {
			return handleResolveError(cmd, s, fragment, err)
		}
		cmd.Printf("tagged %s with %s\n", t.ID, rawTag)
		return nil
	},
}

var tagRmCmd = &cobra.Command{
	Use:   "rm <fragment> <tag>",
	Short: "Remove a tag from an existing task",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		fragment, rawTag := args[0], args[1]
		s, err := newStore()
		if err != nil {
			return err
		}
		t, err := s.RemoveTag(fragment, rawTag)
		if err != nil {
			return handleResolveError(cmd, s, fragment, err)
		}
		cmd.Printf("removed %s from %s\n", rawTag, t.ID)
		return nil
	},
}

func init() {
	tagCmd.AddCommand(tagAddCmd)
	tagCmd.AddCommand(tagRmCmd)
	rootCmd.AddCommand(tagCmd)
}
