package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newPRsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prs",
		Short: "List and manage pull requests",
		Long:  `List pull requests, check review status, and manage PR workflow.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("prs called")
		},
	}
}
