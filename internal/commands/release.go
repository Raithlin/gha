package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newReleaseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "release",
		Short: "Manage releases",
		Long:  `Generate release notes, changelogs, and manage the release process.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("release called")
		},
	}
}
