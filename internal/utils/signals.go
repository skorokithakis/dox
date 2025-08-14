package utils

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

// SetupSignalHandler sets up signal forwarding to a container.
func SetupSignalHandler(ctx context.Context, dockerClient *client.Client, containerID string) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)

	go func() {
		for sig := range sigChan {
			// Forward the signal to the container.
			if err := dockerClient.ContainerKill(ctx, containerID, sig.String()); err != nil {
				// Container might have already exited.
				if !client.IsErrNotFound(err) {
					logrus.Debugf("Failed to forward signal %s: %v", sig, err)
				}
			}
		}
	}()
}

// CleanupSignalHandler stops signal handling.
func CleanupSignalHandler() {
	signal.Stop(make(chan os.Signal, 1))
}