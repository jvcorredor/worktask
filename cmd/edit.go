package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/jvcorredor/bytheway/internal/config"
	"github.com/jvcorredor/bytheway/internal/store"
)

var editCmd = &cobra.Command{
	Use:               "edit <fragment>",
	Short:             "Open a task file in the configured editor",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeFragment(store.FilterAll),
	RunE: func(cmd *cobra.Command, args []string) error {
		fragment := args[0]
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		s := store.New(cfg.TasksDir)
		path, err := s.Path(fragment)
		if err != nil {
			return handleResolveError(cmd, s, fragment, err)
		}
		editor := cfg.ResolveEditor()
		fields := strings.Fields(editor)
		if len(fields) == 0 {
			return fmt.Errorf("edit: resolved editor is empty")
		}
		bin, err := exec.LookPath(fields[0])
		if err != nil {
			return fmt.Errorf("edit: locate editor %q: %w", fields[0], err)
		}
		argv := append(fields, path)
		cmd.PrintErrf("launching %s %s\n", editor, path)
		return syscall.Exec(bin, argv, os.Environ())
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
