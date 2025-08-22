package internal

import (
	"os"
	"path/filepath"
	"runtime"

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

func UserConfigPath() (string, error) {
	if runtime.GOOS == "windows" || runtime.GOOS == "linux" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return "", nil // Return default config if can't get config dir
		}
		return filepath.Join(configDir, "taskeru", "config.toml"), nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", nil // Return default config if can't get home dir
	}

	return filepath.Join(homeDir, ".config", "taskeru", "config.toml"), nil
}

// LoadConfig loads configuration from file or returns default
func LoadConfig() (*Config, error) {
	config := DefaultConfig()

	// Try to load from config file
	configPath, err := UserConfigPath()
	if err != nil {
		return config, nil // Return default config if can't get home dir
	}

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
	configPath, err := UserConfigPath()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
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
	defer func() { _ = file.Close() }()

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
