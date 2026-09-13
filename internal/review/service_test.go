package review

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/internal/interfaces"
	"github.com/raithlin/gha/pkg/model"
)

type fakeProvider struct {
	user         *model.User
	pr           *model.PullRequest
	prs          []*model.PullRequest
	issues       []*model.Issue
	reviews      []*model.Review
	checks       []*model.CheckRun
	checkErr     error
	prPages      map[int][]*model.PullRequest
	issuePages   map[int][]*model.Issue
	prOptions    []interfaces.ListPRsOptions
	issueOptions []interfaces.ListIssuesOptions
}

func (p *fakeProvider) GetAuthenticatedUser(context.Context) (*model.User, error) {
	return p.user, nil
}

func (p *fakeProvider) ListRepositories(context.Context) ([]*model.Repository, error) {
	return nil, errors.New("not implemented")
}

func (p *fakeProvider) GetRepository(context.Context, string, string) (*model.Repository, error) {
	return nil, errors.New("not implemented")
}

func (p *fakeProvider) ListPullRequests(_ context.Context, _ string, _ string, options interfaces.ListPRsOptions) ([]*model.PullRequest, error) {
	p.prOptions = append(p.prOptions, options)
	if p.prPages != nil {
		return p.prPages[options.Page], nil
	}
	return p.prs, nil
}

func (p *fakeProvider) GetPullRequest(context.Context, string, string, int) (*model.PullRequest, error) {
	return p.pr, nil
}

func (p *fakeProvider) CreatePullRequest(context.Context, string, string, *model.PullRequestInput) (*model.PullRequest, error) {
	return nil, errors.New("not implemented")
}

func (p *fakeProvider) UpdatePullRequest(context.Context, string, string, int, *model.PullRequestInput) (*model.PullRequest, error) {
	return nil, errors.New("not implemented")
}

func (p *fakeProvider) ListIssues(_ context.Context, _ string, _ string, options interfaces.ListIssuesOptions) ([]*model.Issue, error) {
	p.issueOptions = append(p.issueOptions, options)
	if p.issuePages != nil {
		return p.issuePages[options.Page], nil
	}
	return p.issues, nil
}

func (p *fakeProvider) GetIssue(context.Context, string, string, int) (*model.Issue, error) {
	return nil, errors.New("not implemented")
}

func (p *fakeProvider) AddComment(context.Context, string, string, int, string) (*model.Comment, error) {
	return nil, errors.New("not implemented")
}

func (p *fakeProvider) ListReviews(context.Context, string, string, int) ([]*model.Review, error) {
	return p.reviews, nil
}

func (p *fakeProvider) ListCheckRuns(context.Context, string, string, string) ([]*model.CheckRun, error) {
	return p.checks, p.checkErr
}

func (p *fakeProvider) SubmitReview(context.Context, string, string, int, *model.ReviewInput) (*model.Review, error) {
	return nil, errors.New("not implemented")
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
		checks: []*model.CheckRun{{Name: "test", Status: "completed", Conclusion: "success"}},
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
	assert.Equal(t, "unavailable", summary.Readiness.ReviewThreadsState)
	assert.Empty(t, summary.Readiness.ApprovedBy, "Bob's latest review is a comment, not an approval")
	require.Len(t, summary.Readiness.ChangesRequestedBy, 1)
	assert.Equal(t, "carol", summary.Readiness.ChangesRequestedBy[0].Login)
	require.Len(t, summary.Readiness.PendingReviewers, 1)
	assert.Equal(t, "alice", summary.Readiness.PendingReviewers[0].Login)
	assert.ElementsMatch(t, []string{"merge_conflict", "large_change", "changes_requested"}, riskKinds(summary.RiskSignals))
	assert.ElementsMatch(t, []string{"resolve_merge_conflicts", "address_requested_changes", "wait_for_review"}, actionKinds(summary.RecommendedActions))
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
