package cmd

import (
	"persephone/internal/purrcommands"

	"github.com/spf13/cobra"
)

// removeCmd removes tracked files from the index and working tree.
var removeCmd = &cobra.Command{
	Use:   "remove <file>...",
	Short: "Remove files",
	RunE: func(cmd *cobra.Command, args []string) error {
		return purrcommands.RemovePurrFiles(args...)
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
