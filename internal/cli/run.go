package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/skorokithakis/dox/internal/config"
	"github.com/skorokithakis/dox/internal/runtime"
)

// newRunCommand creates the run command.
func newRunCommand() *cobra.Command {
	var upgrade bool
	
	cmd := &cobra.Command{
		Use:   "run [command] [arguments...]",
		Short: "Run a containerized command",
		Long: `Run a command in a Docker or Podman container.

The command must have a configuration file in ~/.config/dox/commands/<command>.yaml`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, upgrade)
		},
	}
	
	// Add upgrade flag.
	cmd.Flags().BoolVar(&upgrade, "upgrade", false, "Force pull/rebuild the container image")
	
	// Disable flag parsing after the first argument to pass all flags to the containerized command.
	cmd.TraverseChildren = false
	cmd.FParseErrWhitelist.UnknownFlags = true
	
	return cmd
}

// runCommand handles execution of containerized commands.
func runCommand(cmd *cobra.Command, args []string, upgrade bool) error {
	// First argument is the command to run.
	command := args[0]
	commandArgs := args[1:]

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
	exitCode, err := rt.ExecuteCommand(ctx, commandConfig, command, commandArgs, upgrade, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		logrus.Errorf("Command execution failed: %v", err)
		os.Exit(1)
	}

	os.Exit(exitCode)
	return nil
}