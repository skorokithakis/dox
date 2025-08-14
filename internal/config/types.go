package config

// GlobalConfig represents the global dox configuration.
type GlobalConfig struct {
	Runtime string `mapstructure:"runtime" yaml:"runtime"` // docker or podman
}

// CommandConfig represents configuration for a specific command.
type CommandConfig struct {
	Image       string            `mapstructure:"image" yaml:"image"`             // Container image to use
	Build       *BuildConfig      `mapstructure:"build" yaml:"build"`             // Optional build configuration
	Volumes     []string          `mapstructure:"volumes" yaml:"volumes"`         // Volume mounts
	Environment []string          `mapstructure:"environment" yaml:"environment"` // Environment variables to pass through
	Command     string            `mapstructure:"command" yaml:"command"`         // Optional command override
}

// BuildConfig represents inline Dockerfile build configuration.
type BuildConfig struct {
	DockerfileInline string `mapstructure:"dockerfile_inline" yaml:"dockerfile_inline"` // Inline Dockerfile content
}

// Config represents the complete configuration.
type Config struct {
	Global  GlobalConfig
	Command CommandConfig
}