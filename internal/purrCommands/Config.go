package purrCommands

import (
	"Persephone/internal/ui"
	"Persephone/internal/utils"
	"fmt"
	"strings"
)

// ConfigCommand provides the entry point for the `purr config` CLI action.
//
// Operational Mechanics:
//  - Read Mode (1 argument): Displays the current value for the requested key (e.g. `purr config user.name`).
//  - Write Mode (>= 2 arguments): Updates the key's value and persists the change to the global `.purrconfig` file.
//    If the user inputs multiple unquoted values (e.g. `purr config user.name John Doe`), we reconstruct the name
//    by joining the remaining CLI arguments with spaces for convenience.
func ConfigCommand(args ...string) error {
	if len(args) == 0 {
		config, err := utils.ReadConfig()
		if err != nil {
			return fmt.Errorf("usage: purr config <key> [<value>]\n  Example: purr config user.name \"John Doe\"")
		}
		missingName := config.UserName == ""
		missingEmail := config.UserEmail == ""

		if missingName && missingEmail {
			return fmt.Errorf("user.name and user.email are not set — run:\npurr config user.name <value> && purr config user.email <value>")
		} else if missingName {
			return fmt.Errorf("user.name is not set — run:\npurr config user.name <value>")
		} else if missingEmail {
			return fmt.Errorf("user.email is not set — run:\npurr config user.email <value>")
		}
		
		fmt.Printf("user.name = %s\nuser.email = %s\n", config.UserName, config.UserEmail)
		return nil
	}

	configKey := args[0]

	if len(args) == 1 {
		return readConfig(configKey)
	}

	// Reconstruct potentially multi-word values (e.g. unquoted user names)
	configValue := strings.Join(args[1:], " ")
	return writeConfig(configKey, configValue)
}

// readConfig displays the current value of the given key from the global config file.
// It fails if the key is unrecognized, maintaining configuration schema safety.
func readConfig(key string) error {
	config, err := utils.ReadConfig()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	switch key {
	case "user.name":
		if config.UserName == "" {
			fmt.Println(ui.Metadata("user.name not set"))
		} else {
			fmt.Println(config.UserName)
		}
	case "user.email":
		if config.UserEmail == "" {
			fmt.Println(ui.Metadata("user.email not set"))
		} else {
			fmt.Println(config.UserEmail)
		}
	default:
		return fmt.Errorf("unknown config key: %s\nValid keys: user.name, user.email", key)
	}

	return nil
}

// writeConfig updates the specified key with the new value, creating a new global config file if missing.
func writeConfig(key, value string) error {
	config, err := utils.ReadConfig()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	switch key {
	case "user.name":
		config.UserName = value
		fmt.Printf("%s %s\n", ui.Metadata("Set user.name ="), value)
	case "user.email":
		config.UserEmail = value
		fmt.Printf("%s %s\n", ui.Metadata("Set user.email ="), value)
	default:
		return fmt.Errorf("unknown config key: %s\nValid keys: user.name, user.email", key)
	}

	if err := utils.WriteConfig(config); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

