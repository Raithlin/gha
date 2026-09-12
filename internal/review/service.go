// Package review contains pull request review workflows.
package review

import (
	"context"
	"fmt"

	"github.com/raithlin/gha/internal/interfaces"
	"github.com/raithlin/gha/pkg/model"
)

// Service coordinates pull request review workflows through a provider.
type Service struct {
	provider interfaces.GitHubProvider
}

// NewService creates a review service backed by provider.
func NewService(provider interfaces.GitHubProvider) *Service {
	return &Service{provider: provider}
}

// Get returns a pull request by number.
func (s *Service) Get(ctx context.Context, repository model.RepositoryRef, number int) (*model.PullRequest, error) {
	pr, err := s.provider.GetPullRequest(ctx, repository.Owner, repository.Name, number)
	if err != nil {
		return nil, fmt.Errorf("get pull request %s#%d: %w", repository.String(), number, err)
	}
	return pr, nil
}

// Assigned returns open pull requests assigned to the authenticated user.
func (s *Service) Assigned(ctx context.Context, repository model.RepositoryRef) ([]*model.PullRequest, error) {
	user, err := s.provider.GetAuthenticatedUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("get authenticated user: %w", err)
	}
	issues, err := s.provider.ListIssues(ctx, repository.Owner, repository.Name, interfaces.ListIssuesOptions{
		State: "open", Assignee: user.Login, PerPage: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("list assigned pull requests: %w", err)
	}

	prs := make([]*model.PullRequest, 0, len(issues))
	for _, issue := range issues {
		if issue.PullRequest == nil {
			continue
		}
		prs = append(prs, pullRequestFromIssue(issue))
	}
	return prs, nil
}

// Queue returns open pull requests that explicitly request the authenticated
// user's review.
func (s *Service) Queue(ctx context.Context, repository model.RepositoryRef) ([]*model.PullRequest, error) {
	user, err := s.provider.GetAuthenticatedUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("get authenticated user: %w", err)
	}
	prs, err := s.provider.ListPullRequests(ctx, repository.Owner, repository.Name, interfaces.ListPRsOptions{State: "open", PerPage: 100})
	if err != nil {
		return nil, fmt.Errorf("list pull requests: %w", err)
	}
	return filterRequestedReviewers(prs, user.Login), nil
}

// Mine returns pull requests authored by the authenticated user.
func (s *Service) Mine(ctx context.Context, repository model.RepositoryRef) ([]*model.PullRequest, error) {
	user, err := s.provider.GetAuthenticatedUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("get authenticated user: %w", err)
	}
	prs, err := s.provider.ListPullRequests(ctx, repository.Owner, repository.Name, interfaces.ListPRsOptions{State: "all", PerPage: 100})
	if err != nil {
		return nil, fmt.Errorf("list pull requests: %w", err)
	}

	filtered := make([]*model.PullRequest, 0, len(prs))
	for _, pr := range prs {
		if pr.User.Login == user.Login {
			filtered = append(filtered, pr)
		}
	}
	return filtered, nil
}

func pullRequestFromIssue(issue *model.Issue) *model.PullRequest {
	return &model.PullRequest{
		ID:        issue.ID,
		Number:    issue.Number,
		Title:     issue.Title,
		Body:      issue.Body,
		State:     issue.State,
		User:      issue.User,
		CreatedAt: issue.CreatedAt,
		UpdatedAt: issue.UpdatedAt,
		ClosedAt:  issue.ClosedAt,
	}
}

func filterRequestedReviewers(prs []*model.PullRequest, login string) []*model.PullRequest {
	filtered := make([]*model.PullRequest, 0, len(prs))
	for _, pr := range prs {
		for _, reviewer := range pr.RequestedReviewers {
			if reviewer.Login == login {
				filtered = append(filtered, pr)
				break
			}
		}
	}
	return filtered
}
