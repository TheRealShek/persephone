package cmd

import (
	"Persephone/internal/purrCommands"

	"github.com/spf13/cobra"
)

// configCmd represents the `purr config` command execution tree.
//
// Operational Context:
// Provides the CLI wrapper for setting identity keys in the developer's global config file.
// It forwards execution parameters directly to the config command module to perform disk I/O.
var configCmd = &cobra.Command{
	Use:   "config <key> <value>",
	Short: "Get and set options",
	Long: `Get and set user configuration values.

Examples:
  purr config user.name                  # Read user name
  purr config user.email                 # Read user email
  purr config user.name "John Doe"       # Set user name
  purr config user.email "john@example.com"  # Set user email`,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := purrCommands.ConfigCommand(args...)
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}

