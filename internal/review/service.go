// Package review contains pull request review workflows.
package review

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/raithlin/gha/internal/interfaces"
	"github.com/raithlin/gha/pkg/model"
)

// Service coordinates pull request review workflows through a provider.
type Service struct {
	provider interfaces.CodeHostProvider
}

// PreparePullRequestInput holds a resolved pull-request request before a
// provider mutation is allowed.
type PreparePullRequestInput struct {
	Repository model.RepositoryRef
	Title      string
	Body       string
	Head       string
	Base       string
	DryRun     bool
}

// PreparePullRequest combines provider facts into a stable, read-only plan.
// Individual optional facts are marked unavailable rather than inferred.
func (s *Service) PreparePullRequest(ctx context.Context, input PreparePullRequestInput) (*model.PullRequestPreparation, error) {
	if s == nil || s.provider == nil {
		return nil, fmt.Errorf("pull request preparation is not configured")
	}
	repository, err := s.provider.GetRepository(ctx, input.Repository.Owner, input.Repository.Name)
	if err != nil {
		return nil, fmt.Errorf("get repository %s: %w", input.Repository.String(), err)
	}
	if repository == nil {
		return nil, fmt.Errorf("get repository %s: provider returned no repository", input.Repository.String())
	}
	base := input.Base
	if base == "" {
		base = repository.DefaultBranch
	}
	if base == "" {
		return nil, fmt.Errorf("resolve base branch: provider did not report a default branch")
	}
	permissions, canPush := pullRequestPermissions(repository)
	preparation := &model.PullRequestPreparation{
		SchemaVersion:        model.PullRequestPreparationSchemaVersion,
		Repository:           input.Repository,
		Title:                input.Title,
		Body:                 input.Body,
		Head:                 input.Head,
		Base:                 base,
		DryRun:               input.DryRun,
		Creation:             "planned",
		Permissions:          permissions,
		CanPush:              canPush,
		ExistingPullRequests: []*model.PullRequest{},
		RiskSignals:          []model.RiskSignal{},
		RecommendedActions:   []model.RecommendedAction{},
	}
	if input.Head == base {
		preparation.Comparison = model.BranchComparison{State: "not_applicable", Message: "head and base are the same branch"}
		preparation.RiskSignals = append(preparation.RiskSignals, model.RiskSignal{Kind: "same_branch", Severity: "high", Detail: "head and base resolve to the same branch"})
		preparation.RecommendedActions = pullRequestActions(preparation)
		return preparation, nil
	}
	s.addPullRequestComparison(ctx, preparation)
	s.addExistingPullRequests(ctx, preparation)
	preparation.RecommendedActions = pullRequestActions(preparation)
	return preparation, nil
}

func pullRequestPermissions(repository *model.Repository) (model.ProviderSignal, *bool) {
	if repository.Permissions == nil {
		return model.ProviderSignal{State: "unavailable", Message: "provider did not report caller permissions"}, nil
	}
	canPush := repository.Permissions.Push || repository.Permissions.Admin
	return model.ProviderSignal{State: "available"}, &canPush
}

func (s *Service) addPullRequestComparison(ctx context.Context, preparation *model.PullRequestPreparation) {
	comparison, err := s.provider.CompareBranches(ctx, preparation.Repository.Owner, preparation.Repository.Name, preparation.Base, preparation.Head)
	if err != nil || comparison == nil {
		message := "provider did not return a branch comparison"
		if err != nil {
			message = err.Error()
		}
		preparation.Comparison = model.BranchComparison{State: "unavailable", Message: message}
		preparation.RiskSignals = append(preparation.RiskSignals, model.RiskSignal{Kind: "comparison_unavailable", Severity: "medium", Detail: message})
		return
	}
	preparation.Comparison = *comparison
	if comparison.AheadBy == 0 {
		preparation.RiskSignals = append(preparation.RiskSignals, model.RiskSignal{Kind: "no_changes", Severity: "high", Detail: "head has no commits ahead of base"})
	}
}

func (s *Service) addExistingPullRequests(ctx context.Context, preparation *model.PullRequestPreparation) {
	headFilter := preparation.Head
	if !strings.Contains(headFilter, ":") {
		headFilter = preparation.Repository.Owner + ":" + headFilter
	}
	prs, err := s.provider.ListPullRequests(ctx, preparation.Repository.Owner, preparation.Repository.Name, interfaces.ListPRsOptions{State: "open", Head: headFilter, Base: preparation.Base, PerPage: 100})
	if err != nil {
		preparation.ExistingRequests = model.ProviderSignal{State: "unavailable", Message: err.Error()}
		preparation.RiskSignals = append(preparation.RiskSignals, model.RiskSignal{Kind: "existing_requests_unavailable", Severity: "medium", Detail: err.Error()})
		return
	}
	preparation.ExistingRequests = model.ProviderSignal{State: "available"}
	preparation.ExistingPullRequests = prs
	if len(prs) > 0 {
		preparation.RiskSignals = append(preparation.RiskSignals, model.RiskSignal{Kind: "existing_pull_request", Severity: "high", Detail: "an open pull request already uses this head and base"})
	}
}

// CreatePreparedPullRequest writes only a preflight that remains safe to create.
func (s *Service) CreatePreparedPullRequest(ctx context.Context, preparation *model.PullRequestPreparation) (*model.PullRequest, error) {
	if preparation == nil {
		return nil, fmt.Errorf("pull request preparation is required")
	}
	if err := validatePullRequestPreparation(preparation); err != nil {
		return nil, err
	}
	pr, err := s.provider.CreatePullRequest(ctx, preparation.Repository.Owner, preparation.Repository.Name, &model.PullRequestInput{Title: preparation.Title, Body: preparation.Body, Head: preparation.Head, Base: preparation.Base})
	if err != nil {
		return nil, fmt.Errorf("create pull request for %s: %w", preparation.Repository.String(), err)
	}
	return pr, nil
}

func validatePullRequestPreparation(preparation *model.PullRequestPreparation) error {
	for _, signal := range preparation.RiskSignals {
		if signal.Severity == "high" {
			if signal.Kind == "existing_pull_request" {
				return fmt.Errorf("an open pull request already exists for %s into %s", preparation.Head, preparation.Base)
			}
			return fmt.Errorf("pull request is not ready to create: %s", signal.Detail)
		}
	}
	if preparation.Comparison.State == "unavailable" {
		return fmt.Errorf("pull request comparison is unavailable; inspect the branches and retry")
	}
	if preparation.ExistingRequests.State != "available" {
		return fmt.Errorf("existing pull request lookup is unavailable; inspect the branch and retry")
	}
	if preparation.Permissions.State != "available" || preparation.CanPush == nil {
		return fmt.Errorf("pull request creation permission is unavailable; authenticate with a provider credential that can create pull requests and retry")
	}
	if !*preparation.CanPush {
		return fmt.Errorf("caller does not have permission to create a pull request")
	}
	return nil
}

func pullRequestActions(preparation *model.PullRequestPreparation) []model.RecommendedAction {
	actions := make([]model.RecommendedAction, 0)
	for _, signal := range preparation.RiskSignals {
		switch signal.Kind {
		case "no_changes":
			actions = append(actions, model.RecommendedAction{Action: "add_commits", Reason: "push commits ahead of the base branch before creating a pull request"})
		case "existing_pull_request":
			actions = append(actions, model.RecommendedAction{Action: "review_existing", Reason: "use gha review <number> to inspect the existing pull request"})
		case "comparison_unavailable":
			actions = append(actions, model.RecommendedAction{Action: "inspect_branches", Reason: "verify the selected refs before creating a pull request"})
		}
	}
	if preparation.ExistingRequests.State != "available" {
		actions = append(actions, model.RecommendedAction{Action: "inspect_existing_requests", Reason: "verify that no open pull request already uses the selected head and base"})
	}
	if preparation.Permissions.State != "available" || preparation.CanPush == nil {
		actions = append(actions, model.RecommendedAction{Action: "authorize", Reason: "use provider credentials that can create pull requests before retrying"})
	} else if !*preparation.CanPush {
		actions = append(actions, model.RecommendedAction{Action: "request_permission", Reason: "request permission to create pull requests for this repository"})
	}
	if len(actions) == 0 {
		actions = append(actions, model.RecommendedAction{Action: "create", Reason: "rerun with gha pr create --confirm to create this pull request"})
	}
	return actions
}

// NewService creates a review service backed by provider.
func NewService(provider interfaces.CodeHostProvider) *Service {
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

// ListPublishedReleases returns a bounded listing of non-draft releases. It
// fetches one additional published result so callers can distinguish an empty
// tail from a result truncated by their requested limit.
func (s *Service) ListPublishedReleases(ctx context.Context, repository model.RepositoryRef, limit int) (*model.ReleaseList, error) {
	if s == nil || s.provider == nil {
		return nil, fmt.Errorf("release listing is not configured")
	}
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("limit must be between 1 and 100")
	}

	fetchLimit := limit + 1
	pageSize := fetchLimit
	if pageSize > 100 {
		pageSize = 100
	}
	published := make([]*model.Release, 0, fetchLimit)
	for page := 1; ; page++ {
		releases, err := s.provider.ListReleases(ctx, repository.Owner, repository.Name, interfaces.ListReleasesOptions{PerPage: pageSize, Page: page})
		if err != nil {
			return nil, fmt.Errorf("list releases for %s: %w", repository.String(), err)
		}
		for _, release := range releases {
			if release == nil || release.Draft {
				continue
			}
			published = append(published, release)
			if len(published) >= fetchLimit {
				return publishedReleaseList(repository, limit, published), nil
			}
		}
		if len(releases) < pageSize {
			return publishedReleaseList(repository, limit, published), nil
		}
	}
}

func publishedReleaseList(repository model.RepositoryRef, limit int, releases []*model.Release) *model.ReleaseList {
	truncated := len(releases) > limit
	if truncated {
		releases = releases[:limit]
	}
	return &model.ReleaseList{
		SchemaVersion: model.ReleaseListSchemaVersion,
		Repository:    repository,
		Limit:         limit,
		Truncated:     truncated,
		Releases:      releases,
	}
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
	if user == nil {
		return nil, fmt.Errorf("get authenticated user: provider returned no user")
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
