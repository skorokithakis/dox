package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"
)

// Execute runs the CLI.
func Execute() {

	rootCmd := &cobra.Command{
		Use:   "dox",
		Short: "Execute commands in Docker containers",
		Long: `Dox is a lightweight wrapper that transparently executes commands within Docker or Podman containers
while maintaining the user experience of native host commands.`,
		SilenceUsage: true,
	}

	// Disable default completion command.
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Add subcommands.
	rootCmd.AddCommand(
		newRunCommand(),
		newListCommand(),
		newVersionCommand(),
		newUpgradeCommand(),
		newUpgradeAllCommand(),
		newCleanCommand(),
	)


	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

