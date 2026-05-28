package cmd

import (
	"Persephone/internal/purrCommands"
	"Persephone/internal/utils"
	"fmt"
	"github.com/spf13/cobra"
)

// Define the commit subcommand
var commitCmd = &cobra.Command{
	Use:   "commit -m [message]",
	Short: "Record changes to the repository",
	Run: func(cmd *cobra.Command, args []string) {
		message, _ := cmd.Flags().GetString("message")
		if message == "" {
			fmt.Println("Error: commit message is required. Use -m \"message\"")
			return
		}

		userName, userEmail, err := utils.CheckConfigFile()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		// Now call CommitPurrFiles with the required parameters
		err = purrCommands.CommitPurrFiles(".", message, userName, userEmail)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Println("Changes committed successfully")
	},
}

func init() {
	// Add the -m flag for commit message
	commitCmd.Flags().StringP("message", "m", "", "Commit message")
	rootCmd.AddCommand(commitCmd)
}
