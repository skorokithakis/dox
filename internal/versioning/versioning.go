package versioning

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// CommandVersion represents version information for a command.
type CommandVersion struct {
	Hash        string    `json:"hash"`
	LastUpdated time.Time `json:"last_updated"`
}

// VersionStore manages command versions.
type VersionStore struct {
	configHome string
	versions   map[string]CommandVersion
}

// NewVersionStore creates a new version store.
func NewVersionStore() *VersionStore {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, _ := os.UserHomeDir()
		configHome = filepath.Join(home, ".config")
	}
	
	store := &VersionStore{
		configHome: configHome,
		versions:   make(map[string]CommandVersion),
	}
	
	// Load existing versions.
	_ = store.load()
	
	return store
}

// CalculateFileHash computes SHA-256 hash of a file.
func CalculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}
	
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// GetCommandYAMLPath returns the path to a command's YAML file.
func (v *VersionStore) GetCommandYAMLPath(command string) string {
	return filepath.Join(v.configHome, "dox", "commands", command+".yaml")
}

// HasCommandChanged checks if a command's YAML file has changed since last stored version.
func (v *VersionStore) HasCommandChanged(command string) (bool, error) {
	yamlPath := v.GetCommandYAMLPath(command)
	
	currentHash, err := CalculateFileHash(yamlPath)
	if err != nil {
		return false, fmt.Errorf("failed to calculate hash for %s: %w", command, err)
	}
	
	storedVersion, exists := v.versions[command]
	if !exists {
		// No stored version means this is the first time running the command.
		return true, nil
	}
	
	return storedVersion.Hash != currentHash, nil
}

// UpdateCommandVersion updates the stored hash for a command.
func (v *VersionStore) UpdateCommandVersion(command string) error {
	yamlPath := v.GetCommandYAMLPath(command)
	
	hash, err := CalculateFileHash(yamlPath)
	if err != nil {
		return fmt.Errorf("failed to calculate hash for %s: %w", command, err)
	}
	
	v.versions[command] = CommandVersion{
		Hash:        hash,
		LastUpdated: time.Now(),
	}
	
	return v.save()
}

// GetVersionFilePath returns the path to the versions file.
func (v *VersionStore) GetVersionFilePath() string {
	return filepath.Join(v.configHome, "dox", "command_versions.json")
}

// load reads the versions file from disk.
func (v *VersionStore) load() error {
	filePath := v.GetVersionFilePath()
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, which is fine.
			return nil
		}
		return fmt.Errorf("failed to read versions file: %w", err)
	}
	
	if err := json.Unmarshal(data, &v.versions); err != nil {
		return fmt.Errorf("failed to unmarshal versions: %w", err)
	}
	
	return nil
}

// save writes the versions to disk.
func (v *VersionStore) save() error {
	filePath := v.GetVersionFilePath()
	
	// Ensure the directory exists.
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	data, err := json.MarshalIndent(v.versions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal versions: %w", err)
	}
	
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write versions file: %w", err)
	}
	
	return nil
}

// RemoveCommandVersion removes the stored version for a command.
func (v *VersionStore) RemoveCommandVersion(command string) error {
	delete(v.versions, command)
	return v.save()
}