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
	Run: func(cmd *cobra.Command, args []string) {
		// This function runs when user types: purr add <files>
		err := purrCommands.AddPurrFiles(args...)
		if err != nil {
			fmt.Println(ui.ErrorMessage(err))
			return
		}
		fmt.Println(ui.Successf("Files added to index"))
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
