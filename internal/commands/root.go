package commands

import (
	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/branch"
	"github.com/raithlin/gha/internal/buildinfo"
	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/release"
	"github.com/raithlin/gha/internal/review"
)

const rootHelpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if .Version}}Version: {{.Version}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

// NewRootCmd constructs the application command tree from explicit dependencies.
func NewRootCmd(branchService *branch.Service, reviewService *review.Service, resolver *git.RepositoryResolver, releaseServices ...*release.Service) *cobra.Command {
	build := buildinfo.Current()
	root := &cobra.Command{
		Use:           "gha",
		Short:         "GitHub Assistant - A developer productivity tool",
		Version:       build.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Long: `GHA is a developer productivity tool written in Go.
It helps software developers make better engineering decisions by combining
information from GitHub, Git, CI systems, issue trackers, and local repositories
into a single cohesive experience.`,
	}
	root.SetHelpTemplate(rootHelpTemplate)
	root.AddCommand(newAgentCmd(), newUpdateCmd(), newCapabilitiesCmd(), newDashboardCmd(), newVersionCmd(build), newAnalyzeCmd(nil), newBranchesCmd(branchService), newBranchCmd(branchService, resolver), newTagCmd(), newPRCmd(reviewService, resolver), newPRsCmd(reviewService, resolver), newReleasesCmd(reviewService, resolver), newReleaseCmd(reviewService, resolver, releaseServices...), newReviewCmd(reviewService, resolver))
	return root
}

// Execute runs a freshly constructed command tree.
func Execute(branchService *branch.Service, reviewService *review.Service, resolver *git.RepositoryResolver, releaseServices ...*release.Service) error {
	return NewRootCmd(branchService, reviewService, resolver, releaseServices...).Execute()
}
