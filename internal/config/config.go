package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the silibox configuration file structure.
type Config struct {
	Autosleep AutosleepConfig `yaml:"autosleep"`
}

// AutosleepConfig holds autosleep agent settings.
type AutosleepConfig struct {
	ContainerTimeout time.Duration `yaml:"container_timeout"`
	VMTimeout        time.Duration `yaml:"vm_timeout"`
	PollInterval     time.Duration `yaml:"poll_interval"`
	NoStopVM         bool          `yaml:"no_stop_vm"`
}

// DefaultConfig returns config with default values.
func DefaultConfig() Config {
	return Config{
		Autosleep: AutosleepConfig{
			ContainerTimeout: 15 * time.Minute,
			VMTimeout:        30 * time.Minute,
			PollInterval:     30 * time.Second,
			NoStopVM:         false,
		},
	}
}

// Load reads configuration from ~/.sili/config.yaml.
// If the file doesn't exist, returns default config without error.
func Load() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".sili", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist, return defaults
			return DefaultConfig(), nil
		}
		return Config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}
