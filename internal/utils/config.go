package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GetConfigPath determines the absolute filesystem path for the global VCS configuration file.
// By default, the configuration is stored in the user's home directory as `~/.purrconfig`.
//
// Isolation & Testing Support:
// We support checking the `PURR_CONFIG_PATH` environment variable first. This provides an elegant,
// zero-overhead way to isolate test suites, docker sandboxes, or environment-specific profiles
// without writing to or polluting the developer's actual home directory file.
func GetConfigPath() (string, error) {
	if envPath := os.Getenv("PURR_CONFIG_PATH"); envPath != "" {
		return envPath, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".purrconfig"), nil
}

// ReadConfig parses and loads the configuration file into memory.
// If the config file does not exist, it defaults to returning a clean, empty configuration struct
// instead of failing. This allows the CLI to run unconfigured, only raising errors when
// identity-dependent commands like `purr commit` are executed.
func ReadConfig() (*PurrConfig, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &PurrConfig{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config PurrConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// WriteConfig serializes the configuration struct and writes it to disk.
// We use JSON with standard two-space indentation. This is an intentional choice:
// keeping the config readable and manually editable ensures a premium developer experience,
// while using JSON avoids the parser complexity of INI or YAML configuration packages.
func WriteConfig(config *PurrConfig) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

