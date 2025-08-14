package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandVolumePath(t *testing.T) {
	loader := NewLoader()
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "current directory",
			input:    ".:/workspace",
			expected: func() string {
				cwd, _ := os.Getwd()
				return cwd + ":/workspace"
			}(),
		},
		{
			name:     "environment variable",
			input:    "${HOME}/.ssh:/root/.ssh",
			expected: os.Getenv("HOME") + "/.ssh:/root/.ssh",
		},
		{
			name:     "read-only volume",
			input:    "${HOME}/.config:/config:ro",
			expected: os.Getenv("HOME") + "/.config:/config:ro",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := loader.expandVolumePath(tt.input)
			if result != tt.expected {
				t.Errorf("expandVolumePath(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLoadGlobalConfig(t *testing.T) {
	// Create a temporary config directory.
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "dox")
	os.MkdirAll(configDir, 0755)
	
	// Create a test config file.
	configContent := `runtime: podman`
	configPath := filepath.Join(configDir, "config.yaml")
	os.WriteFile(configPath, []byte(configContent), 0644)
	
	// Override XDG_CONFIG_HOME for testing.
	oldConfig := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldConfig)
	
	loader := NewLoader()
	config, err := loader.LoadGlobalConfig()
	
	if err != nil {
		t.Fatalf("LoadGlobalConfig() error = %v", err)
	}
	
	if config.Runtime != "podman" {
		t.Errorf("config.Runtime = %s, want podman", config.Runtime)
	}
}

func TestLoadCommandConfig(t *testing.T) {
	// Create a temporary config directory.
	tmpDir := t.TempDir()
	commandsDir := filepath.Join(tmpDir, "dox", "commands")
	os.MkdirAll(commandsDir, 0755)
	
	// Create a test command config file.
	configContent := `image: python:3.11-slim
volumes:
  - .:/workspace
  - ${HOME}/.cache:/cache
environment:
  - PATH
  - HOME`
	configPath := filepath.Join(commandsDir, "python.yaml")
	os.WriteFile(configPath, []byte(configContent), 0644)
	
	// Override XDG_CONFIG_HOME for testing.
	oldConfig := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldConfig)
	
	loader := NewLoader()
	config, err := loader.LoadCommandConfig("python")
	
	if err != nil {
		t.Fatalf("LoadCommandConfig() error = %v", err)
	}
	
	if config.Image != "python:3.11-slim" {
		t.Errorf("config.Image = %s, want python:3.11-slim", config.Image)
	}
	
	if len(config.Volumes) != 2 {
		t.Errorf("len(config.Volumes) = %d, want 2", len(config.Volumes))
	}
	
	if len(config.Environment) != 2 {
		t.Errorf("len(config.Environment) = %d, want 2", len(config.Environment))
	}
}

func TestLoadCommandConfigMissing(t *testing.T) {
	// Create a temporary config directory.
	tmpDir := t.TempDir()
	commandsDir := filepath.Join(tmpDir, "dox", "commands")
	os.MkdirAll(commandsDir, 0755)
	
	// Override XDG_CONFIG_HOME for testing.
	oldConfig := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldConfig)
	
	loader := NewLoader()
	_, err := loader.LoadCommandConfig("nonexistent")
	
	if err == nil {
		t.Fatal("LoadCommandConfig() should have returned an error for missing command")
	}
	
	// The error could be either "command doesn't exist" or "failed to read command config"
	// depending on how viper detects the missing file.
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error message should mention the nonexistent command, got %q", err.Error())
	}
}

func TestListCommands(t *testing.T) {
	// Create a temporary config directory.
	tmpDir := t.TempDir()
	commandsDir := filepath.Join(tmpDir, "dox", "commands")
	os.MkdirAll(commandsDir, 0755)
	
	// Create test command files.
	commands := []string{"python", "node", "ruby"}
	for _, cmd := range commands {
		configPath := filepath.Join(commandsDir, cmd+".yaml")
		os.WriteFile(configPath, []byte("image: test"), 0644)
	}
	
	// Also create a non-YAML file that should be ignored.
	os.WriteFile(filepath.Join(commandsDir, "README.md"), []byte("readme"), 0644)
	
	// Override XDG_CONFIG_HOME for testing.
	oldConfig := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldConfig)
	
	loader := NewLoader()
	result, err := loader.ListCommands()
	
	if err != nil {
		t.Fatalf("ListCommands() error = %v", err)
	}
	
	if len(result) != 3 {
		t.Errorf("len(result) = %d, want 3", len(result))
	}
	
	// Check that all expected commands are present.
	for _, expected := range commands {
		found := false
		for _, cmd := range result {
			if cmd == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("command %q not found in result", expected)
		}
	}
}

