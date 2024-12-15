package pkg

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProjectConfig struct {
	RequiredTools   []string `yaml:"required_tools"`
	EnvironmentVars []string `yaml:"environment_vars"`
	RequiredTokens  []string `yaml:"required_tokens"`
	VersionControl  string   `yaml:"version_control"`
}

type Config struct {
	DefaultProject string                   `yaml:"default"`
	Projects       map[string]ProjectConfig `yaml:"projects"`
}

// LoadConfig loads the configuration file from ~/.config/procli/config.yaml
func LoadConfig() *Config {
	configFile := filepath.Join(os.Getenv("HOME"), ".config", "procli", "config.yaml")
	file, err := os.Open(configFile)
	if err != nil {
		// Return a config with an initialized map
		return &Config{Projects: make(map[string]ProjectConfig)}
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		// Ensure the map is initialized even if decoding fails
		return &Config{Projects: make(map[string]ProjectConfig)}
	}

	// Initialize the map if it's nil (e.g., empty file or incomplete config)
	if config.Projects == nil {
		config.Projects = make(map[string]ProjectConfig)
	}

	return &config
}

// SaveConfig saves the configuration to ~/.config/procli/config.yaml
func SaveConfig(config *Config) error {
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "procli")
	os.MkdirAll(configDir, os.ModePerm)
	configFile := filepath.Join(configDir, "config.yaml")

	file, err := os.Create(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	err = encoder.Encode(config)
	if err != nil {
		return err
	}

	return nil
}

// ParseCommaSeparated splits a comma-separated string into a slice
func ParseCommaSeparated(input string) []string {
	var result []string
	for _, item := range strings.Split(input, ",") {
		result = append(result, strings.TrimSpace(item))
	}
	return result
}
