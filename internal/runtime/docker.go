package runtime

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/sirupsen/logrus"
	"github.com/skorokithakis/dox/internal/config"
	"github.com/skorokithakis/dox/internal/utils"
)

// DockerRuntime implements the Runtime interface for Docker.
type DockerRuntime struct {
	client *client.Client
}

// NewDockerRuntime creates a new Docker runtime.
func NewDockerRuntime() (*DockerRuntime, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	
	return &DockerRuntime{
		client: cli,
	}, nil
}

// IsAvailable checks if Docker daemon is available.
func (r *DockerRuntime) IsAvailable(ctx context.Context) error {
	_, err := r.client.Ping(ctx)
	if err != nil {
		return fmt.Errorf("Docker daemon not responding. Is Docker running?")
	}
	return nil
}

// ExecuteCommand runs a command in a Docker container.
func (r *DockerRuntime) ExecuteCommand(ctx context.Context, cfg *config.CommandConfig, command string, args []string, upgrade bool, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	// Build image if inline Dockerfile is provided.
	if cfg.Build != nil && cfg.Build.DockerfileInline != "" {
		imageName := fmt.Sprintf("dox-%s:latest", command)
		
		// Check if image already exists.
		_, _, err := r.client.ImageInspectWithRaw(ctx, imageName)
		imageExists := err == nil
		
		if upgrade && imageExists {
			// Remove the existing image to force rebuild.
			logrus.Infof("Removing existing image %s for rebuild...", imageName)
			if err := r.RemoveImage(ctx, imageName); err != nil {
				logrus.Warnf("Failed to remove existing image: %v", err)
			}
			imageExists = false
		}
		
		if !imageExists {
			// Image doesn't exist or was removed, build it.
			logrus.Infof("Building image %s from inline Dockerfile...", imageName)
			if err := r.BuildImage(ctx, cfg.Build.DockerfileInline, imageName); err != nil {
				return 1, err
			}
		}
		cfg.Image = imageName
	}

	// Prepare command.
	var cmd []string
	if cfg.Command != "" {
		cmd = append([]string{cfg.Command}, args...)
	} else if len(args) > 0 {
		// Only pass args if provided, let container use its default ENTRYPOINT/CMD.
		cmd = args
	}

	// Get current user UID and GID.
	uid := os.Getuid()
	gid := os.Getgid()
	user := fmt.Sprintf("%d:%d", uid, gid)

	// Prepare environment variables.
	var env []string
	for _, envVar := range cfg.Environment {
		if value := os.Getenv(envVar); value != "" {
			env = append(env, fmt.Sprintf("%s=%s", envVar, value))
		}
	}

	// Prepare volume mounts - always mount current directory.
	cwd, _ := os.Getwd()
	volumes := []string{fmt.Sprintf("%s:/workspace", cwd)}
	volumes = append(volumes, cfg.Volumes...)

	// Create container.
	containerConfig := &container.Config{
		Image:        cfg.Image,
		Cmd:          cmd,
		Env:          env,
		User:         user,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		OpenStdin:    true,
		Tty:          isTerminal(),
	}

	// Only set working directory if no inline Dockerfile is provided.
	// When using inline Dockerfile, let the WORKDIR instruction in the Dockerfile take precedence.
	if cfg.Build == nil || cfg.Build.DockerfileInline == "" {
		containerConfig.WorkingDir = "/workspace"
	}

	hostConfig := &container.HostConfig{
		AutoRemove:  true,
		Binds:       volumes,
	}

	// Set network mode if specified.
	if cfg.Network != "" {
		hostConfig.NetworkMode = container.NetworkMode(cfg.Network)
	}

	// Configure port bindings if specified.
	if len(cfg.Ports) > 0 && cfg.Network != "host" {
		portBindings, exposedPorts, err := parsePortMappings(cfg.Ports)
		if err != nil {
			return 1, fmt.Errorf("failed to parse port mappings: %w", err)
		}
		hostConfig.PortBindings = portBindings
		containerConfig.ExposedPorts = exposedPorts
	}

	// Set terminal size if TTY is enabled
	if containerConfig.Tty {
		width, height := utils.GetTerminalSize()
		hostConfig.ConsoleSize = [2]uint{uint(height), uint(width)}
	}

	networkConfig := &network.NetworkingConfig{}

	// Force pull if upgrade flag is set and it's not a locally built image.
	if upgrade && (cfg.Build == nil || cfg.Build.DockerfileInline == "") {
		logrus.Infof("Pulling latest version of image %s...", cfg.Image)
		if pullErr := r.PullImage(ctx, cfg.Image); pullErr != nil {
			logrus.Warnf("Failed to pull latest image: %v. Using existing image if available.", pullErr)
		}
	}
	
	resp, err := r.client.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, "")
	if err != nil {
		// Try to pull the image if it doesn't exist locally.
		if strings.Contains(err.Error(), "No such image") {
			logrus.Infof("Pulling image %s...", cfg.Image)
			if pullErr := r.PullImage(ctx, cfg.Image); pullErr != nil {
				return 1, fmt.Errorf("failed to pull image: %w", pullErr)
			}
			// Retry container creation.
			resp, err = r.client.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, "")
			if err != nil {
				return 1, fmt.Errorf("failed to create container: %w", err)
			}
		} else {
			return 1, fmt.Errorf("failed to create container: %w", err)
		}
	}

	// Attach to container.
	attachOptions := types.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	}

	hijackedResp, err := r.client.ContainerAttach(ctx, resp.ID, attachOptions)
	if err != nil {
		return 1, fmt.Errorf("failed to attach to container: %w", err)
	}
	defer hijackedResp.Close()

	// Start container.
	if err := r.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return 1, fmt.Errorf("failed to start container: %w", err)
	}

	// Setup terminal raw mode for TTY.
	if containerConfig.Tty {
		oldTermState, _ := utils.SetupTerminal()
		defer utils.RestoreTerminal(oldTermState)
	}

	// Setup signal forwarding.
	utils.SetupSignalHandler(ctx, r.client, resp.ID)
	defer utils.CleanupSignalHandler()

	// Handle I/O.
	errChan := make(chan error, 2)

	// Copy stdin to container.
	go func() {
		defer hijackedResp.CloseWrite()
		if stdin != nil {
			_, err := io.Copy(hijackedResp.Conn, stdin)
			errChan <- err
		} else {
			errChan <- nil
		}
	}()

	// Copy container output to stdout/stderr.
	go func() {
		if containerConfig.Tty {
			_, err := io.Copy(stdout, hijackedResp.Reader)
			errChan <- err
		} else {
			_, err := stdcopy.StdCopy(stdout, stderr, hijackedResp.Reader)
			errChan <- err
		}
	}()

	// Wait for container to exit.
	statusCh, errCh := r.client.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		if err != nil {
			return 1, fmt.Errorf("error waiting for container: %w", err)
		}
	case status := <-statusCh:
		return int(status.StatusCode), nil
	case <-ctx.Done():
		return 1, ctx.Err()
	}

	return 0, nil
}

// PullImage pulls a Docker image.
func (r *DockerRuntime) PullImage(ctx context.Context, imageName string) error {
	reader, err := r.client.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Stream the pull output to stdout while reading it.
	decoder := json.NewDecoder(reader)
	for {
		var msg map[string]interface{}
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode pull output: %w", err)
		}

		// Display pull progress.
		if status, ok := msg["status"].(string); ok {
			if progress, ok := msg["progress"].(string); ok && progress != "" {
				fmt.Fprintf(os.Stdout, "%s: %s\r", status, progress)
			} else if id, ok := msg["id"].(string); ok && id != "" {
				fmt.Fprintf(os.Stdout, "%s: %s\n", id, status)
			} else {
				fmt.Fprintln(os.Stdout, status)
			}
		}

		// Check for errors.
		if errorMsg, ok := msg["error"].(string); ok && errorMsg != "" {
			return fmt.Errorf("pull error: %s", errorMsg)
		}
	}
	return nil
}

// BuildImage builds a Docker image from inline Dockerfile.
func (r *DockerRuntime) BuildImage(ctx context.Context, dockerfileContent string, tag string) error {
	// Create a tar archive with the Dockerfile.
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	header := &tar.Header{
		Name: "Dockerfile",
		Mode: 0644,
		Size: int64(len(dockerfileContent)),
	}

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header: %w", err)
	}

	if _, err := tw.Write([]byte(dockerfileContent)); err != nil {
		return fmt.Errorf("failed to write Dockerfile to tar: %w", err)
	}

	if err := tw.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	// Build the image.
	buildOptions := types.ImageBuildOptions{
		Tags:       []string{tag},
		Dockerfile: "Dockerfile",
		Remove:     true,
	}

	buildResp, err := r.client.ImageBuild(ctx, bytes.NewReader(buf.Bytes()), buildOptions)
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}
	defer buildResp.Body.Close()

	// Stream build output to stdout while checking for errors.
	decoder := json.NewDecoder(buildResp.Body)
	for {
		var msg map[string]interface{}
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode build output: %w", err)
		}

		// Display build output stream.
		if stream, ok := msg["stream"].(string); ok && stream != "" {
			fmt.Fprint(os.Stdout, stream)
		}

		// Display aux messages (like image IDs).
		if aux, ok := msg["aux"].(map[string]interface{}); ok {
			if id, ok := aux["ID"].(string); ok {
				fmt.Fprintf(os.Stdout, "Successfully built %s\n", id)
			}
		}

		// Check for errors in the build output.
		if errorDetail, ok := msg["errorDetail"].(map[string]interface{}); ok {
			if errorMsg, ok := errorDetail["message"].(string); ok {
				return fmt.Errorf("build error: %s", errorMsg)
			}
		}
		if errorMsg, ok := msg["error"].(string); ok && errorMsg != "" {
			return fmt.Errorf("build error: %s", errorMsg)
		}
	}
	return nil
}

// ListImages lists Docker images.
func (r *DockerRuntime) ListImages(ctx context.Context) ([]string, error) {
	images, err := r.client.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	var imageNames []string
	for _, img := range images {
		imageNames = append(imageNames, img.RepoTags...)
	}
	return imageNames, nil
}

// RemoveUnusedContainers removes stopped containers.
func (r *DockerRuntime) RemoveUnusedContainers(ctx context.Context) error {
	// List all stopped containers.
	filterArgs := filters.NewArgs()
	filterArgs.Add("status", "exited")
	
	containers, err := r.client.ContainerList(ctx, types.ContainerListOptions{
		Filters: filterArgs,
		All:     true,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Remove each stopped container.
	for _, cnt := range containers {
		if err := r.client.ContainerRemove(ctx, cnt.ID, types.ContainerRemoveOptions{}); err != nil {
			logrus.Warnf("Failed to remove container %s: %v", cnt.ID[:12], err)
		}
	}

	return nil
}

// RemoveImage removes a specific Docker image.
func (r *DockerRuntime) RemoveImage(ctx context.Context, image string) error {
	_, err := r.client.ImageRemove(ctx, image, types.ImageRemoveOptions{
		Force: false,
		PruneChildren: true,
	})
	if err != nil {
		return fmt.Errorf("failed to remove image %s: %w", image, err)
	}
	return nil
}

// isTerminal checks if both stdin and stdout are terminals.
func isTerminal() bool {
	stdinInfo, _ := os.Stdin.Stat()
	stdoutInfo, _ := os.Stdout.Stat()
	// Both stdin and stdout should be terminals for interactive mode.
	return (stdinInfo.Mode()&os.ModeCharDevice) != 0 &&
		(stdoutInfo.Mode()&os.ModeCharDevice) != 0
}

// parsePortMappings parses port mapping strings and returns Docker port bindings.
func parsePortMappings(ports []string) (nat.PortMap, nat.PortSet, error) {
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}

	for _, portMapping := range ports {
		// Parse the port mapping string.
		// Formats: "8080:80", "127.0.0.1:8080:80", "8080:80/tcp", "8080-8090:8080-8090"
		mapping, err := nat.ParsePortSpec(portMapping)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid port mapping %s: %w", portMapping, err)
		}

		for _, m := range mapping {
			exposedPorts[m.Port] = struct{}{}
			portBindings[m.Port] = []nat.PortBinding{m.Binding}
		}
	}

	return portBindings, exposedPorts, nil
}