package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	ConfigFileName = "config.yaml"
	ConfigDirName  = ".tod"
	GlobalConfigDir = ".config/tod"
)

// Loader handles configuration loading and discovery
type Loader struct {
	startDir string
}

// NewLoader creates a new config loader starting from the given directory
func NewLoader(startDir string) *Loader {
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			startDir = "."
		}
	}
	
	return &Loader{
		startDir: startDir,
	}
}

// Load loads the configuration with environment variable overrides
func (l *Loader) Load() (*Config, error) {
	// Find the config file
	configPath, err := l.findConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to find config file: %w", err)
	}
	
	// Load the config
	config, err := l.loadFromFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from %s: %w", configPath, err)
	}
	
	// Apply environment variable overrides
	if err := l.applyEnvOverrides(config); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}
	
	// Validate the final configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	return config, nil
}

// findConfigFile searches upward from the start directory for a config file
func (l *Loader) findConfigFile() (string, error) {
	dir := l.startDir
	
	for {
		// Check for local .tod/config.yaml
		configPath := filepath.Join(dir, ConfigDirName, ConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
		
		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}
	
	// Try global config
	homeDir, err := os.UserHomeDir()
	if err == nil {
		globalConfig := filepath.Join(homeDir, GlobalConfigDir, ConfigFileName)
		if _, err := os.Stat(globalConfig); err == nil {
			return globalConfig, nil
		}
	}
	
	return "", fmt.Errorf("no config file found (searched upward from %s)", l.startDir)
}

// loadFromFile loads configuration from a YAML file
func (l *Loader) loadFromFile(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	
	return &config, nil
}

// applyEnvOverrides applies environment variable overrides to the config
func (l *Loader) applyEnvOverrides(config *Config) error {
	// Native environment variable handling (faster than viper)
	
	// AI configuration overrides
	// Support both TOD_AI_API_KEY and OPENAI_API_KEY for convenience
	if apiKey := os.Getenv("TOD_AI_API_KEY"); apiKey != "" {
		config.AI.APIKey = apiKey
	} else if config.AI.Provider == "openai" && config.AI.APIKey == "" {
		// If provider is OpenAI and no key is set, check for standard OPENAI_API_KEY
		if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
			config.AI.APIKey = apiKey
		}
	}
	if provider := os.Getenv("TOD_AI_PROVIDER"); provider != "" {
		config.AI.Provider = provider
	}
	if model := os.Getenv("TOD_AI_MODEL"); model != "" {
		config.AI.Model = model
	}
	if endpoint := os.Getenv("TOD_AI_ENDPOINT"); endpoint != "" {
		config.AI.Endpoint = endpoint
	}
	
	// Testing configuration overrides
	if framework := os.Getenv("TOD_TESTING_FRAMEWORK"); framework != "" {
		config.Testing.Framework = framework
	}
	if version := os.Getenv("TOD_TESTING_VERSION"); version != "" {
		config.Testing.Version = version
	}
	if testDir := os.Getenv("TOD_TESTING_TEST_DIR"); testDir != "" {
		config.Testing.TestDir = testDir
	}
	
	// Current environment override
	if currentEnv := os.Getenv("TOD_CURRENT_ENV"); currentEnv != "" {
		config.Current = currentEnv
	}
	
	return nil
}

// Save saves the configuration to the specified path
func (l *Loader) Save(config *Config, configPath string) error {
	// Update the metadata
	config.Meta.UpdatedAt = time.Now()
	
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// GetConfigPath returns the path where a config file should be created
func (l *Loader) GetConfigPath() string {
	return filepath.Join(l.startDir, ConfigDirName, ConfigFileName)
}

// IsInitialized checks if a config file exists in the project hierarchy
func (l *Loader) IsInitialized() bool {
	_, err := l.findConfigFile()
	return err == nil
}

// GetProjectRoot returns the root directory containing the .tod folder
func (l *Loader) GetProjectRoot() (string, error) {
	configPath, err := l.findConfigFile()
	if err != nil {
		return "", err
	}
	
	// Return the directory containing the .tod folder
	return filepath.Dir(filepath.Dir(configPath)), nil
}