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
	var assigned, queue, mine bool
	var repository, format string

	command := &cobra.Command{
		Use:   "review [number]",
		Short: "Review pull requests",
		Long: `Review pull requests from GitHub repositories.

The repository is taken from --repo, GHA_REPOSITORY, or the current directory's
origin remote (in that order).

Examples:
  gha review 123                # Review PR #123 in the current repository
  gha review --assigned         # Show PRs assigned to you in the repository
  gha review --queue            # Show PRs awaiting your review
  gha review --mine             # Show your PRs
  gha review --repo owner/repo  # Specify a repository explicitly`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			modeCount := boolCount(assigned, queue, mine)
			if len(args) == 0 && modeCount == 0 {
				return cmd.Help()
			}
			if len(args) > 0 && modeCount > 0 {
				return fmt.Errorf("a pull request number cannot be combined with --assigned, --queue, or --mine")
			}
			if modeCount > 1 {
				return fmt.Errorf("use only one of --assigned, --queue, or --mine")
			}

			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			target, err := resolver.Resolve(cmd.Context(), repository)
			if err != nil {
				return err
			}

			switch {
			case assigned:
				prs, err := service.Assigned(cmd.Context(), target)
				if err != nil {
					return err
				}
				return output.PullRequestList(cmd.OutOrStdout(), outputFormat, prs, "Assigned Pull Requests")
			case queue:
				prs, err := service.Queue(cmd.Context(), target)
				if err != nil {
					return err
				}
				return output.PullRequestList(cmd.OutOrStdout(), outputFormat, prs, "Pull Requests Awaiting Your Review")
			case mine:
				prs, err := service.Mine(cmd.Context(), target)
				if err != nil {
					return err
				}
				return output.PullRequestList(cmd.OutOrStdout(), outputFormat, prs, "Your Pull Requests")
			default:
				number, err := strconv.Atoi(args[0])
				if err != nil || number < 1 {
					return fmt.Errorf("invalid pull request number %q", args[0])
				}
				pr, err := service.Get(cmd.Context(), target, number)
				if err != nil {
					return err
				}
				return output.PullRequest(cmd.OutOrStdout(), outputFormat, pr)
			}
		},
	}

	command.Flags().BoolVarP(&assigned, "assigned", "a", false, "List pull requests assigned to you")
	command.Flags().BoolVarP(&queue, "queue", "q", false, "List pull requests awaiting your review")
	command.Flags().BoolVarP(&mine, "mine", "m", false, "List your pull requests")
	command.Flags().StringVarP(&repository, "repo", "r", "", "Repository to inspect (owner/repo)")
	command.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")
	return command
}

func boolCount(values ...bool) int {
	count := 0
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}
