package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Loader handles configuration loading.
type Loader struct {
	configHome string
}

// NewLoader creates a new configuration loader.
func NewLoader() *Loader {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, _ := os.UserHomeDir()
		configHome = filepath.Join(home, ".config")
	}
	return &Loader{
		configHome: configHome,
	}
}

// LoadGlobalConfig loads the global dox configuration.
func (l *Loader) LoadGlobalConfig() (*GlobalConfig, error) {
	globalViper := viper.New()
	globalViper.SetConfigFile(filepath.Join(l.configHome, "dox", "config.yaml"))
	globalViper.SetDefault("runtime", "docker")

	config := &GlobalConfig{
		Runtime: "docker", // Set default value directly.
	}
	
	if err := globalViper.ReadInConfig(); err != nil {
		// It's okay if global config doesn't exist, use defaults.
		// Just return the default config.
		return config, nil
	}

	if err := globalViper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal global config: %w", err)
	}

	return config, nil
}

// LoadCommandConfig loads configuration for a specific command.
func (l *Loader) LoadCommandConfig(command string) (*CommandConfig, error) {
	commandViper := viper.New()
	configPath := filepath.Join(l.configHome, "dox", "commands", command+".yaml")
	commandViper.SetConfigFile(configPath)

	if err := commandViper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("command '%s' doesn't exist. Create %s", command, configPath)
		}
		return nil, fmt.Errorf("failed to read command config: %w", err)
	}

	config := &CommandConfig{}
	if err := commandViper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal command config: %w", err)
	}

	// Validate required fields.
	if config.Image == "" && (config.Build == nil || config.Build.DockerfileInline == "") {
		return nil, fmt.Errorf("configuration missing required field: image or build.dockerfile_inline")
	}

	// Expand environment variables in volume paths.
	for i, volume := range config.Volumes {
		config.Volumes[i] = l.expandVolumePath(volume)
	}

	return config, nil
}

// ListCommands returns a list of available commands.
func (l *Loader) ListCommands() ([]string, error) {
	commandsDir := filepath.Join(l.configHome, "dox", "commands")
	entries, err := os.ReadDir(commandsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read commands directory: %w", err)
	}

	var commands []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			command := strings.TrimSuffix(entry.Name(), ".yaml")
			commands = append(commands, command)
		}
	}

	return commands, nil
}

// expandVolumePath expands environment variables and special paths in volume mount strings.
func (l *Loader) expandVolumePath(volume string) string {
	parts := strings.SplitN(volume, ":", 3)
	if len(parts) < 2 {
		return volume
	}

	source := parts[0]
	
	// Handle special case for current directory.
	if source == "." {
		cwd, _ := os.Getwd()
		source = cwd
	} else {
		// Expand environment variables.
		source = os.ExpandEnv(source)
	}

	// Reconstruct the volume string.
	parts[0] = source
	return strings.Join(parts, ":")
}