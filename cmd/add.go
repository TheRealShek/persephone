package cmd

import (
	"Persephone/internal/purrCommands"
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
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println("Files added to index")
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
