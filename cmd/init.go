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
	Run: func(cmd *cobra.Command, args []string) {
		// This function runs when user types: purr init
		err := purrCommands.InitPurrDirectories(".")
		if err != nil {
			fmt.Println(ui.ErrorMessage(err))
			return
		}
		fmt.Println(ui.Successf("Initialized empty repository"))
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
