package cmd

import (
	"Persephone/internal/objects"
	"Persephone/internal/purrCommands"
	"Persephone/internal/ui"

	"fmt"

	"github.com/spf13/cobra"
)

// commitCmd represents the `purr commit` command execution tree.
//
// Operational Controls:
// This wrapper extracts and validates required VCS context before committing:
//  1. Message Enforcement: Requires the `-m` flag to prevent empty commit logs.
//  2. Pre-flight Config Verification: Checks that global config identity parameters
//     (name, email) are set, throwing actionable configuration suggestions if missing.
//  3. Decoupled Processing: Forwards data to the command engine to assemble object nodes.
var commitCmd = &cobra.Command{
	Use:                   "commit -m \"message\"",
	Short:                 "Record changes",
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		message, _ := cmd.Flags().GetString("message")
		if message == "" {
			return ui.NewHintError(fmt.Errorf("commit message is required. Use -m \"message\""))
		}

		userName, userEmail, err := objects.CheckConfigFile()
		if err != nil {
			return err
		}

		err = purrCommands.CommitPurrFiles(".", message, userName, userEmail)
		if err != nil {
			return err
		}

		fmt.Println(ui.Successf("Changes committed successfully"))
		return nil
	},
}

func init() {
	commitCmd.Flags().StringP("message", "m", "", "Commit message")
	rootCmd.AddCommand(commitCmd)
}
