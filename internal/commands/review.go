package commands

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/internal/review"
)

// newReviewCmd constructs the review command with its explicit dependencies.
func newReviewCmd(service *review.Service, resolver *git.RepositoryResolver) *cobra.Command {
	var repository, path, format string

	command := &cobra.Command{
		Use:   "review <number>",
		Short: "Inspect a pull request for review",
		Long: `Inspect one pull request from a GitHub repository.

The repository is taken from --repo, the origin remote in --path,
GHA_REPOSITORY, or the current directory's origin remote (in that order).

Examples:
  gha review 123                # Inspect PR #123 in the current repository
  gha review 123 --repo owner/repo  # Specify a repository explicitly`,
		Args: exactArgsWithFormat(1, &format),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			number, err := strconv.Atoi(args[0])
			if err != nil || number < 1 {
				return renderCommandError(cmd, outputFormat, "invalid_argument", fmt.Errorf("invalid pull request number %q", args[0]))
			}
			target, err := resolver.ResolveAtPath(cmd.Context(), repository, path)
			if err != nil {
				return renderCommandError(cmd, outputFormat, "repository_resolution_failed", err)
			}
			summary, err := service.Inspect(cmd.Context(), target, number)
			if err != nil {
				return renderCommandError(cmd, outputFormat, "review_inspection_failed", err)
			}
			return output.ReviewSummary(cmd.OutOrStdout(), outputFormat, summary)
		},
	}

	command.Flags().StringVarP(&repository, "repo", "r", "", "Repository to inspect (owner/repo)")
	command.Flags().StringVar(&path, "path", "", "Local Git checkout whose origin selects the repository")
	command.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")
	command.SilenceUsage = true
	command.SilenceErrors = true
	return command
}
