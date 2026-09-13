package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/interfaces"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/internal/review"
	"github.com/raithlin/gha/pkg/model"
)

// newPRsCmd constructs the pull request listing command with explicit dependencies.
func newPRsCmd(service *review.Service, resolver *git.RepositoryResolver) *cobra.Command {
	var assigned, queue, mine bool
	var repository, format, state, author, reviewer, base, head, sort, direction string
	var limit int

	command := &cobra.Command{
		Use:   "prs",
		Short: "List pull requests",
		Long: `List pull requests in a GitHub repository.

The repository is taken from --repo, GHA_REPOSITORY, or the current directory's
origin remote (in that order). Use gha review <number> to inspect one pull request.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if boolCount(assigned, queue, mine) > 1 {
				return fmt.Errorf("use only one of --assigned, --queue, or --mine")
			}
			if (assigned || queue || mine) && hasListFilters(cmd) {
				return fmt.Errorf("--assigned, --queue, and --mine cannot be combined with list filters")
			}
			if limit < 1 || limit > 100 {
				return fmt.Errorf("limit must be between 1 and 100")
			}
			if !assigned && !queue && !mine {
				if err := validateListOptions(state, sort, direction); err != nil {
					return err
				}
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
				prs, err := service.AssignedLimited(cmd.Context(), target, limit)
				if err != nil {
					return err
				}
				return output.PullRequestList(cmd.OutOrStdout(), outputFormat, limitPullRequests(prs, limit), "Pull Requests Assigned to You")
			case queue:
				prs, err := service.QueueLimited(cmd.Context(), target, limit)
				if err != nil {
					return err
				}
				return output.PullRequestList(cmd.OutOrStdout(), outputFormat, limitPullRequests(prs, limit), "Pull Requests Awaiting Your Review")
			case mine:
				prs, err := service.MineLimited(cmd.Context(), target, limit)
				if err != nil {
					return err
				}
				return output.PullRequestList(cmd.OutOrStdout(), outputFormat, limitPullRequests(prs, limit), "Your Pull Requests")
			default:
				perPage := limit
				if author != "" || reviewer != "" {
					perPage = 100
				}
				author, reviewer, err = resolveListUsers(cmd, service, author, reviewer)
				if err != nil {
					return err
				}
				if head != "" && !strings.Contains(head, ":") {
					user, err := service.AuthenticatedUser(cmd.Context())
					if err != nil {
						return err
					}
					head = user.Login + ":" + head
				}
				prs, err := service.ListMatching(cmd.Context(), target, interfaces.ListPRsOptions{
					State: state, Head: head, Base: base, Sort: sort, Direction: direction, PerPage: perPage,
				}, limit, func(pr *model.PullRequest) bool {
					return matchesPullRequest(pr, author, reviewer)
				})
				if err != nil {
					return err
				}
				return output.PullRequestList(cmd.OutOrStdout(), outputFormat, prs, "Pull Requests")
			}
		},
	}

	command.Flags().BoolVarP(&assigned, "assigned", "a", false, "List pull requests assigned to you")
	command.Flags().BoolVarP(&queue, "queue", "q", false, "List pull requests awaiting your review")
	command.Flags().BoolVarP(&mine, "mine", "m", false, "List pull requests authored by you")
	command.Flags().StringVarP(&repository, "repo", "r", "", "Repository to inspect (owner/repo)")
	command.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().StringVar(&state, "state", "open", "Filter by state (open, closed, all)")
	command.Flags().StringVar(&author, "author", "", "Filter by author login (use @me for yourself)")
	command.Flags().StringVar(&reviewer, "reviewer", "", "Filter by requested reviewer login (use @me for yourself)")
	command.Flags().StringVar(&base, "base", "", "Filter by base branch")
	command.Flags().StringVar(&head, "head", "", "Filter by head branch (bare branch uses your login; owner:branch also accepted)")
	command.Flags().StringVar(&sort, "sort", "", "Sort by created, updated, popularity, or long-running")
	command.Flags().StringVar(&direction, "direction", "", "Sort direction (asc or desc)")
	command.Flags().IntVarP(&limit, "limit", "l", 30, "Maximum pull requests to return (1-100)")
	return command
}

func hasListFilters(cmd *cobra.Command) bool {
	for _, name := range []string{"state", "author", "reviewer", "base", "head", "sort", "direction"} {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}

func validateListOptions(state, sort, direction string) error {
	if !oneOf(state, "open", "closed", "all") {
		return fmt.Errorf("unsupported state %q (use open, closed, or all)", state)
	}
	if sort != "" && !oneOf(sort, "created", "updated", "popularity", "long-running") {
		return fmt.Errorf("unsupported sort %q", sort)
	}
	if direction != "" && !oneOf(direction, "asc", "desc") {
		return fmt.Errorf("unsupported direction %q (use asc or desc)", direction)
	}
	return nil
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func filterPullRequests(cmd *cobra.Command, service *review.Service, prs []*model.PullRequest, author, reviewer string) ([]*model.PullRequest, error) {
	author, reviewer, err := resolveListUsers(cmd, service, author, reviewer)
	if err != nil {
		return nil, err
	}

	filtered := make([]*model.PullRequest, 0, len(prs))
	for _, pr := range prs {
		if matchesPullRequest(pr, author, reviewer) {
			filtered = append(filtered, pr)
		}
	}
	return filtered, nil
}

func resolveListUsers(cmd *cobra.Command, service *review.Service, author, reviewer string) (string, string, error) {
	author, reviewer = strings.TrimSpace(author), strings.TrimSpace(reviewer)
	if author != "@me" && reviewer != "@me" {
		return author, reviewer, nil
	}
	user, err := service.AuthenticatedUser(cmd.Context())
	if err != nil {
		return "", "", err
	}
	if author == "@me" {
		author = user.Login
	}
	if reviewer == "@me" {
		reviewer = user.Login
	}
	return author, reviewer, nil
}

func matchesPullRequest(pr *model.PullRequest, author, reviewer string) bool {
	return (author == "" || pr.User.Login == author) && (reviewer == "" || hasRequestedReviewer(pr, reviewer))
}

func hasRequestedReviewer(pr *model.PullRequest, login string) bool {
	for _, requested := range pr.RequestedReviewers {
		if requested.Login == login {
			return true
		}
	}
	return false
}

func limitPullRequests(prs []*model.PullRequest, limit int) []*model.PullRequest {
	if len(prs) <= limit {
		return prs
	}
	return prs[:limit]
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
