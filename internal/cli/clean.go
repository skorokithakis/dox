package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/skorokithakis/dox/internal/config"
	"github.com/skorokithakis/dox/internal/runtime"
)

// newCleanCommand creates the clean command.
func newCleanCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clean",
		Short: "Remove unused containers",
		Long:  "Remove all stopped containers to free up resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime.
			loader := config.NewLoader()
			globalConfig, err := loader.LoadGlobalConfig()
			if err != nil {
				return fmt.Errorf("failed to load global config: %w", err)
			}

			var rt runtime.Runtime
			switch globalConfig.Runtime {
			case "podman":
				podmanRuntime, err := runtime.NewPodmanRuntime()
				if err != nil {
					return err
				}
				rt = podmanRuntime
			default:
				dockerRuntime, err := runtime.NewDockerRuntime()
				if err != nil {
					return err
				}
				rt = dockerRuntime
			}

			// Check if runtime is available.
			ctx := context.Background()
			if err := rt.IsAvailable(ctx); err != nil {
				return err
			}

			// Remove unused containers.
			fmt.Println("Removing unused containers...")
			if err := rt.RemoveUnusedContainers(ctx); err != nil {
				return fmt.Errorf("failed to remove containers: %w", err)
			}

			fmt.Println("Cleanup complete.")
			return nil
		},
	}
}