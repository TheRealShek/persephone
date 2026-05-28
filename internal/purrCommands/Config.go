package purrCommands

import (
	"Persephone/internal/utils"
	"fmt"
	"strings"
)

// ConfigCommand handles the `purr config` command
// Usage:
//   - purr config user.name                  -> read user name
//   - purr config user.email                 -> read user email
//   - purr config user.name "John Doe"       -> set user name
//   - purr config user.email "john@example.com" -> set user email
func ConfigCommand(args ...string) error {
	// Check if we have at least one argument (the config key)
	if len(args) == 0 {
		return fmt.Errorf("usage: purr config <key> [<value>]\n  Example: purr config user.name \"John Doe\"")
	}

	configKey := args[0]

	// Read mode: no value provided
	if len(args) == 1 {
		return readConfig(configKey)
	}

	// Write mode: value provided
	configValue := strings.Join(args[1:], " ")
	return writeConfig(configKey, configValue)
}

// readConfig reads and displays a specific config value
func readConfig(key string) error {
	config, err := utils.ReadConfig()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	switch key {
	case "user.name":
		if config.UserName == "" {
			fmt.Println("user.name not set")
		} else {
			fmt.Println(config.UserName)
		}
	case "user.email":
		if config.UserEmail == "" {
			fmt.Println("user.email not set")
		} else {
			fmt.Println(config.UserEmail)
		}
	default:
		return fmt.Errorf("unknown config key: %s\nValid keys: user.name, user.email", key)
	}

	return nil
}

// writeConfig updates a specific config value and saves to disk
func writeConfig(key, value string) error {
	// Read existing config (or get empty config if file doesn't exist)
	config, err := utils.ReadConfig()
	if err != nil {
		// If config doesn't exist, start with empty config
		config = &utils.PurrConfig{}
	}

	// Update the appropriate field
	switch key {
	case "user.name":
		config.UserName = value
		fmt.Printf("Set user.name = %s\n", value)
	case "user.email":
		config.UserEmail = value
		fmt.Printf("Set user.email = %s\n", value)
	default:
		return fmt.Errorf("unknown config key: %s\nValid keys: user.name, user.email", key)
	}

	// Write updated config to disk
	if err := utils.WriteConfig(config); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
