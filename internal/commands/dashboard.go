package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDashboardCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "dashboard",
		Short: "Reserved for the future TUI dashboard",
		Long: `The interactive dashboard is not implemented in this build.

Use gha capabilities --format json to discover commands that are ready for
automation.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("dashboard is unavailable: the TUI dashboard is not implemented in this build")
		},
	}
	command.SilenceUsage = true
	command.SilenceErrors = true
	return command
}
