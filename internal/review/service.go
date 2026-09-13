// Package review contains pull request review workflows.
package review

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

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
	return s.ListMatching(ctx, repository, options, 0, nil)
}

// ListMatching lists pull requests across all pages, retaining matching pull
// requests until limit is reached. A zero limit fetches every page.
func (s *Service) ListMatching(ctx context.Context, repository model.RepositoryRef, options interfaces.ListPRsOptions, limit int, matches func(*model.PullRequest) bool) ([]*model.PullRequest, error) {
	pageSize := options.PerPage
	if pageSize < 1 || pageSize > 100 {
		pageSize = 100
	}
	page := options.Page
	if page < 1 {
		page = 1
	}

	prs := make([]*model.PullRequest, 0)
	for {
		pageOptions := options
		pageOptions.PerPage = pageSize
		pageOptions.Page = page
		pagePRs, err := s.provider.ListPullRequests(ctx, repository.Owner, repository.Name, pageOptions)
		if err != nil {
			return nil, fmt.Errorf("list pull requests for %s: %w", repository.String(), err)
		}
		for _, pr := range pagePRs {
			if matches != nil && !matches(pr) {
				continue
			}
			prs = append(prs, pr)
			if limit > 0 && len(prs) >= limit {
				return prs, nil
			}
		}
		if len(pagePRs) < pageSize {
			return prs, nil
		}
		page++
	}
}

// Get returns a pull request by number.
func (s *Service) Get(ctx context.Context, repository model.RepositoryRef, number int) (*model.PullRequest, error) {
	pr, err := s.provider.GetPullRequest(ctx, repository.Owner, repository.Name, number)
	if err != nil {
		return nil, fmt.Errorf("get pull request %s#%d: %w", repository.String(), number, err)
	}
	return pr, nil
}

// ReleaseNotes collects merged pull requests since the supplied inclusive
// timestamp. It is read-only: callers may use its result to publish notes
// through their normal release process.
func (s *Service) ReleaseNotes(ctx context.Context, repository model.RepositoryRef, since time.Time, limit int) (*model.ReleaseNotes, error) {
	fetchLimit := limit
	if limit > 0 {
		fetchLimit++
	}
	prs, err := s.ListMatching(ctx, repository, interfaces.ListPRsOptions{
		State: "closed", Since: since.UTC().Format(time.RFC3339), Sort: "updated", Direction: "asc", PerPage: 100,
	}, fetchLimit, func(pr *model.PullRequest) bool {
		mergedAt, ok := parseMergedAt(pr)
		return ok && !mergedAt.Before(since)
	})
	if err != nil {
		return nil, fmt.Errorf("list merged pull requests for release notes: %w", err)
	}

	sort.SliceStable(prs, func(i, j int) bool {
		left, _ := parseMergedAt(prs[i])
		right, _ := parseMergedAt(prs[j])
		return left.Before(right)
	})
	truncated := limit > 0 && len(prs) > limit
	if truncated {
		prs = prs[:limit]
	}

	contributors := make([]model.User, 0)
	seen := make(map[string]bool)
	for _, pr := range prs {
		if pr.User.Login == "" || seen[pr.User.Login] {
			continue
		}
		seen[pr.User.Login] = true
		contributors = append(contributors, pr.User)
	}

	return &model.ReleaseNotes{
		SchemaVersion: model.ReleaseNotesSchemaVersion,
		Repository:    repository,
		Since:         since.UTC().Format(time.RFC3339),
		Limit:         limit,
		Truncated:     truncated,
		PullRequests:  prs,
		Contributors:  contributors,
	}, nil
}

func parseMergedAt(pr *model.PullRequest) (time.Time, bool) {
	if pr == nil || pr.MergedAt == "" {
		return time.Time{}, false
	}
	mergedAt, err := time.Parse(time.RFC3339, pr.MergedAt)
	return mergedAt, err == nil
}

// Inspect returns a decision-ready review summary for one pull request.
func (s *Service) Inspect(ctx context.Context, repository model.RepositoryRef, number int) (*model.ReviewSummary, error) {
	var (
		pr        *model.PullRequest
		reviews   []*model.Review
		threads   []*model.ReviewThread
		prErr     error
		reviewErr error
		threadErr error
		wg        sync.WaitGroup
	)

	wg.Add(3)
	go func() {
		defer wg.Done()
		pr, prErr = s.Get(ctx, repository, number)
	}()
	go func() {
		defer wg.Done()
		threads, threadErr = s.provider.ListReviewThreads(ctx, repository.Owner, repository.Name, number)
		if threadErr != nil {
			threadErr = fmt.Errorf("list review threads for %s#%d: %w", repository.String(), number, threadErr)
		}
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

	ciStatus := "unavailable"
	if pr.Head.SHA != "" {
		checkRuns, err := s.provider.ListCheckRuns(ctx, repository.Owner, repository.Name, pr.Head.SHA)
		if err == nil {
			ciStatus = summarizeCheckRuns(checkRuns)
		}
	}
	threadState := "unavailable"
	if threadErr == nil {
		threadState = summarizeReviewThreads(threads)
	}
	return summarize(pr, reviews, ciStatus, threadState), nil
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
	return s.AssignedLimited(ctx, repository, 0)
}

// AssignedLimited returns assigned pull requests across all issue pages.
func (s *Service) AssignedLimited(ctx context.Context, repository model.RepositoryRef, limit int) ([]*model.PullRequest, error) {
	user, err := s.AuthenticatedUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("get authenticated user: %w", err)
	}
	const perPage = 100
	prs := make([]*model.PullRequest, 0)
	for page := 1; ; page++ {
		issues, err := s.provider.ListIssues(ctx, repository.Owner, repository.Name, interfaces.ListIssuesOptions{
			State: "open", Assignee: user.Login, PerPage: perPage, Page: page,
		})
		if err != nil {
			return nil, fmt.Errorf("list assigned pull requests: %w", err)
		}
		for _, issue := range issues {
			if issue.PullRequest == nil {
				continue
			}
			prs = append(prs, pullRequestFromIssue(issue))
			if limit > 0 && len(prs) >= limit {
				return prs, nil
			}
		}
		if len(issues) < perPage {
			return prs, nil
		}
	}
}

// Queue returns open pull requests that explicitly request the authenticated
// user's review.
func (s *Service) Queue(ctx context.Context, repository model.RepositoryRef) ([]*model.PullRequest, error) {
	return s.QueueLimited(ctx, repository, 0)
}

// QueueLimited returns requested-review pull requests across all pages.
func (s *Service) QueueLimited(ctx context.Context, repository model.RepositoryRef, limit int) ([]*model.PullRequest, error) {
	user, err := s.AuthenticatedUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("get authenticated user: %w", err)
	}
	return s.ListMatching(ctx, repository, interfaces.ListPRsOptions{State: "open", PerPage: 100}, limit, func(pr *model.PullRequest) bool {
		return hasRequestedReviewer(pr, user.Login)
	})
}

// Mine returns pull requests authored by the authenticated user.
func (s *Service) Mine(ctx context.Context, repository model.RepositoryRef) ([]*model.PullRequest, error) {
	return s.MineLimited(ctx, repository, 0)
}

// MineLimited returns the authenticated user's pull requests across all pages.
func (s *Service) MineLimited(ctx context.Context, repository model.RepositoryRef, limit int) ([]*model.PullRequest, error) {
	user, err := s.AuthenticatedUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("get authenticated user: %w", err)
	}
	return s.ListMatching(ctx, repository, interfaces.ListPRsOptions{State: "all", PerPage: 100}, limit, func(pr *model.PullRequest) bool {
		return pr.User.Login == user.Login
	})
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

func hasRequestedReviewer(pr *model.PullRequest, login string) bool {
	for _, reviewer := range pr.RequestedReviewers {
		if reviewer.Login == login {
			return true
		}
	}
	return false
}

func summarize(pr *model.PullRequest, reviews []*model.Review, ciStatus, threadState string) *model.ReviewSummary {
	readiness := model.ReviewReadiness{
		Mergeable:          pr.Mergeable,
		MergeableState:     pr.MergeableState,
		CIStatus:           ciStatus,
		ReviewThreadsState: threadState,
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

func summarizeReviewThreads(threads []*model.ReviewThread) string {
	if len(threads) == 0 {
		return "none"
	}
	for _, thread := range threads {
		if thread == nil || !thread.IsResolved {
			return "unresolved"
		}
	}
	return "resolved"
}

// summarizeCheckRuns collapses GitHub's per-check status into the readiness
// signal consumed by the review summary.
func summarizeCheckRuns(checkRuns []*model.CheckRun) string {
	if len(checkRuns) == 0 {
		return "none"
	}

	pending := false
	for _, checkRun := range checkRuns {
		if checkRun.Status != "completed" {
			pending = true
			continue
		}
		switch checkRun.Conclusion {
		case "success", "neutral", "skipped":
			continue
		default:
			return "failure"
		}
	}
	if pending {
		return "pending"
	}
	return "success"
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
	if summary.Readiness.CIStatus == "failure" {
		summary.RiskSignals = append(summary.RiskSignals, model.RiskSignal{
			Kind: "ci_failure", Severity: "high", Detail: "one or more CI checks have failed",
		})
	}
	if summary.Readiness.ReviewThreadsState == "unresolved" {
		summary.RiskSignals = append(summary.RiskSignals, model.RiskSignal{
			Kind: "unresolved_review_threads", Severity: "medium", Detail: "one or more review threads are unresolved",
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
	switch readiness.CIStatus {
	case "failure":
		summary.RecommendedActions = append(summary.RecommendedActions, model.RecommendedAction{
			Action: "fix_ci", Reason: "one or more CI checks have failed",
		})
	case "pending":
		summary.RecommendedActions = append(summary.RecommendedActions, model.RecommendedAction{
			Action: "wait_for_ci", Reason: "CI checks are still running",
		})
	case "unavailable":
		summary.RecommendedActions = append(summary.RecommendedActions, model.RecommendedAction{
			Action: "check_ci", Reason: "CI status is not available from the configured provider",
		})
	}
	if readiness.ReviewThreadsState == "unresolved" {
		summary.RecommendedActions = append(summary.RecommendedActions, model.RecommendedAction{
			Action: "resolve_review_threads", Reason: "one or more review threads are unresolved",
		})
	}
}
