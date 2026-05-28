package cmd

import (
	"Persephone/internal/purrCommands"
	"Persephone/internal/ui"
	"fmt"

	"github.com/spf13/cobra"
)

// Define the init subcommand
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		// This function runs when user types: purr init
		err := purrCommands.InitPurrDirectories(".")
		if err != nil {
			return err
		}
		fmt.Println(ui.Successf("Initialized empty repository"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
