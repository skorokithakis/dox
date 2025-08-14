package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newVersionCommand creates the version command.
func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show dox version",
		Long:  "Display the version of dox",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("dox version %s\n", version)
			return nil
		},
	}
}