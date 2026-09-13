package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/interfaces"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/internal/review"
	"github.com/raithlin/gha/pkg/model"
)

// newPRsCmd constructs the pull request listing command with explicit dependencies.
//
//nolint:gocyclo // The command owns validation and four provider-backed listing modes.
func newPRsCmd(service *review.Service, resolver *git.RepositoryResolver) *cobra.Command {
	var assigned, queue, mine bool
	var repository, path, format, state, author, reviewer, base, head, sort, direction, since string
	var limit int

	command := &cobra.Command{
		Use:   "prs",
		Short: "List pull requests",
		Long: `List pull requests in a GitHub repository.

The repository is taken from --repo, the origin remote in --path,
GHA_REPOSITORY, or the current directory's origin remote (in that order). Use
gha review <number> to inspect one pull request.`,
		Args: noArgsWithFormat(&format),
		RunE: func(cmd *cobra.Command, _ []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			if boolCount(assigned, queue, mine) > 1 {
				return renderCommandError(cmd, outputFormat, "invalid_argument", fmt.Errorf("use only one of --assigned, --queue, or --mine"))
			}
			if (assigned || queue || mine) && hasListFilters(cmd) {
				return renderCommandError(cmd, outputFormat, "invalid_argument", fmt.Errorf("--assigned, --queue, and --mine cannot be combined with list filters"))
			}
			if limit < 1 || limit > 100 {
				return renderCommandError(cmd, outputFormat, "invalid_argument", fmt.Errorf("limit must be between 1 and 100"))
			}
			if !assigned && !queue && !mine {
				if err := validateListOptions(state, sort, direction, since); err != nil {
					return renderCommandError(cmd, outputFormat, "invalid_argument", err)
				}
			}
			target, err := resolver.ResolveAtPath(cmd.Context(), repository, path)
			if err != nil {
				return renderCommandError(cmd, outputFormat, "repository_resolution_failed", err)
			}
			fetchLimit := limit + 1

			switch {
			case assigned:
				prs, err := service.AssignedLimited(cmd.Context(), target, fetchLimit)
				if err != nil {
					return renderCommandError(cmd, outputFormat, "pull_request_list_failed", err)
				}
				return output.PullRequestList(cmd.OutOrStdout(), outputFormat, pullRequestList(target, limit, prs), "Pull Requests Assigned to You")
			case queue:
				prs, err := service.QueueLimited(cmd.Context(), target, fetchLimit)
				if err != nil {
					return renderCommandError(cmd, outputFormat, "pull_request_list_failed", err)
				}
				return output.PullRequestList(cmd.OutOrStdout(), outputFormat, pullRequestList(target, limit, prs), "Pull Requests Awaiting Your Review")
			case mine:
				prs, err := service.MineLimited(cmd.Context(), target, fetchLimit)
				if err != nil {
					return renderCommandError(cmd, outputFormat, "pull_request_list_failed", err)
				}
				return output.PullRequestList(cmd.OutOrStdout(), outputFormat, pullRequestList(target, limit, prs), "Your Pull Requests")
			default:
				perPage := limit
				if author != "" || reviewer != "" {
					perPage = 100
				}
				author, reviewer, err = resolveListUsers(cmd, service, author, reviewer)
				if err != nil {
					return renderCommandError(cmd, outputFormat, "authenticated_user_failed", err)
				}
				if head != "" && !strings.Contains(head, ":") {
					user, err := service.AuthenticatedUser(cmd.Context())
					if err != nil {
						return renderCommandError(cmd, outputFormat, "authenticated_user_failed", err)
					}
					head = user.Login + ":" + head
				}
				prs, err := service.ListMatching(cmd.Context(), target, interfaces.ListPRsOptions{
					State: state, Head: head, Base: base, Sort: sort, Direction: direction, Since: since, PerPage: perPage,
				}, fetchLimit, func(pr *model.PullRequest) bool {
					return matchesPullRequest(pr, author, reviewer)
				})
				if err != nil {
					return renderCommandError(cmd, outputFormat, "pull_request_list_failed", err)
				}
				return output.PullRequestList(cmd.OutOrStdout(), outputFormat, pullRequestList(target, limit, prs), "Pull Requests")
			}
		},
	}

	command.Flags().BoolVarP(&assigned, "assigned", "a", false, "List pull requests assigned to you")
	command.Flags().BoolVarP(&queue, "queue", "q", false, "List pull requests awaiting your review")
	command.Flags().BoolVarP(&mine, "mine", "m", false, "List pull requests authored by you")
	command.Flags().StringVarP(&repository, "repo", "r", "", "Repository to inspect (owner/repo)")
	command.Flags().StringVar(&path, "path", "", "Local Git checkout whose origin selects the repository")
	command.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().StringVar(&state, "state", "open", "Filter by state (open, closed, all)")
	command.Flags().StringVar(&author, "author", "", "Filter by author login (use @me for yourself)")
	command.Flags().StringVar(&reviewer, "reviewer", "", "Filter by requested reviewer login (use @me for yourself)")
	command.Flags().StringVar(&base, "base", "", "Filter by base branch")
	command.Flags().StringVar(&head, "head", "", "Filter by head branch (bare branch uses your login; owner:branch also accepted)")
	command.Flags().StringVar(&sort, "sort", "", "Sort by created, updated, popularity, or long-running")
	command.Flags().StringVar(&direction, "direction", "", "Sort direction (asc or desc)")
	command.Flags().StringVar(&since, "since", "", "Return pull requests updated since this RFC 3339 timestamp")
	command.Flags().IntVarP(&limit, "limit", "l", 30, "Maximum pull requests to return (1-100)")
	command.SilenceUsage = true
	command.SilenceErrors = true
	return command
}

func hasListFilters(cmd *cobra.Command) bool {
	for _, name := range []string{"state", "author", "reviewer", "base", "head", "sort", "direction", "since"} {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}

func validateListOptions(state, sort, direction, since string) error {
	if !oneOf(state, "open", "closed", "all") {
		return fmt.Errorf("unsupported state %q (use open, closed, or all)", state)
	}
	if sort != "" && !oneOf(sort, "created", "updated", "popularity", "long-running") {
		return fmt.Errorf("unsupported sort %q", sort)
	}
	if direction != "" && !oneOf(direction, "asc", "desc") {
		return fmt.Errorf("unsupported direction %q (use asc or desc)", direction)
	}
	if since != "" {
		if _, err := time.Parse(time.RFC3339, since); err != nil {
			return fmt.Errorf("invalid --since %q (use RFC 3339)", since)
		}
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

func pullRequestList(repository model.RepositoryRef, limit int, prs []*model.PullRequest) *model.PullRequestList {
	truncated := len(prs) > limit
	return &model.PullRequestList{
		SchemaVersion: model.PullRequestListSchemaVersion,
		Repository:    repository,
		Limit:         limit,
		Truncated:     truncated,
		PullRequests:  limitPullRequests(prs, limit),
	}
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
