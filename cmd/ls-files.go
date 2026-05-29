package cmd

import (
	"Persephone/internal/purrCommands"

	"github.com/spf13/cobra"
)

// Define the ls subcommand
var lsFilesCmd = &cobra.Command{
	Use:   "ls [flags]",
	Short: "Show information about files in the index",
	Long:  "Display the SHA-1 hash, mode, and path of all files currently staged in the .purr index.",
	RunE: func(cmd *cobra.Command, args []string) error {
		debug, _ := cmd.Flags().GetBool("debug")

		if err := purrCommands.ListFiles(".", debug); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lsFilesCmd)
	lsFilesCmd.Flags().BoolP("debug", "d", false, "Show detailed metadata for each file")
}
