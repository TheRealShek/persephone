package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GetConfigPath returns the path to the .purrconfig file in the user's home directory
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

// ReadConfig reads the .purrconfig file and returns the configuration
// If the file doesn't exist, returns an empty config without error
func ReadConfig() (*PurrConfig, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return empty config if file doesn't exist
		return &PurrConfig{}, nil
	}

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var config PurrConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// WriteConfig writes the configuration to the .purrconfig file
func WriteConfig(config *PurrConfig) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Marshal config to JSON with indentation for readability
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	// Write to file with appropriate permissions
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
