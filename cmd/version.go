package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	intversion "github.com/jvcorredor/bytheway/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the btw version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		info := intversion.Get()
		switch format {
		case formatJSON:
			payload := struct {
				Version string `json:"version"`
				Commit  string `json:"commit"`
				Date    string `json:"date"`
				Source  string `json:"source"`
			}{info.Version, info.Commit, info.Date, info.Source}
			data, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return err
			}
			data = append(data, '\n')
			_, err = cmd.OutOrStdout().Write(data)
			return err
		case formatHuman:
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "btw %s (commit %s, built %s)\n", info.Version, info.Commit, info.Date)
			return err
		default:
			return fmt.Errorf("unknown format: %s", format)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
