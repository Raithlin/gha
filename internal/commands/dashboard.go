package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// dashboardCmd represents the dashboard command
var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Launch the TUI dashboard",
	Long:  `Launch the TUI dashboard for monitoring GitHub activity and engineering metrics.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("dashboard called")
	},
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
}