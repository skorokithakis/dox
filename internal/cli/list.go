package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/stavros/dox/internal/config"
)

// newListCommand creates the list command.
func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available commands",
		Long:  "List all commands configured in the dox commands directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			loader := config.NewLoader()
			commands, err := loader.ListCommands()
			if err != nil {
				return fmt.Errorf("failed to list commands: %w", err)
			}

			if len(commands) == 0 {
				fmt.Println("No commands configured.")
				fmt.Println("\nTo add a command, create a YAML file in:")
				fmt.Println("  ${XDG_CONFIG_HOME}/dox/commands/<command>.yaml")
				return nil
			}

			fmt.Println("Available commands:")
			sort.Strings(commands)
			for _, command := range commands {
				fmt.Printf("  %s\n", command)
			}

			return nil
		},
	}
}