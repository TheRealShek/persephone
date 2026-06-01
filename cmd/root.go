/*
Copyright © 2025 [Abhishek Thakur](https://github.com/TheRealShek/persephone)
*/
package cmd

import (
	"Persephone/internal/ui"
	"errors"
	"fmt"
	"os"

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
	// NOTE: The default Cobra `--toggle` flag was intentionally removed. Do not re-add it.
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
		ui.Enabled() // Initialize HSL color maps

		// Tagline
		fmt.Fprintln(out, ui.HelpTagline("Persephone — distributed VCS built in Go"))
		fmt.Fprintln(out)

		if cmd.Runnable() || cmd.HasAvailableSubCommands() {
			fmt.Fprintln(out, ui.HelpSection("Usage"))
			fmt.Fprintf(out, "  %s\n\n", cmd.UseLine())
		}

		if cmd.HasAvailableSubCommands() {
			fmt.Fprintln(out, ui.HelpSection("Available Commands"))

			getCmd := func(name string) *cobra.Command {
				for _, c := range cmd.Commands() {
					if c.Name() == name {
						return c
					}
				}
				return nil
			}

			repoCmds := []string{"init", "config"}
			repoFound := false
			for _, name := range repoCmds {
				if getCmd(name) != nil {
					repoFound = true
					break
				}
			}
			if repoFound {
				fmt.Fprintln(out, ui.HelpGroup("Repository"))
				for _, name := range repoCmds {
					if c := getCmd(name); c != nil {
						fmt.Fprint(out, ui.HelpCommand(c.Name(), c.Short, c.UseLine())+"\n")
					}
				}
				fmt.Fprintln(out)
			}

			indexCmds := []string{"add", "ls", "commit"}
			indexFound := false
			for _, name := range indexCmds {
				if getCmd(name) != nil {
					indexFound = true
					break
				}
			}
			if indexFound {
				fmt.Fprintln(out, ui.HelpGroup("Index"))
				for _, name := range indexCmds {
					if c := getCmd(name); c != nil {
						fmt.Fprint(out, ui.HelpCommand(c.Name(), c.Short, c.UseLine())+"\n")
					}
				}
				fmt.Fprintln(out)
			}

			historyCmds := []string{"log"}
			historyFound := false
			for _, name := range historyCmds {
				if getCmd(name) != nil {
					historyFound = true
					break
				}
			}
			if historyFound {
				fmt.Fprintln(out, ui.HelpGroup("History"))
				for _, name := range historyCmds {
					if c := getCmd(name); c != nil {
						fmt.Fprint(out, ui.HelpCommand(c.Name(), c.Short, c.UseLine())+"\n")
					}
				}
				fmt.Fprintln(out)
			}
		}

		if cmd.HasAvailableFlags() {
			fmt.Fprintln(out, ui.HelpSection("Flags"))
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
				fmt.Fprint(out, ui.HelpFlag(flagName, f.Usage)+"\n")
			})
			fmt.Fprintln(out)
		}

		fmt.Fprintln(out, ui.HelpFooter("purr [command] --help for more info"))
	}

	rootCmd.SetHelpFunc(helpFunc)
	rootCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		helpFunc(cmd, nil)
		return nil
	})
}
