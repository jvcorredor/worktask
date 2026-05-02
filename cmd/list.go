package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jvcorredor/worktask/internal/render"
	"github.com/jvcorredor/worktask/internal/store"
)

const defaultClosedLimit = 20

var (
	listAll    bool
	listClosed bool
	listLimit  int
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

		tasks, err := s.List(filter, limit)
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
			return render.HumanList(cmd.OutOrStdout(), tasks)
		default:
			return fmt.Errorf("unknown format: %s", format)
		}
	},
}

func init() {
	listCmd.Flags().BoolVar(&listAll, "all", false, "include closed tasks alongside open")
	listCmd.Flags().BoolVar(&listClosed, "closed", false, "show closed tasks only")
	listCmd.Flags().IntVar(&listLimit, "limit", 0, "cap on closed entries (default 20)")
	rootCmd.AddCommand(listCmd)
}
