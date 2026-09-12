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
	user   *model.User
	prs    []*model.PullRequest
	issues []*model.Issue
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

func (p *fakeProvider) ListPullRequests(context.Context, string, string, interfaces.ListPRsOptions) ([]*model.PullRequest, error) {
	return p.prs, nil
}

func (p *fakeProvider) GetPullRequest(context.Context, string, string, int) (*model.PullRequest, error) {
	return nil, errors.New("not implemented")
}

func (p *fakeProvider) CreatePullRequest(context.Context, string, string, *model.PullRequestInput) (*model.PullRequest, error) {
	return nil, errors.New("not implemented")
}

func (p *fakeProvider) UpdatePullRequest(context.Context, string, string, int, *model.PullRequestInput) (*model.PullRequest, error) {
	return nil, errors.New("not implemented")
}

func (p *fakeProvider) ListIssues(context.Context, string, string, interfaces.ListIssuesOptions) ([]*model.Issue, error) {
	return p.issues, nil
}

func (p *fakeProvider) GetIssue(context.Context, string, string, int) (*model.Issue, error) {
	return nil, errors.New("not implemented")
}

func (p *fakeProvider) AddComment(context.Context, string, string, int, string) (*model.Comment, error) {
	return nil, errors.New("not implemented")
}

func (p *fakeProvider) ListReviews(context.Context, string, string, int) ([]*model.Review, error) {
	return nil, errors.New("not implemented")
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
