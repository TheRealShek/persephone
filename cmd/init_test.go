package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfirmReinitialize(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		want       bool
		wantPrompt string
	}{
		{name: "yes short", input: "y\n", want: true},
		{name: "yes full", input: "YES\n", want: true},
		{name: "no short", input: "n\n", want: false},
		{name: "default no", input: "\n", want: false},
		{name: "EOF defaults no", input: "", want: false},
		{name: "invalid then yes", input: "maybe\nyes\n", want: true, wantPrompt: "Please answer yes or no."},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			got, err := confirmReinitialize(strings.NewReader(tc.input), &out)
			if err != nil {
				t.Fatalf("confirmReinitialize() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("confirmReinitialize() = %v, want %v", got, tc.want)
			}
			if !strings.Contains(out.String(), "Repository already exists") {
				t.Fatalf("confirmation prompt missing warning: %q", out.String())
			}
			if tc.wantPrompt != "" && !strings.Contains(out.String(), tc.wantPrompt) {
				t.Fatalf("confirmation output = %q, want %q", out.String(), tc.wantPrompt)
			}
		})
	}
}
