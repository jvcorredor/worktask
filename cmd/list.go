package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jvcorredor/worktask/internal/render"
	"github.com/jvcorredor/worktask/internal/store"
	"github.com/jvcorredor/worktask/internal/tag"
	"github.com/jvcorredor/worktask/internal/tty"
)

const defaultClosedLimit = 20

var (
	listAll    bool
	listClosed bool
	listLimit  int
	listTag    string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks (open by default)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := newStore()
		if err != nil {
			return err
		}

		filter := store.FilterOpen
		switch {
		case listClosed:
			filter = store.FilterClosed
		case listAll:
			filter = store.FilterAll
		}

		limit := defaultClosedLimit
		if listLimit > 0 {
			limit = listLimit
		}

		var normalizedTag string
		if listTag != "" {
			normalizedTag = tag.Normalize(listTag)
			if err := tag.Validate(normalizedTag); err != nil {
				return fmt.Errorf("invalid --tag %q: %w", listTag, err)
			}
		}

		tasks, err := s.List(filter, limit, normalizedTag)
		if err != nil {
			return err
		}
		switch format {
		case formatJSON:
			data, err := render.JSONList(tasks)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		case formatHuman:
			out := cmd.OutOrStdout()
			return render.HumanList(out, tasks, tty.IsTerminal(out))
		default:
			return fmt.Errorf("unknown format: %s", format)
		}
	},
}

func init() {
	listCmd.Flags().BoolVar(&listAll, "all", false, "include closed tasks alongside open")
	listCmd.Flags().BoolVar(&listClosed, "closed", false, "show closed tasks only")
	listCmd.Flags().IntVar(&listLimit, "limit", 0, "cap on closed entries (default 20)")
	listCmd.Flags().StringVar(&listTag, "tag", "", "filter to tasks carrying this tag (exact match)")
	rootCmd.AddCommand(listCmd)
}
