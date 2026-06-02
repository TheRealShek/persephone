package cmd

import (
	"Persephone/internal/purrCommands"
	"Persephone/internal/ui"
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

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
		if err := purrCommands.InitPurrDirectories("."); err != nil {
			if errors.Is(err, purrCommands.ErrRepositoryAlreadyInitialized) {
				confirmed, confirmErr := confirmReinitialize(cmd.InOrStdin(), cmd.OutOrStdout())
				if confirmErr != nil {
					return confirmErr
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), ui.Metadata("Reinitialization cancelled"))
					return nil
				}
				if err := purrCommands.ReinitializePurrDirectories("."); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), ui.Successf("Reinitialized existing repository"))
				return nil
			}
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), ui.Successf("Initialized empty repository"))
		return nil
	},
}

// confirmReinitialize keeps interactive policy at the CLI boundary. The core package only performs
// filesystem work after the caller has made an explicit decision, which keeps automated callers safe.
func confirmReinitialize(in io.Reader, out io.Writer) (bool, error) {
	scanner := bufio.NewScanner(in)
	for {
		fmt.Fprint(out, ui.Warningf("Repository already exists. Reinitialize it? [y/N] "))
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return false, fmt.Errorf("failed to read reinitialization confirmation: %w", err)
			}
			return false, nil
		}

		switch strings.ToLower(strings.TrimSpace(scanner.Text())) {
		case "y", "yes":
			return true, nil
		case "", "n", "no":
			return false, nil
		default:
			fmt.Fprintln(out, ui.Warningf("Please answer yes or no."))
		}
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
}
