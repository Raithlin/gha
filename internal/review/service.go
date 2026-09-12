// Package review contains pull request review workflows.
package review

import (
	"context"
	"fmt"
	"sync"

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

// List returns pull requests matching options for a repository.
func (s *Service) List(ctx context.Context, repository model.RepositoryRef, options interfaces.ListPRsOptions) ([]*model.PullRequest, error) {
	prs, err := s.provider.ListPullRequests(ctx, repository.Owner, repository.Name, options)
	if err != nil {
		return nil, fmt.Errorf("list pull requests for %s: %w", repository.String(), err)
	}
	return prs, nil
}

// Get returns a pull request by number.
func (s *Service) Get(ctx context.Context, repository model.RepositoryRef, number int) (*model.PullRequest, error) {
	pr, err := s.provider.GetPullRequest(ctx, repository.Owner, repository.Name, number)
	if err != nil {
		return nil, fmt.Errorf("get pull request %s#%d: %w", repository.String(), number, err)
	}
	return pr, nil
}

// Inspect returns a decision-ready review summary for one pull request.
func (s *Service) Inspect(ctx context.Context, repository model.RepositoryRef, number int) (*model.ReviewSummary, error) {
	var (
		pr        *model.PullRequest
		reviews   []*model.Review
		prErr     error
		reviewErr error
		wg        sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		pr, prErr = s.Get(ctx, repository, number)
	}()
	go func() {
		defer wg.Done()
		reviews, reviewErr = s.provider.ListReviews(ctx, repository.Owner, repository.Name, number)
		if reviewErr != nil {
			reviewErr = fmt.Errorf("list reviews for %s#%d: %w", repository.String(), number, reviewErr)
		}
	}()
	wg.Wait()

	if prErr != nil {
		return nil, prErr
	}
	if reviewErr != nil {
		return nil, reviewErr
	}
	if pr == nil {
		return nil, fmt.Errorf("get pull request %s#%d: provider returned no pull request", repository.String(), number)
	}
	if reviews == nil {
		reviews = make([]*model.Review, 0)
	}
	return summarize(pr, reviews), nil
}

// AuthenticatedUser returns the user associated with the provider credentials.
func (s *Service) AuthenticatedUser(ctx context.Context) (*model.User, error) {
	user, err := s.provider.GetAuthenticatedUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("get authenticated user: %w", err)
	}
	return user, nil
}

// Assigned returns open pull requests assigned to the authenticated user.
func (s *Service) Assigned(ctx context.Context, repository model.RepositoryRef) ([]*model.PullRequest, error) {
	user, err := s.AuthenticatedUser(ctx)
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
	user, err := s.AuthenticatedUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("get authenticated user: %w", err)
	}
	prs, err := s.List(ctx, repository, interfaces.ListPRsOptions{State: "open", PerPage: 100})
	if err != nil {
		return nil, err
	}
	return filterRequestedReviewers(prs, user.Login), nil
}

// Mine returns pull requests authored by the authenticated user.
func (s *Service) Mine(ctx context.Context, repository model.RepositoryRef) ([]*model.PullRequest, error) {
	user, err := s.AuthenticatedUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("get authenticated user: %w", err)
	}
	prs, err := s.List(ctx, repository, interfaces.ListPRsOptions{State: "all", PerPage: 100})
	if err != nil {
		return nil, err
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

func summarize(pr *model.PullRequest, reviews []*model.Review) *model.ReviewSummary {
	readiness := model.ReviewReadiness{
		Mergeable:          pr.Mergeable,
		MergeableState:     pr.MergeableState,
		CIStatus:           "unavailable",
		ReviewThreadsState: "unavailable",
		ApprovedBy:         make([]model.User, 0),
		ChangesRequestedBy: make([]model.User, 0),
		PendingReviewers:   append(make([]model.User, 0, len(pr.RequestedReviewers)), pr.RequestedReviewers...),
	}
	latestReviews := latestReviewsByUser(reviews)
	for _, review := range latestReviews {
		switch review.State {
		case "APPROVED":
			readiness.ApprovedBy = append(readiness.ApprovedBy, review.User)
		case "CHANGES_REQUESTED":
			readiness.ChangesRequestedBy = append(readiness.ChangesRequestedBy, review.User)
		}
	}

	summary := &model.ReviewSummary{
		SchemaVersion:      model.ReviewSummarySchemaVersion,
		PullRequest:        pr,
		Reviews:            reviews,
		Readiness:          readiness,
		RiskSignals:        make([]model.RiskSignal, 0),
		RecommendedActions: make([]model.RecommendedAction, 0),
	}
	addRiskSignals(summary)
	addRecommendedActions(summary)
	return summary
}

func latestReviewsByUser(reviews []*model.Review) map[string]*model.Review {
	latest := make(map[string]*model.Review, len(reviews))
	for _, review := range reviews {
		if previous, ok := latest[review.User.Login]; !ok || review.SubmittedAt >= previous.SubmittedAt {
			latest[review.User.Login] = review
		}
	}
	return latest
}

func addRiskSignals(summary *model.ReviewSummary) {
	pr := summary.PullRequest
	if pr.Mergeable != nil && !*pr.Mergeable {
		summary.RiskSignals = append(summary.RiskSignals, model.RiskSignal{
			Kind: "merge_conflict", Severity: "high", Detail: "GitHub reports that the pull request is not mergeable",
		})
	}
	if pr.ChangedFiles >= 75 {
		summary.RiskSignals = append(summary.RiskSignals, model.RiskSignal{
			Kind: "large_change", Severity: "high", Detail: fmt.Sprintf("%d files changed", pr.ChangedFiles),
		})
	} else if pr.ChangedFiles >= 25 {
		summary.RiskSignals = append(summary.RiskSignals, model.RiskSignal{
			Kind: "large_change", Severity: "medium", Detail: fmt.Sprintf("%d files changed", pr.ChangedFiles),
		})
	}
	if len(summary.Readiness.ChangesRequestedBy) > 0 {
		summary.RiskSignals = append(summary.RiskSignals, model.RiskSignal{
			Kind: "changes_requested", Severity: "high", Detail: "a reviewer has requested changes",
		})
	}
}

func addRecommendedActions(summary *model.ReviewSummary) {
	readiness := summary.Readiness
	if readiness.Mergeable != nil && !*readiness.Mergeable {
		summary.RecommendedActions = append(summary.RecommendedActions, model.RecommendedAction{
			Action: "resolve_merge_conflicts", Reason: "the pull request is not mergeable",
		})
	}
	if len(readiness.ChangesRequestedBy) > 0 {
		summary.RecommendedActions = append(summary.RecommendedActions, model.RecommendedAction{
			Action: "address_requested_changes", Reason: "a reviewer has requested changes", Reviewers: readiness.ChangesRequestedBy,
		})
	}
	if len(readiness.PendingReviewers) > 0 {
		summary.RecommendedActions = append(summary.RecommendedActions, model.RecommendedAction{
			Action: "wait_for_review", Reason: "review is still requested", Reviewers: readiness.PendingReviewers,
		})
	}
	summary.RecommendedActions = append(summary.RecommendedActions, model.RecommendedAction{
		Action: "check_ci", Reason: "CI status is not available from the configured provider",
	})
}
