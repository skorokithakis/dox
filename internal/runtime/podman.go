package runtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/stavros/dox/internal/config"
	"github.com/stavros/dox/internal/utils"
)

// PodmanRuntime implements the Runtime interface for Podman.
type PodmanRuntime struct{}

// NewPodmanRuntime creates a new Podman runtime.
func NewPodmanRuntime() (*PodmanRuntime, error) {
	return &PodmanRuntime{}, nil
}

// IsAvailable checks if Podman is available.
func (r *PodmanRuntime) IsAvailable(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "podman", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Podman not available. Is Podman installed?")
	}
	return nil
}

// ExecuteCommand runs a command in a Podman container.
func (r *PodmanRuntime) ExecuteCommand(ctx context.Context, cfg *config.CommandConfig, command string, args []string, upgrade bool, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	// Build image if inline Dockerfile is provided.
	if cfg.Build != nil && cfg.Build.DockerfileInline != "" {
		imageName := fmt.Sprintf("dox-%s:latest", command)
		
		if upgrade {
			// Remove existing image to force rebuild.
			logrus.Infof("Removing existing image %s for rebuild...", imageName)
			removeCmd := exec.CommandContext(ctx, "podman", "rmi", imageName)
			if err := removeCmd.Run(); err != nil {
				// Image might not exist, which is fine.
				logrus.Debugf("Image removal failed (might not exist): %v", err)
			}
		}
		
		if err := r.BuildImage(ctx, cfg.Build.DockerfileInline, imageName); err != nil {
			return 1, err
		}
		cfg.Image = imageName
	} else if upgrade {
		// Force pull if upgrade flag is set and it's not a locally built image.
		logrus.Infof("Pulling latest version of image %s...", cfg.Image)
		if err := r.PullImage(ctx, cfg.Image); err != nil {
			logrus.Warnf("Failed to pull latest image: %v. Using existing image if available.", err)
		}
	}

	// Prepare Podman run arguments.
	podmanArgs := []string{"run", "--rm", "-it"}

	// Network mode.
	podmanArgs = append(podmanArgs, "--network=host")

	// User mapping.
	uid := os.Getuid()
	gid := os.Getgid()
	podmanArgs = append(podmanArgs, fmt.Sprintf("--user=%d:%d", uid, gid))

	// Working directory.
	podmanArgs = append(podmanArgs, "-w", "/workspace")

	// Volume mounts - always mount current directory.
	cwd, _ := os.Getwd()
	podmanArgs = append(podmanArgs, "-v", fmt.Sprintf("%s:/workspace", cwd))
	for _, volume := range cfg.Volumes {
		podmanArgs = append(podmanArgs, "-v", volume)
	}

	// Environment variables.
	for _, envVar := range cfg.Environment {
		if value := os.Getenv(envVar); value != "" {
			podmanArgs = append(podmanArgs, "-e", fmt.Sprintf("%s=%s", envVar, value))
		}
	}

	// Image.
	podmanArgs = append(podmanArgs, cfg.Image)

	// Command and arguments.
	if cfg.Command != "" {
		podmanArgs = append(podmanArgs, cfg.Command)
		podmanArgs = append(podmanArgs, args...)
	} else if len(args) > 0 {
		// Only pass args if provided, let container use its default ENTRYPOINT/CMD.
		podmanArgs = append(podmanArgs, args...)
	}

	// Check if we should setup terminal raw mode.
	isInteractive := false
	for _, arg := range podmanArgs {
		if arg == "-it" {
			isInteractive = true
			break
		}
	}

	// Setup terminal raw mode for interactive containers.
	if isInteractive {
		oldTermState, _ := utils.SetupTerminal()
		defer utils.RestoreTerminal(oldTermState)
	}

	// Execute Podman.
	cmd := exec.CommandContext(ctx, "podman", podmanArgs...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// Run the command.
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, fmt.Errorf("failed to run podman: %w", err)
	}

	return 0, nil
}

// PullImage pulls a Podman image.
func (r *PodmanRuntime) PullImage(ctx context.Context, image string) error {
	cmd := exec.CommandContext(ctx, "podman", "pull", image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w\nOutput: %s", image, err, output)
	}
	return nil
}

// BuildImage builds a Podman image from inline Dockerfile.
func (r *PodmanRuntime) BuildImage(ctx context.Context, dockerfileContent string, tag string) error {
	// Create a temporary Dockerfile.
	tmpfile, err := os.CreateTemp("", "Dockerfile")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(dockerfileContent)); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}
	if err := tmpfile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Build the image.
	cmd := exec.CommandContext(ctx, "podman", "build", "-t", tag, "-f", tmpfile.Name(), ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to build image: %w\nOutput: %s", err, output)
	}

	return nil
}

// ListImages lists Podman images.
func (r *PodmanRuntime) ListImages(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "podman", "images", "--format", "{{.Repository}}:{{.Tag}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var images []string
	for _, line := range lines {
		if line != "" && !strings.Contains(line, "<none>") {
			images = append(images, line)
		}
	}

	return images, nil
}

// RemoveUnusedContainers removes stopped containers.
func (r *PodmanRuntime) RemoveUnusedContainers(ctx context.Context) error {
	// List exited containers.
	cmd := exec.CommandContext(ctx, "podman", "ps", "-aq", "--filter", "status=exited")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	containerIDs := strings.Fields(string(output))
	if len(containerIDs) == 0 {
		return nil
	}

	// Remove each container.
	for _, id := range containerIDs {
		cmd := exec.CommandContext(ctx, "podman", "rm", id)
		if err := cmd.Run(); err != nil {
			logrus.Warnf("Failed to remove container %s: %v", id, err)
		}
	}

	return nil
}

// RemoveImage removes a specific Podman image.
func (r *PodmanRuntime) RemoveImage(ctx context.Context, image string) error {
	cmd := exec.CommandContext(ctx, "podman", "rmi", image)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove image %s: %w", image, err)
	}
	return nil
}