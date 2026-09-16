package review

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/internal/interfaces"
	"github.com/raithlin/gha/pkg/model"
)

type fakeProvider struct {
	user           *model.User
	userErr        error
	repository     *model.Repository
	comparison     *model.BranchComparison
	pr             *model.PullRequest
	prErr          error
	prs            []*model.PullRequest
	prsErr         error
	issues         []*model.Issue
	issuesErr      error
	reviews        []*model.Review
	reviewsErr     error
	checks         []*model.CheckRun
	checkErr       error
	threads        []*model.ReviewThread
	threadErr      error
	prPages        map[int][]*model.PullRequest
	issuePages     map[int][]*model.Issue
	prOptions      []interfaces.ListPRsOptions
	issueOptions   []interfaces.ListIssuesOptions
	releases       []*model.Release
	releasesErr    error
	releasePages   map[int][]*model.Release
	releaseOptions []interfaces.ListReleasesOptions
	created        int
	createErr      error
}

func (p *fakeProvider) GetAuthenticatedUser(context.Context) (*model.User, error) {
	return p.user, p.userErr
}

func (p *fakeProvider) ListRepositories(context.Context) ([]*model.Repository, error) {
	return nil, errors.New("not implemented")
}

func (p *fakeProvider) GetRepository(context.Context, string, string) (*model.Repository, error) {
	if p.repository == nil {
		return nil, errors.New("not implemented")
	}
	return p.repository, nil
}

func (p *fakeProvider) ListPullRequests(_ context.Context, _ string, _ string, options interfaces.ListPRsOptions) ([]*model.PullRequest, error) {
	p.prOptions = append(p.prOptions, options)
	if p.prPages != nil {
		return p.prPages[options.Page], p.prsErr
	}
	return p.prs, p.prsErr
}

func (p *fakeProvider) ListReleases(_ context.Context, _ string, _ string, options interfaces.ListReleasesOptions) ([]*model.Release, error) {
	p.releaseOptions = append(p.releaseOptions, options)
	if p.releasePages != nil {
		return p.releasePages[options.Page], p.releasesErr
	}
	return p.releases, p.releasesErr
}

func (p *fakeProvider) GetPullRequest(context.Context, string, string, int) (*model.PullRequest, error) {
	return p.pr, p.prErr
}

func (p *fakeProvider) CreatePullRequest(context.Context, string, string, *model.PullRequestInput) (*model.PullRequest, error) {
	p.created++
	return &model.PullRequest{Number: p.created}, p.createErr
}

func (p *fakeProvider) CompareBranches(context.Context, string, string, string, string) (*model.BranchComparison, error) {
	if p.comparison == nil {
		return nil, errors.New("not implemented")
	}
	return p.comparison, nil
}

func TestPreparePullRequestUsesDefaultBaseAndQualifiedHeadFilter(t *testing.T) {
	provider := &fakeProvider{
		repository: &model.Repository{DefaultBranch: "main", Permissions: &model.RepositoryPermissions{Push: true}},
		comparison: &model.BranchComparison{State: "ahead", AheadBy: 2},
	}
	service := NewService(provider)

	preparation, err := service.PreparePullRequest(context.Background(), PreparePullRequestInput{
		Repository: model.RepositoryRef{Owner: "acme", Name: "project"}, Title: "Improve reviews", Head: "feature",
	})

	require.NoError(t, err)
	assert.Equal(t, "main", preparation.Base)
	assert.Equal(t, "available", preparation.ExistingRequests.State)
	require.Len(t, provider.prOptions, 1)
	assert.Equal(t, "acme:feature", provider.prOptions[0].Head)
	assert.Equal(t, "main", provider.prOptions[0].Base)
}

func (p *fakeProvider) UpdatePullRequest(context.Context, string, string, int, *model.PullRequestInput) (*model.PullRequest, error) {
	return nil, errors.New("not implemented")
}

func (p *fakeProvider) ListIssues(_ context.Context, _ string, _ string, options interfaces.ListIssuesOptions) ([]*model.Issue, error) {
	p.issueOptions = append(p.issueOptions, options)
	if p.issuePages != nil {
		return p.issuePages[options.Page], p.issuesErr
	}
	return p.issues, p.issuesErr
}

func (p *fakeProvider) GetIssue(context.Context, string, string, int) (*model.Issue, error) {
	return nil, errors.New("not implemented")
}

func (p *fakeProvider) AddComment(context.Context, string, string, int, string) (*model.Comment, error) {
	return nil, errors.New("not implemented")
}

func (p *fakeProvider) ListReviews(context.Context, string, string, int) ([]*model.Review, error) {
	return p.reviews, p.reviewsErr
}

func (p *fakeProvider) ListCheckRuns(context.Context, string, string, string) ([]*model.CheckRun, error) {
	return p.checks, p.checkErr
}

func (p *fakeProvider) ListReviewThreads(context.Context, string, string, int) ([]*model.ReviewThread, error) {
	return p.threads, p.threadErr
}

func (p *fakeProvider) SubmitReview(context.Context, string, string, int, *model.ReviewInput) (*model.Review, error) {
	return nil, errors.New("not implemented")
}

func TestListPublishedReleasesExcludesDraftsAcrossPagesAndMakesTruncationExplicit(t *testing.T) {
	provider := &fakeProvider{releasePages: map[int][]*model.Release{
		1: {{TagName: "v1.2.2-draft", Draft: true}, {TagName: "v1.2.1-draft", Draft: true}, {TagName: "v1.2.0-draft", Draft: true}},
		2: {{TagName: "v1.2.0"}, {TagName: "v1.1.0"}, {TagName: "v1.0.0"}},
	}}
	service := NewService(provider)

	releases, err := service.ListPublishedReleases(context.Background(), model.RepositoryRef{Owner: "Raithlin", Name: "gha"}, 2)

	require.NoError(t, err)
	assert.Equal(t, model.ReleaseListSchemaVersion, releases.SchemaVersion)
	assert.True(t, releases.Truncated)
	require.Len(t, releases.Releases, 2)
	assert.Equal(t, "v1.2.0", releases.Releases[0].TagName)
	assert.Equal(t, "v1.1.0", releases.Releases[1].TagName)
	require.Len(t, provider.releaseOptions, 2)
	assert.Equal(t, 3, provider.releaseOptions[0].PerPage)
	assert.Equal(t, 1, provider.releaseOptions[0].Page)
	assert.Equal(t, 2, provider.releaseOptions[1].Page)
}

func TestCreatePreparedPullRequestFailsClosedWhenSafetySignalsAreNotAvailable(t *testing.T) {
	canPush := true
	canNotPush := false
	tests := []struct {
		name        string
		preparation *model.PullRequestPreparation
		wantError   string
	}{
		{
			name: "existing pull request lookup unavailable",
			preparation: &model.PullRequestPreparation{
				Repository:       model.RepositoryRef{Owner: "Raithlin", Name: "gha"},
				Head:             "feature/api",
				Base:             "master",
				Comparison:       model.BranchComparison{State: "ahead"},
				Permissions:      model.ProviderSignal{State: "available"},
				CanPush:          &canPush,
				ExistingRequests: model.ProviderSignal{State: "unavailable", Message: "GitHub API error: 404 Not Found"},
			},
			wantError: "existing pull request lookup is unavailable",
		},
		{
			name: "repository permission unavailable",
			preparation: &model.PullRequestPreparation{
				Repository:       model.RepositoryRef{Owner: "Raithlin", Name: "gha"},
				Head:             "feature/api",
				Base:             "master",
				Comparison:       model.BranchComparison{State: "ahead"},
				Permissions:      model.ProviderSignal{State: "unavailable"},
				ExistingRequests: model.ProviderSignal{State: "available"},
			},
			wantError: "pull request creation permission is unavailable",
		},
		{
			name: "repository permission denied",
			preparation: &model.PullRequestPreparation{
				Repository:       model.RepositoryRef{Owner: "Raithlin", Name: "gha"},
				Head:             "feature/api",
				Base:             "master",
				Comparison:       model.BranchComparison{State: "ahead"},
				Permissions:      model.ProviderSignal{State: "available"},
				CanPush:          &canNotPush,
				ExistingRequests: model.ProviderSignal{State: "available"},
			},
			wantError: "caller does not have permission to create a pull request",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := &fakeProvider{}
			_, err := NewService(provider).CreatePreparedPullRequest(context.Background(), test.preparation)

			require.Error(t, err)
			assert.ErrorContains(t, err, test.wantError)
			assert.Zero(t, provider.created)
		})
	}
}

func TestPullRequestActionsGuideUnavailableCreationSafety(t *testing.T) {
	actions := pullRequestActions(&model.PullRequestPreparation{
		Permissions:      model.ProviderSignal{State: "unavailable"},
		ExistingRequests: model.ProviderSignal{State: "unavailable"},
	})

	assert.ElementsMatch(t, []string{"authorize", "inspect_existing_requests"}, actionKinds(actions))
}

func TestQueueFiltersRequestedReviewers(t *testing.T) {
	service := NewService(&fakeProvider{
		user: &model.User{Login: "stephen"},
		prs: []*model.PullRequest{
			{Number: 1, RequestedReviewers: []model.User{{Login: "stephen"}}},
			{Number: 2},
		},
	})

	prs, err := service.Queue(context.Background(), model.RepositoryRef{Owner: "Raithlin", Name: "gha"})
	require.NoError(t, err)
	require.Len(t, prs, 1)
	assert.Equal(t, 1, prs[0].Number)
}

func TestAssignedExcludesIssues(t *testing.T) {
	service := NewService(&fakeProvider{
		user: &model.User{Login: "stephen"},
		issues: []*model.Issue{
			{Number: 1, Title: "Issue"},
			{Number: 2, Title: "Pull request", PullRequest: &model.PullRequestReference{}},
		},
	})

	prs, err := service.Assigned(context.Background(), model.RepositoryRef{Owner: "Raithlin", Name: "gha"})
	require.NoError(t, err)
	require.Len(t, prs, 1)
	assert.Equal(t, 2, prs[0].Number)
}

func TestInspectBuildsDecisionReadySummary(t *testing.T) {
	mergeable := false
	service := NewService(&fakeProvider{
		pr: &model.PullRequest{
			Number: 42, Mergeable: &mergeable, ChangedFiles: 42,
			Head:               model.BranchRef{SHA: "abc123"},
			RequestedReviewers: []model.User{{Login: "alice"}},
		},
		checks:  []*model.CheckRun{{Name: "test", Status: "completed", Conclusion: "success"}},
		threads: []*model.ReviewThread{{IsResolved: true}, {IsResolved: false}},
		reviews: []*model.Review{
			{User: model.User{Login: "bob"}, State: "APPROVED", SubmittedAt: "2026-09-12T10:00:00Z"},
			{User: model.User{Login: "carol"}, State: "CHANGES_REQUESTED", SubmittedAt: "2026-09-12T11:00:00Z"},
			{User: model.User{Login: "bob"}, State: "COMMENTED", SubmittedAt: "2026-09-12T12:00:00Z"},
		},
	})

	summary, err := service.Inspect(context.Background(), model.RepositoryRef{Owner: "Raithlin", Name: "gha"}, 42)

	require.NoError(t, err)
	assert.Equal(t, model.ReviewSummarySchemaVersion, summary.SchemaVersion)
	assert.Equal(t, "success", summary.Readiness.CIStatus)
	assert.Equal(t, "unresolved", summary.Readiness.ReviewThreadsState)
	assert.Empty(t, summary.Readiness.ApprovedBy, "Bob's latest review is a comment, not an approval")
	require.Len(t, summary.Readiness.ChangesRequestedBy, 1)
	assert.Equal(t, "carol", summary.Readiness.ChangesRequestedBy[0].Login)
	require.Len(t, summary.Readiness.PendingReviewers, 1)
	assert.Equal(t, "alice", summary.Readiness.PendingReviewers[0].Login)
	assert.ElementsMatch(t, []string{"merge_conflict", "large_change", "changes_requested", "unresolved_review_threads"}, riskKinds(summary.RiskSignals))
	assert.ElementsMatch(t, []string{"resolve_merge_conflicts", "address_requested_changes", "wait_for_review", "resolve_review_threads"}, actionKinds(summary.RecommendedActions))
}

func TestSummarizeCheckRuns(t *testing.T) {
	tests := []struct {
		name   string
		checks []*model.CheckRun
		want   string
	}{
		{name: "no checks", want: "none"},
		{name: "successful checks", checks: []*model.CheckRun{{Status: "completed", Conclusion: "success"}, {Status: "completed", Conclusion: "skipped"}}, want: "success"},
		{name: "running check", checks: []*model.CheckRun{{Status: "in_progress"}}, want: "pending"},
		{name: "failed check", checks: []*model.CheckRun{{Status: "completed", Conclusion: "failure"}}, want: "failure"},
		{name: "failed and pending checks", checks: []*model.CheckRun{{Status: "completed", Conclusion: "failure"}, {Status: "queued"}}, want: "failure"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, summarizeCheckRuns(test.checks))
		})
	}
}

func TestListMatchingFindsFilteredPullRequestOnLaterPage(t *testing.T) {
	firstPage := make([]*model.PullRequest, 100)
	for i := range firstPage {
		firstPage[i] = &model.PullRequest{Number: i + 1, User: model.User{Login: "other"}}
	}
	provider := &fakeProvider{prPages: map[int][]*model.PullRequest{
		1: firstPage,
		2: {{Number: 101, User: model.User{Login: "alice"}}},
	}}
	service := NewService(provider)

	prs, err := service.ListMatching(context.Background(), model.RepositoryRef{Owner: "Raithlin", Name: "gha"}, interfaces.ListPRsOptions{State: "open", PerPage: 100}, 1, func(pr *model.PullRequest) bool {
		return pr.User.Login == "alice"
	})

	require.NoError(t, err)
	require.Len(t, prs, 1)
	assert.Equal(t, 101, prs[0].Number)
	require.Len(t, provider.prOptions, 2)
	assert.Equal(t, 2, provider.prOptions[1].Page)
}

func TestAssignedLimitedFindsPullRequestOnLaterIssuePage(t *testing.T) {
	firstPage := make([]*model.Issue, 100)
	for i := range firstPage {
		firstPage[i] = &model.Issue{Number: i + 1}
	}
	provider := &fakeProvider{
		user: &model.User{Login: "stephen"},
		issuePages: map[int][]*model.Issue{
			1: firstPage,
			2: {{Number: 101, PullRequest: &model.PullRequestReference{}}},
		},
	}
	service := NewService(provider)

	prs, err := service.AssignedLimited(context.Background(), model.RepositoryRef{Owner: "Raithlin", Name: "gha"}, 1)

	require.NoError(t, err)
	require.Len(t, prs, 1)
	assert.Equal(t, 101, prs[0].Number)
	require.Len(t, provider.issueOptions, 2)
	assert.Equal(t, 2, provider.issueOptions[1].Page)
}

func TestQueueAndMineLimitedFindMatchesOnLaterPages(t *testing.T) {
	firstPage := make([]*model.PullRequest, 100)
	for i := range firstPage {
		firstPage[i] = &model.PullRequest{Number: i + 1, User: model.User{Login: "other"}}
	}

	tests := []struct {
		name    string
		laterPR *model.PullRequest
		list    func(*Service, model.RepositoryRef) ([]*model.PullRequest, error)
	}{
		{
			name:    "queue",
			laterPR: &model.PullRequest{Number: 101, RequestedReviewers: []model.User{{Login: "stephen"}}},
			list: func(service *Service, repository model.RepositoryRef) ([]*model.PullRequest, error) {
				return service.QueueLimited(context.Background(), repository, 1)
			},
		},
		{
			name:    "mine",
			laterPR: &model.PullRequest{Number: 101, User: model.User{Login: "stephen"}},
			list: func(service *Service, repository model.RepositoryRef) ([]*model.PullRequest, error) {
				return service.MineLimited(context.Background(), repository, 1)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := &fakeProvider{
				user: &model.User{Login: "stephen"},
				prPages: map[int][]*model.PullRequest{
					1: firstPage,
					2: {test.laterPR},
				},
			}
			prs, err := test.list(NewService(provider), model.RepositoryRef{Owner: "Raithlin", Name: "gha"})

			require.NoError(t, err)
			require.Len(t, prs, 1)
			assert.Equal(t, 101, prs[0].Number)
			require.Len(t, provider.prOptions, 2)
		})
	}
}

func TestInspectLeavesCIUnavailableWhenCheckLookupFails(t *testing.T) {
	service := NewService(&fakeProvider{
		pr:       &model.PullRequest{Number: 42, Head: model.BranchRef{SHA: "abc123"}},
		checkErr: errors.New("checks permission denied"),
	})

	summary, err := service.Inspect(context.Background(), model.RepositoryRef{Owner: "Raithlin", Name: "gha"}, 42)

	require.NoError(t, err)
	assert.Equal(t, "unavailable", summary.Readiness.CIStatus)
	assert.ElementsMatch(t, []string{"check_ci"}, actionKinds(summary.RecommendedActions))
}

func TestInspectLeavesReviewThreadsUnavailableWhenLookupFails(t *testing.T) {
	service := NewService(&fakeProvider{
		pr:        &model.PullRequest{Number: 42},
		threadErr: errors.New("review threads permission denied"),
	})

	summary, err := service.Inspect(context.Background(), model.RepositoryRef{Owner: "Raithlin", Name: "gha"}, 42)

	require.NoError(t, err)
	assert.Equal(t, "unavailable", summary.Readiness.ReviewThreadsState)
}

func TestReleaseNotesIncludesOnlyMergedPullRequestsInWindow(t *testing.T) {
	since := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	service := NewService(&fakeProvider{prs: []*model.PullRequest{
		{Number: 3, Title: "Later", User: model.User{Login: "alice"}, MergedAt: "2026-09-03T00:00:00Z"},
		{Number: 2, Title: "Earlier", User: model.User{Login: "bob"}, MergedAt: "2026-09-01T00:00:00Z"},
		{Number: 1, Title: "Closed without merging", User: model.User{Login: "carol"}},
		{Number: 4, Title: "Before window", User: model.User{Login: "dave"}, MergedAt: "2026-08-31T23:59:59Z"},
	}})

	notes, err := service.ReleaseNotes(context.Background(), model.RepositoryRef{Owner: "Raithlin", Name: "gha"}, since, 100)

	require.NoError(t, err)
	assert.Equal(t, model.ReleaseNotesSchemaVersion, notes.SchemaVersion)
	assert.Equal(t, "2026-09-01T00:00:00Z", notes.Since)
	assert.Equal(t, 100, notes.Limit)
	assert.False(t, notes.Truncated)
	require.Len(t, notes.PullRequests, 2)
	assert.Equal(t, 2, notes.PullRequests[0].Number, "notes are ordered by merge time")
	assert.Equal(t, 3, notes.PullRequests[1].Number)
	assert.Equal(t, []model.User{{Login: "bob"}, {Login: "alice"}}, notes.Contributors)
}

func TestReleaseNotesLimitsMatchingPullRequests(t *testing.T) {
	service := NewService(&fakeProvider{prs: []*model.PullRequest{
		{Number: 1, MergedAt: "2026-09-01T00:00:00Z"},
		{Number: 2, MergedAt: "2026-09-02T00:00:00Z"},
	}})

	notes, err := service.ReleaseNotes(context.Background(), model.RepositoryRef{Owner: "Raithlin", Name: "gha"}, time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC), 1)

	require.NoError(t, err)
	require.Len(t, notes.PullRequests, 1)
	assert.Equal(t, 1, notes.PullRequests[0].Number)
	assert.Equal(t, 1, notes.Limit)
	assert.True(t, notes.Truncated)
}

func TestSummarizeReviewThreads(t *testing.T) {
	tests := []struct {
		name    string
		threads []*model.ReviewThread
		want    string
	}{
		{name: "no threads", want: "none"},
		{name: "resolved threads", threads: []*model.ReviewThread{{IsResolved: true}}, want: "resolved"},
		{name: "unresolved thread", threads: []*model.ReviewThread{{IsResolved: true}, {IsResolved: false}}, want: "unresolved"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, summarizeReviewThreads(test.threads))
		})
	}
}

func TestServiceWrapsProviderErrorsAndHandlesNilResponses(t *testing.T) {
	repository := model.RepositoryRef{Owner: "acme", Name: "project"}
	_, err := NewService(&fakeProvider{prsErr: errors.New("offline")}).List(context.Background(), repository, interfaces.ListPRsOptions{})
	assert.ErrorContains(t, err, "list pull requests for acme/project: offline")
	_, err = NewService(&fakeProvider{prErr: errors.New("missing")}).Get(context.Background(), repository, 1)
	assert.ErrorContains(t, err, "get pull request acme/project#1: missing")
	_, err = NewService(&fakeProvider{userErr: errors.New("unauthorized")}).AuthenticatedUser(context.Background())
	assert.ErrorContains(t, err, "get authenticated user: unauthorized")
	_, err = NewService(&fakeProvider{issuesErr: errors.New("offline")}).Assigned(context.Background(), repository)
	assert.ErrorContains(t, err, "get authenticated user")
	_, err = NewService(&fakeProvider{user: &model.User{Login: "me"}, issuesErr: errors.New("offline")}).Assigned(context.Background(), repository)
	assert.ErrorContains(t, err, "list assigned pull requests: offline")
	_, err = NewService(&fakeProvider{releasesErr: errors.New("offline")}).ListPublishedReleases(context.Background(), repository, 1)
	assert.ErrorContains(t, err, "list releases for acme/project: offline")
	_, err = (*Service)(nil).ListPublishedReleases(context.Background(), repository, 1)
	assert.ErrorContains(t, err, "not configured")
	_, err = NewService(&fakeProvider{}).ListPublishedReleases(context.Background(), repository, 0)
	assert.ErrorContains(t, err, "limit must be between 1 and 100")

	_, err = NewService(&fakeProvider{}).Inspect(context.Background(), repository, 1)
	assert.ErrorContains(t, err, "provider returned no pull request")
}

func TestReviewHelperBranches(t *testing.T) {
	assert.False(t, hasRequestedReviewer(&model.PullRequest{}, "alice"))
	assert.Equal(t, &model.PullRequest{ID: 1, Number: 2, Title: "Issue", State: "open"}, pullRequestFromIssue(&model.Issue{ID: 1, Number: 2, Title: "Issue", State: "open"}))
	assert.Equal(t, "unresolved", summarizeReviewThreads([]*model.ReviewThread{nil}))
	assert.False(t, func() bool { _, ok := parseMergedAt(&model.PullRequest{MergedAt: "not-a-time"}); return ok }())
	assert.False(t, func() bool { _, ok := parseMergedAt(nil); return ok }())
	assert.Equal(t, "success", summarizeCheckRuns([]*model.CheckRun{{Status: "completed", Conclusion: "neutral"}}))

	summary := summarize(&model.PullRequest{ChangedFiles: 25}, nil, "pending", "none")
	assert.ElementsMatch(t, []string{"large_change"}, riskKinds(summary.RiskSignals))
	assert.ElementsMatch(t, []string{"wait_for_ci"}, actionKinds(summary.RecommendedActions))
	summary = summarize(&model.PullRequest{ChangedFiles: 75}, nil, "failure", "unresolved")
	assert.ElementsMatch(t, []string{"large_change", "ci_failure", "unresolved_review_threads"}, riskKinds(summary.RiskSignals))
}

func TestMineReturnsAuthenticatedUsersPullRequests(t *testing.T) {
	service := NewService(&fakeProvider{user: &model.User{Login: "alice"}, prs: []*model.PullRequest{{Number: 1, User: model.User{Login: "alice"}}, {Number: 2, User: model.User{Login: "bob"}}}})
	prs, err := service.Mine(context.Background(), model.RepositoryRef{Owner: "acme", Name: "project"})
	require.NoError(t, err)
	require.Len(t, prs, 1)
	assert.Equal(t, 1, prs[0].Number)
}

func TestPullRequestPreparationPreservesAmbiguousProviderStates(t *testing.T) {
	repository := model.RepositoryRef{Owner: "acme", Name: "project"}
	_, err := (*Service)(nil).PreparePullRequest(context.Background(), PreparePullRequestInput{Repository: repository})
	assert.ErrorContains(t, err, "not configured")
	_, err = NewService(&fakeProvider{repository: &model.Repository{}}).PreparePullRequest(context.Background(), PreparePullRequestInput{Repository: repository, Head: "feature"})
	assert.ErrorContains(t, err, "did not report a default branch")

	same, err := NewService(&fakeProvider{repository: &model.Repository{DefaultBranch: "main"}}).PreparePullRequest(context.Background(), PreparePullRequestInput{Repository: repository, Head: "main"})
	require.NoError(t, err)
	assert.Equal(t, "not_applicable", same.Comparison.State)
	assert.ElementsMatch(t, []string{"same_branch"}, riskKinds(same.RiskSignals))

	provider := &fakeProvider{repository: &model.Repository{DefaultBranch: "main"}, comparison: nil, prsErr: errors.New("forbidden")}
	preparation, err := NewService(provider).PreparePullRequest(context.Background(), PreparePullRequestInput{Repository: repository, Head: "feature"})
	require.NoError(t, err)
	assert.Equal(t, "unavailable", preparation.Comparison.State)
	assert.Equal(t, "unavailable", preparation.ExistingRequests.State)
	assert.ElementsMatch(t, []string{"comparison_unavailable", "existing_requests_unavailable"}, riskKinds(preparation.RiskSignals))
	assert.ElementsMatch(t, []string{"inspect_branches", "inspect_existing_requests", "authorize"}, actionKinds(preparation.RecommendedActions))
}

func TestCreatePreparedPullRequestExercisesValidationAndProviderFailure(t *testing.T) {
	yes := true
	base := &model.PullRequestPreparation{Repository: model.RepositoryRef{Owner: "acme", Name: "project"}, Head: "feature", Base: "main", Comparison: model.BranchComparison{State: "ahead", AheadBy: 1}, ExistingRequests: model.ProviderSignal{State: "available"}, Permissions: model.ProviderSignal{State: "available"}, CanPush: &yes}
	provider := &fakeProvider{}
	created, err := NewService(provider).CreatePreparedPullRequest(context.Background(), base)
	require.NoError(t, err)
	assert.Equal(t, 1, created.Number)
	_, err = NewService(&fakeProvider{createErr: errors.New("write rejected")}).CreatePreparedPullRequest(context.Background(), base)
	assert.ErrorContains(t, err, "create pull request for acme/project: write rejected")
	_, err = NewService(provider).CreatePreparedPullRequest(context.Background(), nil)
	assert.ErrorContains(t, err, "preparation is required")
}

func TestInspectToleratesNullReviewEntries(t *testing.T) {
	service := NewService(&fakeProvider{pr: &model.PullRequest{Number: 1}, reviews: []*model.Review{nil}})
	summary, err := service.Inspect(context.Background(), model.RepositoryRef{Owner: "acme", Name: "project"}, 1)
	require.NoError(t, err)
	require.Empty(t, summary.Readiness.ApprovedBy)
	require.Empty(t, summary.Readiness.ChangesRequestedBy)
}

func TestReviewWorkflowsIgnoreNullListEntries(t *testing.T) {
	repository := model.RepositoryRef{Owner: "acme", Name: "project"}
	provider := &fakeProvider{user: &model.User{Login: "alice"}, prs: []*model.PullRequest{nil, {Number: 1, User: model.User{Login: "alice"}, MergedAt: "2026-09-02T00:00:00Z"}}, issues: []*model.Issue{nil, {Number: 2, PullRequest: &model.PullRequestReference{}}}}
	service := NewService(provider)
	prs, err := service.Mine(context.Background(), repository)
	require.NoError(t, err)
	require.Len(t, prs, 1)
	assigned, err := service.Assigned(context.Background(), repository)
	require.NoError(t, err)
	require.Len(t, assigned, 1)
	notes, err := service.ReleaseNotes(context.Background(), repository, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), 10)
	require.NoError(t, err)
	require.Len(t, notes.PullRequests, 1)
	assert.Equal(t, "unavailable", summarizeCheckRuns([]*model.CheckRun{nil, {Status: "completed", Conclusion: "failure"}}))
}

func TestReviewFailureAndSafetyBranchesStayExplicit(t *testing.T) {
	repository := model.RepositoryRef{Owner: "acme", Name: "project"}
	provider := &fakeProvider{
		repository: &model.Repository{DefaultBranch: "main", Permissions: &model.RepositoryPermissions{Push: true}},
		comparison: &model.BranchComparison{State: "ahead", AheadBy: 1},
		prs:        []*model.PullRequest{{Number: 1}},
	}
	preparation, err := NewService(provider).PreparePullRequest(context.Background(), PreparePullRequestInput{Repository: repository, Head: "feature"})
	require.NoError(t, err)
	assert.Contains(t, actionKinds(preparation.RecommendedActions), "review_existing")
	assert.ErrorContains(t, validatePullRequestPreparation(preparation), "already exists")

	noChanges, err := NewService(&fakeProvider{repository: &model.Repository{DefaultBranch: "main", Permissions: &model.RepositoryPermissions{Push: true}}, comparison: &model.BranchComparison{State: "ahead"}}).PreparePullRequest(context.Background(), PreparePullRequestInput{Repository: repository, Head: "feature"})
	require.NoError(t, err)
	assert.Contains(t, actionKinds(noChanges.RecommendedActions), "add_commits")

	assert.ErrorContains(t, validatePullRequestPreparation(&model.PullRequestPreparation{Comparison: model.BranchComparison{State: "unavailable"}}), "comparison is unavailable")
	denied := false
	actions := pullRequestActions(&model.PullRequestPreparation{ExistingRequests: model.ProviderSignal{State: "available"}, Permissions: model.ProviderSignal{State: "available"}, CanPush: &denied})
	assert.Contains(t, actionKinds(actions), "request_permission")

	_, err = NewService(&fakeProvider{prsErr: errors.New("offline")}).ReleaseNotes(context.Background(), repository, time.Now().UTC(), 1)
	assert.ErrorContains(t, err, "list merged pull requests")
	_, err = NewService(&fakeProvider{releases: []*model.Release{}}).ListPublishedReleases(context.Background(), repository, 100)
	require.NoError(t, err)

	_, err = NewService(&fakeProvider{pr: &model.PullRequest{}, reviewsErr: errors.New("reviews unavailable")}).Inspect(context.Background(), repository, 1)
	assert.ErrorContains(t, err, "list reviews")
	_, err = NewService(&fakeProvider{userErr: errors.New("unauthorized")}).QueueLimited(context.Background(), repository, 1)
	assert.ErrorContains(t, err, "get authenticated user")
	_, err = NewService(&fakeProvider{userErr: errors.New("unauthorized")}).MineLimited(context.Background(), repository, 1)
	assert.ErrorContains(t, err, "get authenticated user")

	summary := summarize(&model.PullRequest{}, []*model.Review{{User: model.User{Login: "alice"}, State: "APPROVED"}}, "none", "none")
	assert.Equal(t, []model.User{{Login: "alice"}}, summary.Readiness.ApprovedBy)
}

func riskKinds(signals []model.RiskSignal) []string {
	kinds := make([]string, 0, len(signals))
	for _, signal := range signals {
		kinds = append(kinds, signal.Kind)
	}
	return kinds
}

func actionKinds(actions []model.RecommendedAction) []string {
	kinds := make([]string, 0, len(actions))
	for _, action := range actions {
		kinds = append(kinds, action.Action)
	}
	return kinds
}
