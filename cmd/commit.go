package cmd

import (
	"Persephone/internal/purrCommands"
	"Persephone/internal/ui"
	"Persephone/internal/utils"
	"fmt"

	"github.com/spf13/cobra"
)

// Define the commit subcommand
var commitCmd = &cobra.Command{
	Use:   "commit -m [message]",
	Short: "Record changes to the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		message, _ := cmd.Flags().GetString("message")
		if message == "" {
			return fmt.Errorf("commit message is required. Use -m \"message\"")
		}

		userName, userEmail, err := utils.CheckConfigFile()
		if err != nil {
			return err
		}

		// Now call CommitPurrFiles with the required parameters
		err = purrCommands.CommitPurrFiles(".", message, userName, userEmail)
		if err != nil {
			return err
		}

		fmt.Println(ui.Successf("Changes committed successfully"))
		return nil
	},
}

func init() {
	// Add the -m flag for commit message
	commitCmd.Flags().StringP("message", "m", "", "Commit message")
	rootCmd.AddCommand(commitCmd)
}
