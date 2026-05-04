package cmd

import (
	"github.com/spf13/cobra"

	"github.com/jvcorredor/bytheway/internal/store"
)

var appendCmd = &cobra.Command{
	Use:               "append <fragment> <text>",
	Short:             "Append text to the body of a task",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: completeFragment(store.FilterAll),
	RunE: func(cmd *cobra.Command, args []string) error {
		fragment, text := args[0], args[1]
		s, err := newStore()
		if err != nil {
			return err
		}
		t, err := s.Append(fragment, text)
		if err != nil {
			return handleResolveError(cmd, s, fragment, err)
		}
		cmd.Printf("appended to %s\n", t.ID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(appendCmd)
}
