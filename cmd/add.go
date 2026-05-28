package cmd

import (
	"Persephone/internal/purrCommands"
	"Persephone/internal/ui"
	"fmt"

	"github.com/spf13/cobra"
)

// Define the add subcommand
var addCmd = &cobra.Command{
	Use:   "add [files]",
	Short: "Add file contents to the index",
	RunE: func(cmd *cobra.Command, args []string) error {
		// This function runs when user types: purr add <files>
		err := purrCommands.AddPurrFiles(args...)
		if err != nil {
			return err
		}
		fmt.Println(ui.Successf("Files added to index"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
