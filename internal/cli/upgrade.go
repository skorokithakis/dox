package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/skorokithakis/dox/internal/config"
	"github.com/skorokithakis/dox/internal/runtime"
)

// newUpgradeCommand creates the upgrade command.
func newUpgradeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade <command>",
		Short: "Upgrade a specific command's image",
		Long:  "Pull the latest version of the image for a specific command",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			command := args[0]
			
			// Load command configuration.
			loader := config.NewLoader()
			commandConfig, err := loader.LoadCommandConfig(command)
			if err != nil {
				return err
			}

			// Get runtime first (needed for inline Dockerfile handling).
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

			// Handle inline Dockerfile - remove the existing image to force rebuild.
			if commandConfig.Build != nil && commandConfig.Build.DockerfileInline != "" {
				imageName := fmt.Sprintf("dox-%s:latest", command)
				fmt.Printf("Command '%s' uses inline Dockerfile. Removing existing image to force rebuild...\n", command)
				
				// Try to remove the image. Ignore errors if image doesn't exist.
				if err := rt.RemoveImage(ctx, imageName); err != nil {
					// Only log if it's not a "not found" error.
					if !strings.Contains(err.Error(), "No such image") && !strings.Contains(err.Error(), "not found") {
						fmt.Printf("Warning: could not remove image %s: %v\n", imageName, err)
					}
				} else {
					fmt.Printf("Successfully removed image %s. It will be rebuilt on next run.\n", imageName)
				}
				return nil
			}

			// Skip if image is SHA-pinned.
			if strings.Contains(commandConfig.Image, "@sha256:") {
				fmt.Printf("Command '%s' uses SHA-pinned image. Skipping upgrade.\n", command)
				return nil
			}

			// Pull the latest image.
			fmt.Printf("Upgrading image for command '%s': %s\n", command, commandConfig.Image)
			if err := rt.PullImage(ctx, commandConfig.Image); err != nil {
				return fmt.Errorf("failed to pull image: %w", err)
			}

			fmt.Printf("Successfully upgraded '%s'\n", command)
			return nil
		},
	}
}

// newUpgradeAllCommand creates the upgrade-all command.
func newUpgradeAllCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade-all",
		Short: "Upgrade all command images",
		Long:  "Pull the latest version of all non-SHA-pinned images",
		RunE: func(cmd *cobra.Command, args []string) error {
			loader := config.NewLoader()
			
			// List all commands.
			commands, err := loader.ListCommands()
			if err != nil {
				return fmt.Errorf("failed to list commands: %w", err)
			}

			if len(commands) == 0 {
				fmt.Println("No commands to upgrade.")
				return nil
			}

			// Get runtime.
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

			// Upgrade each command.
			upgradedCount := 0
			for _, command := range commands {
				commandConfig, err := loader.LoadCommandConfig(command)
				if err != nil {
					fmt.Printf("Failed to load config for '%s': %v\n", command, err)
					continue
				}

				// Handle inline Dockerfile - remove the existing image to force rebuild.
				if commandConfig.Build != nil && commandConfig.Build.DockerfileInline != "" {
					imageName := fmt.Sprintf("dox-%s:latest", command)
					fmt.Printf("Rebuilding '%s': removing image %s\n", command, imageName)
					
					// Try to remove the image. Ignore errors if image doesn't exist.
					if err := rt.RemoveImage(ctx, imageName); err != nil {
						// Only log if it's not a "not found" error.
						if !strings.Contains(err.Error(), "No such image") && !strings.Contains(err.Error(), "not found") {
							fmt.Printf("Warning: could not remove image %s: %v\n", imageName, err)
						}
					} else {
						upgradedCount++
					}
					continue
				}

				// Skip if SHA-pinned.
				if strings.Contains(commandConfig.Image, "@sha256:") {
					fmt.Printf("Skipping '%s': SHA-pinned image\n", command)
					continue
				}

				fmt.Printf("Upgrading '%s': %s\n", command, commandConfig.Image)
				if err := rt.PullImage(ctx, commandConfig.Image); err != nil {
					fmt.Printf("Failed to upgrade '%s': %v\n", command, err)
					continue
				}

				upgradedCount++
			}

			fmt.Printf("\nUpgraded %d command(s)\n", upgradedCount)
			return nil
		},
	}
}