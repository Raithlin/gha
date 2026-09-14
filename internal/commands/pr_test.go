package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/interfaces"
	"github.com/raithlin/gha/internal/review"
	"github.com/raithlin/gha/pkg/model"
)

func TestPRPrepareResolvesBaseAndReportsComparison(t *testing.T) {
	provider := &prProvider{repository: &model.Repository{DefaultBranch: "main", Permissions: &model.RepositoryPermissions{Push: true}}, comparison: &model.BranchComparison{State: "ahead", AheadBy: 2}}
	command := newPRCmd(review.NewService(provider), git.NewRepositoryResolver("acme/project"))
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"prepare", "--title", "Improve reviews", "--head", "feature", "--format", "json"})

	require.NoError(t, command.Execute())
	var preparation model.PullRequestPreparation
	require.NoError(t, json.Unmarshal(output.Bytes(), &preparation))
	assert.Equal(t, model.PullRequestPreparationSchemaVersion, preparation.SchemaVersion)
	assert.Equal(t, "main", preparation.Base)
	assert.Equal(t, "feature", preparation.Head)
	assert.Equal(t, "planned", preparation.Creation)
	assert.Equal(t, 2, preparation.Comparison.AheadBy)
	assert.Empty(t, provider.created)
}

func TestPRCreateRequiresConfirmationAndDryRunDoesNotCreate(t *testing.T) {
	provider := &prProvider{repository: &model.Repository{DefaultBranch: "main"}, comparison: &model.BranchComparison{State: "ahead", AheadBy: 1}}
	command := newPRCmd(review.NewService(provider), git.NewRepositoryResolver("acme/project"))
	command.SetArgs([]string{"create", "--title", "Improve reviews", "--head", "feature"})
	err := command.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "--confirm")
	assert.Empty(t, provider.created)

	command = newPRCmd(review.NewService(provider), git.NewRepositoryResolver("acme/project"))
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"create", "--title", "Improve reviews", "--head", "feature", "--dry-run", "--format", "json"})
	require.NoError(t, command.Execute())
	var preparation model.PullRequestPreparation
	require.NoError(t, json.Unmarshal(output.Bytes(), &preparation))
	assert.True(t, preparation.DryRun)
	assert.Equal(t, "planned", preparation.Creation)
	assert.Empty(t, provider.created)
}

func TestPRCreateRejectsExistingPullRequestBeforeWriting(t *testing.T) {
	provider := &prProvider{repository: &model.Repository{DefaultBranch: "main"}, comparison: &model.BranchComparison{State: "ahead", AheadBy: 1}, pullRequests: []*model.PullRequest{{Number: 7, Title: "Existing", State: "open"}}}
	command := newPRCmd(review.NewService(provider), git.NewRepositoryResolver("acme/project"))
	command.SetArgs([]string{"create", "--title", "Improve reviews", "--head", "feature", "--confirm"})

	err := command.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "already exists")
	assert.Empty(t, provider.created)
}

func TestPRCreateWritesOnlyAfterConfirmedSafePreflight(t *testing.T) {
	provider := &prProvider{repository: &model.Repository{DefaultBranch: "main"}, comparison: &model.BranchComparison{State: "ahead", AheadBy: 1}}
	command := newPRCmd(review.NewService(provider), git.NewRepositoryResolver("acme/project"))
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"create", "--title", "Improve reviews", "--head", "feature", "--confirm", "--format", "json"})

	require.NoError(t, command.Execute())
	var preparation model.PullRequestPreparation
	require.NoError(t, json.Unmarshal(output.Bytes(), &preparation))
	assert.Equal(t, "completed", preparation.Creation)
	require.Len(t, provider.created, 1)
	assert.Equal(t, "feature", provider.created[0].Head)
	assert.Equal(t, "main", provider.created[0].Base)
	assert.Equal(t, 8, preparation.CreatedPullRequest.Number)
}

type prProvider struct {
	repository   *model.Repository
	comparison   *model.BranchComparison
	pullRequests []*model.PullRequest
	created      []*model.PullRequestInput
}

func (p *prProvider) GetAuthenticatedUser(context.Context) (*model.User, error) {
	return &model.User{Login: "stephen"}, nil
}
func (p *prProvider) ListRepositories(context.Context) ([]*model.Repository, error) { return nil, nil }
func (p *prProvider) GetRepository(context.Context, string, string) (*model.Repository, error) {
	return p.repository, nil
}
func (p *prProvider) ListPullRequests(context.Context, string, string, interfaces.ListPRsOptions) ([]*model.PullRequest, error) {
	return p.pullRequests, nil
}
func (p *prProvider) GetPullRequest(context.Context, string, string, int) (*model.PullRequest, error) {
	return nil, nil
}
func (p *prProvider) CreatePullRequest(_ context.Context, _ string, _ string, input *model.PullRequestInput) (*model.PullRequest, error) {
	p.created = append(p.created, input)
	return &model.PullRequest{Number: 8, Title: input.Title, State: "open", Head: model.BranchRef{Ref: input.Head}, Base: model.BranchRef{Ref: input.Base}}, nil
}
func (p *prProvider) UpdatePullRequest(context.Context, string, string, int, *model.PullRequestInput) (*model.PullRequest, error) {
	return nil, nil
}
func (p *prProvider) ListIssues(context.Context, string, string, interfaces.ListIssuesOptions) ([]*model.Issue, error) {
	return nil, nil
}
func (p *prProvider) GetIssue(context.Context, string, string, int) (*model.Issue, error) {
	return nil, nil
}
func (p *prProvider) AddComment(context.Context, string, string, int, string) (*model.Comment, error) {
	return nil, nil
}
func (p *prProvider) ListReviews(context.Context, string, string, int) ([]*model.Review, error) {
	return nil, nil
}
func (p *prProvider) ListCheckRuns(context.Context, string, string, string) ([]*model.CheckRun, error) {
	return nil, nil
}
func (p *prProvider) ListReviewThreads(context.Context, string, string, int) ([]*model.ReviewThread, error) {
	return nil, nil
}
func (p *prProvider) SubmitReview(context.Context, string, string, int, *model.ReviewInput) (*model.Review, error) {
	return nil, nil
}
func (p *prProvider) CompareBranches(context.Context, string, string, string, string) (*model.BranchComparison, error) {
	return p.comparison, nil
}
