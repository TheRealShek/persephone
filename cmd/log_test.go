package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogCommandRegisteredInHistoryHelp(t *testing.T) {
	command, _, err := rootCmd.Find([]string{"log"})
	if err != nil {
		t.Fatalf("rootCmd.Find(log) error = %v", err)
	}
	if command != logCmd {
		t.Fatalf("rootCmd.Find(log) = %v, want logCmd", command)
	}

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	setCustomHelp(rootCmd)
	if err := rootCmd.Help(); err != nil {
		t.Fatalf("rootCmd.Help() error = %v", err)
	}

	help := out.String()
	if !strings.Contains(help, "History") || !strings.Contains(help, "log") {
		t.Fatalf("root help does not list log in History group:\n%s", help)
	}
}
