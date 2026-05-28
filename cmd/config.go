package cmd

import (
	"Persephone/internal/purrCommands"
	"fmt"

	"github.com/spf13/cobra"
)

// Define the config subcommand
var configCmd = &cobra.Command{
	Use:   "config <key> [value]",
	Short: "Get and set repository or global options",
	Long: `Get and set user configuration values.

Examples:
  purr config user.name                  # Read user name
  purr config user.email                 # Read user email
  purr config user.name "John Doe"       # Set user name
  purr config user.email "john@example.com"  # Set user email`,
	Run: func(cmd *cobra.Command, args []string) {
		// This function runs when user types: purr config <key> [value]
		err := purrCommands.ConfigCommand(args...)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
