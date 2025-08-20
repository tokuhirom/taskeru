package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"taskeru/internal"
)

func InitConfigCommand() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	
	configPath := filepath.Join(homeDir, ".config", "taskeru", "config.toml")
	
	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Configuration file already exists at %s\n", configPath)
		fmt.Println("\nCurrent settings:")
		fmt.Println("=================")
		
		// Load and display current config
		config, _ := internal.LoadConfig()
		fmt.Printf("editor.add_timestamp = %v\n", config.Editor.AddTimestamp)
		
		return nil
	}
	
	// Create default config
	if err := internal.SaveDefaultConfig(); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	
	fmt.Printf("Created configuration file at %s\n", configPath)
	fmt.Println("\nDefault settings:")
	fmt.Println("=================")
	fmt.Println("editor.add_timestamp = false")
	fmt.Println("\nEdit this file to customize taskeru behavior.")
	fmt.Println("\nAvailable settings:")
	fmt.Println("  editor.add_timestamp - Add timestamps when editing tasks (true/false)")
	
	return nil
}