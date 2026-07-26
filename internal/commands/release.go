package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// releaseCmd represents the release command
var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Manage releases",
	Long:  `Generate release notes, changelogs, and manage the release process.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("release called")
	},
}

func init() {
	rootCmd.AddCommand(releaseCmd)
}