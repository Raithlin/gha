package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/internal/review"
)

type releasesOptions struct {
	repository string
	path       string
	format     string
	limit      int
}

// newReleasesCmd constructs the bounded published-release discovery command.
func newReleasesCmd(service *review.Service, resolver *git.RepositoryResolver) *cobra.Command {
	options := &releasesOptions{}
	command := &cobra.Command{
		Use:   "releases",
		Short: "List published releases",
		Long: `List published releases in a GitHub repository with a stable, bounded result.

Draft releases are excluded. The repository is taken from --repo, the origin
remote in --path, GHA_REPOSITORY, or the current directory's origin remote (in
that order). Use direct "gh release" commands when you need an unbounded or
provider-specific release operation.`,
		Args: noArgsWithFormat(&options.format),
		RunE: newReleasesRunE(service, resolver, options),
	}
	command.Flags().StringVarP(&options.repository, "repo", "r", "", "Repository to inspect (owner/repo)")
	command.Flags().StringVar(&options.path, "path", "", "Local Git checkout whose origin selects the repository")
	command.Flags().StringVarP(&options.format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().IntVarP(&options.limit, "limit", "l", 30, "Maximum published releases to return (1-100)")
	command.SilenceUsage = true
	command.SilenceErrors = true
	return command
}

func newReleasesRunE(service *review.Service, resolver *git.RepositoryResolver, options *releasesOptions) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		outputFormat, err := output.ParseFormat(options.format)
		if err != nil {
			return err
		}
		if options.limit < 1 || options.limit > 100 {
			return renderCommandError(cmd, outputFormat, "invalid_argument", fmt.Errorf("limit must be between 1 and 100"))
		}
		target, err := resolver.ResolveAtPath(cmd.Context(), options.repository, options.path)
		if err != nil {
			return renderCommandError(cmd, outputFormat, "repository_resolution_failed", err)
		}
		releases, err := service.ListPublishedReleases(cmd.Context(), target, options.limit)
		if err != nil {
			return renderCommandError(cmd, outputFormat, "release_list_failed", err)
		}
		return output.ReleaseList(cmd.OutOrStdout(), outputFormat, releases)
	}
}
