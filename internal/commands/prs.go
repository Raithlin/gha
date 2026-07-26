package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// prsCmd represents the prs command
var prsCmd = &cobra.Command{
	Use:   "prs",
	Short: "List and manage pull requests",
	Long:  `List pull requests, check review status, and manage PR workflow.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("prs called")
	},
}

func init() {
	rootCmd.AddCommand(prsCmd)
}