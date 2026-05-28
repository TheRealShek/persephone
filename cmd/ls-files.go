package cmd

import (
	"Persephone/internal/purrCommands"
	"Persephone/internal/ui"
	"fmt"

	"github.com/spf13/cobra"
)

// Define the ls-files subcommand
var lsFilesCmd = &cobra.Command{
	Use:   "ls-files [flags]",
	Short: "Show information about files in the index",
	Long:  "Display the SHA-1 hash, mode, and path of all files currently staged in the .purr index.",
	Run: func(cmd *cobra.Command, args []string) {
		debug, _ := cmd.Flags().GetBool("debug")

		if err := purrCommands.ListFiles(".", debug); err != nil {
			fmt.Println(ui.ErrorMessage(err))
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(lsFilesCmd)
	lsFilesCmd.Flags().BoolP("debug", "d", false, "Show detailed metadata for each file")
}
