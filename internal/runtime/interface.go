package runtime

import (
	"context"
	"io"

	"github.com/stavros/dox/internal/config"
)

// Runtime defines the interface for container runtimes.
type Runtime interface {
	// ExecuteCommand runs a command in a container.
	ExecuteCommand(ctx context.Context, config *config.CommandConfig, command string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error)
	
	// PullImage pulls a container image.
	PullImage(ctx context.Context, image string) error
	
	// BuildImage builds an image from inline Dockerfile.
	BuildImage(ctx context.Context, dockerfileContent string, tag string) error
	
	// ListImages lists all images.
	ListImages(ctx context.Context) ([]string, error)
	
	// RemoveUnusedContainers removes stopped containers.
	RemoveUnusedContainers(ctx context.Context) error
	
	// RemoveImage removes a specific image.
	RemoveImage(ctx context.Context, image string) error
	
	// IsAvailable checks if the runtime is available.
	IsAvailable(ctx context.Context) error
}

// ContainerOptions represents options for container execution.
type ContainerOptions struct {
	Image       string
	Command     []string
	Env         []string
	Volumes     []string
	WorkingDir  string
	User        string
	Interactive bool
	TTY         bool
	Remove      bool
	Network     string
}