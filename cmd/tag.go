package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jvcorredor/worktask/internal/render"
	"github.com/jvcorredor/worktask/internal/store"
	"github.com/jvcorredor/worktask/internal/tty"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Mutate and discover tags on tasks",
}

var (
	tagLsAll    bool
	tagLsClosed bool
)

var tagLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List distinct tags in use, with counts",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := newStore()
		if err != nil {
			return err
		}
		filter := store.FilterOpen
		switch {
		case tagLsClosed:
			filter = store.FilterClosed
		case tagLsAll:
			filter = store.FilterAll
		}
		tagCounts, err := s.ListTags(filter)
		if err != nil {
			return err
		}
		switch format {
		case formatJSON:
			data, err := render.JSONTagList(tagCounts)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		case formatHuman:
			out := cmd.OutOrStdout()
			return render.HumanTagList(out, tagCounts, tty.IsTerminal(out))
		default:
			return fmt.Errorf("unknown format: %s", format)
		}
	},
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
	tagLsCmd.Flags().BoolVar(&tagLsAll, "all", false, "include closed tasks alongside open")
	tagLsCmd.Flags().BoolVar(&tagLsClosed, "closed", false, "scan closed tasks only")
	tagCmd.AddCommand(tagAddCmd)
	tagCmd.AddCommand(tagRmCmd)
	tagCmd.AddCommand(tagLsCmd)
	rootCmd.AddCommand(tagCmd)
}
