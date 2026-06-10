package cmd

import (
	"Persephone/internal/purrCommands"

	"github.com/spf13/cobra"
)

// addCmd represents the `purr add` command execution tree.
//
// Controller Separation of Concerns:
// This command serves as a lightweight CLI front-end controller. It handles command routing,
// consumes argument parameters, catches operational errors, and delegates all processing, hashing,
// worker pooling, and index writes to `purrCommands.AddPurrFiles` within the decoupled commands engine.
var addCmd = &cobra.Command{
	Use:   "add <file|.>",
	Short: "Stage files",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := purrCommands.AddPurrFiles(args...)
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
