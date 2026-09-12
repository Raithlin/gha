package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDashboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dashboard",
		Short: "Launch the TUI dashboard",
		Long:  `Launch the TUI dashboard for monitoring GitHub activity and engineering metrics.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("dashboard called")
		},
	}
}
