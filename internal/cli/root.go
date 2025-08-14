package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/stavros/dox/internal/config"
	"github.com/stavros/dox/internal/runtime"
)

var (
	version = "1.0.0"
)

// Execute runs the CLI.
func Execute() {
	// Check if first argument is a containerized command before Cobra parses it.
	if len(os.Args) > 1 {
		command := os.Args[1]
		// Check if it's not a built-in command.
		if command != "list" && command != "version" && command != "upgrade" && command != "upgrade-all" && command != "clean" && command != "--help" && command != "-h" && command != "help" {
			// Try to load it as a containerized command.
			loader := config.NewLoader()
			if _, err := loader.LoadCommandConfig(command); err == nil {
				// It's a valid containerized command, run it directly.
				if err := runContainerizedCommand(command, os.Args[2:]); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}
				return
			}
		}
	}

	rootCmd := &cobra.Command{
		Use:   "dox [command] [arguments...]",
		Short: "Execute commands in Docker containers",
		Long: `Dox is a lightweight wrapper that transparently executes commands within Docker or Podman containers
while maintaining the user experience of native host commands.`,
		RunE:         runCommand,
		SilenceUsage: true,
	}

	// Disable default completion command.
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Add subcommands.
	rootCmd.AddCommand(
		newListCommand(),
		newVersionCommand(),
		newUpgradeCommand(),
		newUpgradeAllCommand(),
		newCleanCommand(),
	)

	// Handle unknown commands as containerized commands.
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return err
	})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// runContainerizedCommand handles execution of a containerized command.
func runContainerizedCommand(command string, commandArgs []string) error {
	// Load configuration.
	loader := config.NewLoader()
	
	globalConfig, err := loader.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	commandConfig, err := loader.LoadCommandConfig(command)
	if err != nil {
		return err
	}

	// Create runtime based on configuration.
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

	// Execute the command in container.
	exitCode, err := rt.ExecuteCommand(ctx, commandConfig, command, commandArgs, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		logrus.Errorf("Command execution failed: %v", err)
		os.Exit(1)
	}

	os.Exit(exitCode)
	return nil
}

// runCommand handles execution of containerized commands.
func runCommand(cmd *cobra.Command, args []string) error {
	// If no args, show help.
	if len(args) == 0 {
		return cmd.Help()
	}

	// First argument is the command to run.
	command := args[0]
	commandArgs := args[1:]

	// Check if it's a built-in command that wasn't caught.
	switch command {
	case "list", "version", "upgrade", "upgrade-all", "clean":
		// These should have been caught by subcommands.
		return cmd.Help()
	}

	// Run the containerized command.
	return runContainerizedCommand(command, commandArgs)
}