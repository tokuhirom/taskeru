package cmd

import (
	"fmt"
	"os"

	"taskeru/internal"
)

func InitConfigCommand() error {
	configPath, err := internal.UserConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Check if config already exists
	if content, err := os.ReadFile(configPath); err == nil {
		fmt.Printf("Configuration file already exists at %s\n", configPath)
		fmt.Println("\nCurrent settings:")
		fmt.Println("=================")
		fmt.Printf("%s\n", string(content))
	} else {
		// Create default config
		if err := internal.SaveDefaultConfig(); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}

		fmt.Printf("Created configuration file at %s\n", configPath)
	}

	return nil
}
