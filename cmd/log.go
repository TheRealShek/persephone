package cmd

import (
	"persephone/internal/purrcommands"

	"github.com/spf13/cobra"
)

// logCmd is intentionally a thin CLI boundary. Commit parsing, ancestry validation, and rendering
// live in the command engine so future history views can reuse them without depending on Cobra.
var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show commit history",
	RunE: func(cmd *cobra.Command, args []string) error {
		return purrcommands.LogCommits(".", cmd.OutOrStdout())
	},
}

func init() {
	rootCmd.AddCommand(logCmd)
}
