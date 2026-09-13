package commands

import (
	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/review"
)

// NewRootCmd constructs the application command tree from explicit dependencies.
func NewRootCmd(reviewService *review.Service, resolver *git.RepositoryResolver) *cobra.Command {
	root := &cobra.Command{
		Use:   "gha",
		Short: "GitHub Assistant - A developer productivity tool",
		Long: `GHA is a developer productivity tool written in Go.
It helps software developers make better engineering decisions by combining
information from GitHub, Git, CI systems, issue trackers, and local repositories
into a single cohesive experience.`,
	}
	root.AddCommand(newDashboardCmd(), newPRsCmd(reviewService, resolver), newReleaseCmd(reviewService, resolver), newReviewCmd(reviewService, resolver))
	return root
}

// Execute runs a freshly constructed command tree.
func Execute(reviewService *review.Service, resolver *git.RepositoryResolver) error {
	return NewRootCmd(reviewService, resolver).Execute()
}
