package cmd

import (
	"Persephone/internal/purrCommands"
	"Persephone/internal/ui"
	"fmt"

	"github.com/spf13/cobra"
)

// initCmd represents the `purr init` command execution tree.
//
// Operational Context:
// Directs workspace bootstrapping by triggering the creation of VCS structures
// (the `.purr` repository database directory, refs catalog, objects store, and empty index).
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Init repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		isNewRepo, err := purrCommands.InitPurrDirectories(".")
		if err != nil {
			return err
		}
		if isNewRepo {
			fmt.Println(ui.Successf("Initialized empty repository"))
		} else {
			fmt.Println(ui.Successf("Reinitialized existing repository"))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

