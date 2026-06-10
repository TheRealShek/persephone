package cmd

import (
	"Persephone/internal/purrCommands"

	"github.com/spf13/cobra"
)

// lsFilesCmd represents the `purr ls` command execution tree.
//
// Operational Context:
// Extracts visual parameters (such as the `--debug` flag) and triggers index rendering.
// Keeping this wrapper simple isolates CLI parsing details from formatting logic.
var lsFilesCmd = &cobra.Command{
	Use:                   "ls",
	Short:                 "Show staged files",
	DisableFlagsInUseLine: true,
	Long:                  "Display the SHA-1 hash, mode, and path of all files currently staged in the .purr index.",
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
