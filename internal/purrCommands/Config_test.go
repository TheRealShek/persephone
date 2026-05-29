package purrCommands_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Persephone/internal/purrCommands"
	"Persephone/internal/utils"
)

// setConfigEnv is a helper that points PURR_CONFIG_PATH to an isolated temp file
// and returns the path so tests can read it back.
func setConfigEnv(t *testing.T) string {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), ".purrconfig")
	t.Setenv("PURR_CONFIG_PATH", configPath)
	return configPath
}

// readConfigFile is a helper that reads and unmarshals the config file at path.
func readConfigFile(t *testing.T, path string) utils.PurrConfig {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}
	var cfg utils.PurrConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}
	return cfg
}

func TestConfigCommand_NoArgs(t *testing.T) {
	_ = setConfigEnv(t)

	err := purrCommands.ConfigCommand()
	if err == nil {
		t.Fatal("expected error when called with no arguments, got nil")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("expected usage message in error, got: %s", err.Error())
	}
}

func TestConfigCommand_SetUserName(t *testing.T) {
	configPath := setConfigEnv(t)

	err := purrCommands.ConfigCommand("user.name", "Alice")
	if err != nil {
		t.Fatalf("unexpected error setting user.name: %v", err)
	}

	cfg := readConfigFile(t, configPath)
	if cfg.UserName != "Alice" {
		t.Errorf("user.name = %q, want %q", cfg.UserName, "Alice")
	}
}

func TestConfigCommand_SetUserEmail(t *testing.T) {
	configPath := setConfigEnv(t)

	err := purrCommands.ConfigCommand("user.email", "alice@example.com")
	if err != nil {
		t.Fatalf("unexpected error setting user.email: %v", err)
	}

	cfg := readConfigFile(t, configPath)
	if cfg.UserEmail != "alice@example.com" {
		t.Errorf("user.email = %q, want %q", cfg.UserEmail, "alice@example.com")
	}
}

func TestConfigCommand_ReadUserName(t *testing.T) {
	_ = setConfigEnv(t)

	// Set first so there is something to read
	if err := purrCommands.ConfigCommand("user.name", "Bob"); err != nil {
		t.Fatalf("setup: failed to set user.name: %v", err)
	}

	// Reading should succeed without error
	err := purrCommands.ConfigCommand("user.name")
	if err != nil {
		t.Errorf("unexpected error reading user.name: %v", err)
	}
}

func TestConfigCommand_ReadUserEmail(t *testing.T) {
	_ = setConfigEnv(t)

	if err := purrCommands.ConfigCommand("user.email", "bob@example.com"); err != nil {
		t.Fatalf("setup: failed to set user.email: %v", err)
	}

	err := purrCommands.ConfigCommand("user.email")
	if err != nil {
		t.Errorf("unexpected error reading user.email: %v", err)
	}
}

func TestConfigCommand_UnknownKey(t *testing.T) {
	_ = setConfigEnv(t)

	tests := []struct {
		name string
		args []string
	}{
		{name: "read unknown key", args: []string{"invalid.key"}},
		{name: "write unknown key", args: []string{"invalid.key", "value"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := purrCommands.ConfigCommand(tc.args...)
			if err == nil {
				t.Fatal("expected error for unknown config key, got nil")
			}
			if !strings.Contains(err.Error(), "unknown config key") {
				t.Errorf("expected 'unknown config key' in error, got: %s", err.Error())
			}
		})
	}
}

func TestConfigCommand_SetAndOverwrite(t *testing.T) {
	configPath := setConfigEnv(t)

	// First write
	if err := purrCommands.ConfigCommand("user.name", "FirstName"); err != nil {
		t.Fatalf("first set failed: %v", err)
	}

	// Overwrite with a different value
	if err := purrCommands.ConfigCommand("user.name", "SecondName"); err != nil {
		t.Fatalf("overwrite set failed: %v", err)
	}

	cfg := readConfigFile(t, configPath)
	if cfg.UserName != "SecondName" {
		t.Errorf("user.name = %q after overwrite, want %q", cfg.UserName, "SecondName")
	}
}

func TestConfigCommand_ReadUnsetValue(t *testing.T) {
	_ = setConfigEnv(t)

	// Config file does not exist yet — reading an unset value should not error.
	err := purrCommands.ConfigCommand("user.name")
	if err != nil {
		t.Errorf("expected no error when reading unset user.name, got: %v", err)
	}

	err = purrCommands.ConfigCommand("user.email")
	if err != nil {
		t.Errorf("expected no error when reading unset user.email, got: %v", err)
	}
}
