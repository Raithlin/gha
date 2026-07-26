package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// reviewCmd represents the review command
var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Review pull requests",
	Long:  `Assist with reviewing pull requests by providing context, metrics, and suggestions.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("review called")
	},
}

func init() {
	rootCmd.AddCommand(reviewCmd)
}