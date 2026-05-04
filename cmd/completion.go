package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/jvcorredor/bytheway/internal/store"
)

// completeFragment returns a cobra ValidArgsFunction that lists task
// ids from the store as completion candidates, with the task's first
// body line attached as the description so fish (and zsh) can render
// it next to the candidate in the completion menu. The filter argument
// selects which subset of tasks is offered: store.FilterOpen for
// `close`, store.FilterClosed for `reopen`, and store.FilterAll for
// commands that accept either.
//
// Only the first positional argument receives task candidates;
// subsequent arguments return no candidates and suppress filename
// fallback so a stray <Tab> does not flood the user with file
// completions in commands like `update <fragment> <text>`.
//
// Errors loading the store or listing tasks are swallowed: completion
// silently returns no candidates rather than dumping a stack trace
// into the user's shell.
func completeFragment(filter store.Filter) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		s, err := newStore()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		tasks, err := s.List(filter, 0, "")
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		out := make([]string, 0, len(tasks))
		for _, t := range tasks {
			desc := completionFirstLine(t.Body)
			if desc == "" {
				out = append(out, t.ID)
				continue
			}
			out = append(out, t.ID+"\t"+desc)
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
}

func completionFirstLine(body string) string {
	body = strings.TrimRight(body, "\n")
	if i := strings.IndexByte(body, '\n'); i >= 0 {
		return body[:i]
	}
	return body
}
