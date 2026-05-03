package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jvcorredor/worktask/internal/render"
	"github.com/jvcorredor/worktask/internal/store"
	"github.com/jvcorredor/worktask/internal/tty"
)

var showCmd = &cobra.Command{
	Use:   "show <fragment>",
	Short: "Print a single task by id or description fragment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fragment := args[0]
		s, err := newStore()
		if err != nil {
			return err
		}

		t, raw, path, err := s.Get(fragment)
		if err != nil {
			return handleResolveError(cmd, s, fragment, err)
		}

		switch format {
		case formatJSON:
			data, err := render.JSONShow(t, path)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		case formatHuman:
			out := cmd.OutOrStdout()
			return render.HumanShow(out, t, raw, path, tty.IsTerminal(out))
		default:
			return fmt.Errorf("unknown format: %s", format)
		}
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}

func handleResolveError(cmd *cobra.Command, s *store.Store, fragment string, err error) error {
	var amb *store.ErrAmbiguous
	var nm *store.ErrNoMatch
	switch {
	case errors.As(err, &amb):
		if format == formatJSON {
			data, jerr := render.JSONErrorAmbiguous(fragment, amb.Candidates)
			if jerr != nil {
				return jerr
			}
			if _, werr := cmd.OutOrStdout().Write(data); werr != nil {
				return werr
			}
		} else {
			cmd.PrintErrf("ambiguous fragment %q. matches:\n", fragment)
			stderr := cmd.ErrOrStderr()
			if rerr := render.HumanCandidates(stderr, amb.Candidates, tty.IsTerminal(stderr)); rerr != nil {
				return rerr
			}
		}
		return errExit
	case errors.As(err, &nm):
		openTasks, lerr := s.List(store.FilterOpen, 0)
		if lerr != nil {
			return lerr
		}
		if format == formatJSON {
			data, jerr := render.JSONErrorNoMatch(fragment, openTasks)
			if jerr != nil {
				return jerr
			}
			if _, werr := cmd.OutOrStdout().Write(data); werr != nil {
				return werr
			}
		} else {
			cmd.PrintErrf("no match for %q. open tasks:\n", fragment)
			stderr := cmd.ErrOrStderr()
			if rerr := render.HumanList(stderr, openTasks, tty.IsTerminal(stderr)); rerr != nil {
				return rerr
			}
		}
		return errExit
	default:
		return err
	}
}
