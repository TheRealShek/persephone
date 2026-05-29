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

// rootCmd represents the base command when called without any subcommands.
// It serves as the gateway to the CLI, routing inputs to child subcommands (`init`, `add`, `commit`, etc.).
var rootCmd = &cobra.Command{
	Use:   "purr",
	Short: "Persephone - A modern VCS built with Go",
	Long:  `Persephone is a version control system built in Go with performance and modern features in mind.`,
}

// Execute orchestrates command execution, handles standard error reporting, and configures CLI shell routing.
// This is the primary entry point called by main.main().
func Execute() {
	// Silence Cobra's default diagnostic printers: we catch and render errors manually
	// using the premium Terminal UI theme to maintain styling coherence.
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	// Intercept default text printers and inject our premium terminal color scheme
	setCustomHelp(rootCmd)

	err := rootCmd.Execute()
	if err != nil {
		var hintErr *ui.HintError
		if errors.As(err, &hintErr) {
			// Print actionable config suggestions (e.g. telling the user how to configure user.name)
			fmt.Println(ui.Hintf("%v", hintErr.Err))
			os.Exit(1)
		} else {
			fmt.Println(ui.Errorf("%v", err))
			os.Exit(1)
		}
	}
}

func init() {
	// Root-level global flags or configurations go here.
}

// setCustomHelp intercepts Cobra's default reflection-based help printer.
//
// Terminal UI Aesthetics & Layout:
// Rather than outputting default plain-text text, we override SetHelpFunc and SetUsageFunc
// to dynamically render help blocks with:
//  1. Semantic Theme: Uses Outfit/Inter typography-inspired terminal palettes (lipgloss-coded)
//     where headers, flags, descriptions, and commands are visually separated by color.
//  2. Aligned Columns: Available commands and flags are measured and formatted into aligned columns
//     (using `%-12s` and `%-14s` paddings) to allow instant scanning.
func setCustomHelp(rootCmd *cobra.Command) {
	helpFunc := func(cmd *cobra.Command, args []string) {
		out := cmd.OutOrStdout()
		ui.Enabled() // Initialize HSL color maps before formatting text buffers

		// Print long description if present, falling back to short description
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

		// Iterate through active subcommands and format them into an aligned list
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

		// Iterate through active flags, showing short and long forms aligned together
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

