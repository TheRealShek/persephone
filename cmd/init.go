package cmd

import (
	"Persephone/internal/purrCommands"
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
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println("Initialized empty repository")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
