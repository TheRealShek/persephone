/*
Copyright © 2025 [Abhishek Thakur](https://github.com/TheRealShek/persephone)
*/
package cmd

import (
	"Persephone/internal/ui"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "purr",
	Short: "Persephone - A modern VCS built with Go",
	Long:  `Persephone is a version control system built in Go with performance and modern features in mind.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	// Enable beautiful dynamic terminal help colorization
	setCustomHelp(rootCmd)

	err := rootCmd.Execute()
	if err != nil {
		var hintErr *ui.HintError
		if errors.As(err, &hintErr) {
			fmt.Println(ui.Hintf("%v", hintErr.Err))
			os.Exit(1)
		} else {
			fmt.Println(ui.Errorf("%v", err))
			os.Exit(1)
		}
	}
}

func init() {
	// Here you will define your flags and configuration settings.
}

// setCustomHelp sets the HelpFunc and UsageFunc using ui styling
func setCustomHelp(rootCmd *cobra.Command) {
	helpFunc := func(cmd *cobra.Command, args []string) {
		out := cmd.OutOrStdout()
		ui.Enabled() // Initialize UI styles before any rendering

		if cmd.Long != "" {
			fmt.Fprintln(out, ui.Metadata(strings.TrimSpace(cmd.Long)))
			fmt.Fprintln(out)
		} else if cmd.Short != "" {
			fmt.Fprintln(out, ui.Metadata(strings.TrimSpace(cmd.Short)))
			fmt.Fprintln(out)
		}

		if cmd.Runnable() || cmd.HasAvailableSubCommands() {
			fmt.Fprintln(out, ui.SectionHeader("Usage:"))
			fmt.Fprintf(out, "  %s\n\n", cmd.UseLine())
		}

		if cmd.HasAvailableSubCommands() {
			fmt.Fprintln(out, ui.SectionHeader("Available Commands:"))
			for _, c := range cmd.Commands() {
				if !c.IsAvailableCommand() || c.Name() == "help" || c.Name() == "completion" {
					continue
				}
				paddedName := fmt.Sprintf("%-12s", c.Name())
				fmt.Fprintf(out, "  %s %s\n", ui.Added(paddedName), ui.Metadata(c.Short))
			}
			fmt.Fprintln(out)
		}

		if cmd.HasAvailableFlags() {
			fmt.Fprintln(out, ui.SectionHeader("Flags:"))
			cmd.Flags().VisitAll(func(f *pflag.Flag) {
				if f.Hidden {
					return
				}
				var flagName string
				if f.Shorthand != "" {
					flagName = fmt.Sprintf("-%s, --%s", f.Shorthand, f.Name)
				} else {
					flagName = fmt.Sprintf("    --%s", f.Name)
				}
				paddedFlag := fmt.Sprintf("%-14s", flagName)
				fmt.Fprintf(out, "  %s %s\n", ui.Info(paddedFlag), ui.Metadata(f.Usage))
			})
			fmt.Fprintln(out)
		}

		if cmd.HasAvailableSubCommands() {
			footer := fmt.Sprintf("Use \"%s [command] --help\" for more information about a command.", cmd.CommandPath())
			fmt.Fprintln(out, ui.Metadata(footer))
		}
	}

	rootCmd.SetHelpFunc(helpFunc)
	rootCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		helpFunc(cmd, nil)
		return nil
	})
}
