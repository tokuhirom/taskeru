package internal

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the application configuration
type Config struct {
	Editor EditorConfig `toml:"editor"`
}

// EditorConfig contains editor-related settings
type EditorConfig struct {
	AddTimestamp bool `toml:"add_timestamp"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Editor: EditorConfig{
			AddTimestamp: false, // Default to false for opt-in behavior
		},
	}
}

// LoadConfig loads configuration from file or returns default
func LoadConfig() (*Config, error) {
	config := DefaultConfig()

	// Try to load from config file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config, nil // Return default config if can't get home dir
	}

	configPath := filepath.Join(homeDir, ".config", "taskeru", "config.toml")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Config file doesn't exist, use defaults
		return config, nil
	}

	// Load config from file
	if _, err := toml.DecodeFile(configPath, config); err != nil {
		// If there's an error reading the config, return default
		return DefaultConfig(), nil
	}

	return config, nil
}

// SaveDefaultConfig creates a default config file
func SaveDefaultConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(homeDir, ".config", "taskeru")
	configPath := filepath.Join(configDir, "config.toml")

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		return nil // Config already exists
	}

	// Create default config file
	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write default config with comments
	content := `# Taskeru Configuration File

[editor]
# Add timestamp when editing tasks
# When enabled, adds "## YYYY-MM-DD(Day) HH:MM" to notes
add_timestamp = false
`

	_, err = file.WriteString(content)
	return err
}
