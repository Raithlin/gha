package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
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

func TestDraftBodySummarizesCommitDescriptionsWithoutListingFiles(t *testing.T) {
	body := draftBody(model.PullRequestDraft{State: "available", CommitSubjects: []string{"Add review summaries", "Cover empty reviewer state"}, ChangedFiles: []string{"internal/review/service.go"}})
	assert.Contains(t, body, "Add review summaries")
	assert.Contains(t, body, "Cover empty reviewer state")
	assert.NotContains(t, body, "Changed files")
	assert.NotContains(t, body, "service.go")
	assert.Contains(t, draftBody(model.PullRequestDraft{State: "available"}), "No commits found")
	assert.Contains(t, draftBody(model.PullRequestDraft{State: "unavailable"}), "Review the commits")
}

func TestPreparePullRequestBuildsDraftFromLocalCommits(t *testing.T) {
	repo := t.TempDir()
	runPRGit(t, repo, "init", "--quiet")
	runPRGit(t, repo, "config", "user.email", "test@example.com")
	runPRGit(t, repo, "config", "user.name", "Test")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "base.txt"), []byte("base"), 0600))
	runPRGit(t, repo, "add", ".")
	runPRGit(t, repo, "commit", "--quiet", "-m", "base")
	runPRGit(t, repo, "branch", "-M", "main")
	runPRGit(t, repo, "checkout", "-b", "feature")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "feature.txt"), []byte("feature"), 0600))
	runPRGit(t, repo, "add", ".")
	runPRGit(t, repo, "commit", "--quiet", "-m", "Add feature summary")

	command := &cobra.Command{}
	command.SetContext(context.Background())
	provider := &prProvider{repository: &model.Repository{DefaultBranch: "main", Permissions: &model.RepositoryPermissions{Push: true}}, comparison: &model.BranchComparison{State: "ahead", AheadBy: 1}}
	preparation, _, err := preparePullRequest(command, review.NewService(provider), git.NewRepositoryResolver("acme/project"), &prOptions{format: "json", repository: "acme/project", head: "feature", path: repo}, false)
	require.NoError(t, err)
	assert.Equal(t, "Add feature summary", preparation.Title)
	assert.Equal(t, "available", preparation.Draft.State)
	assert.Equal(t, []string{"Add feature summary"}, preparation.Draft.CommitSubjects)
	assert.Contains(t, preparation.Body, "Add feature summary")
	assert.NotContains(t, preparation.Body, "Changed files")
}

func runPRGit(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, output)
}

func TestPRCreateRunsByDefaultAndDryRunDoesNotCreate(t *testing.T) {
	provider := &prProvider{repository: &model.Repository{DefaultBranch: "main", Permissions: &model.RepositoryPermissions{Push: true}}, comparison: &model.BranchComparison{State: "ahead", AheadBy: 1}}
	command := newPRCmd(review.NewService(provider), git.NewRepositoryResolver("acme/project"))
	command.SetArgs([]string{"create", "--title", "Improve reviews", "--head", "feature"})
	err := command.Execute()
	require.NoError(t, err)
	assert.Len(t, provider.created, 1)

	command = newPRCmd(review.NewService(provider), git.NewRepositoryResolver("acme/project"))
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"create", "--title", "Improve reviews", "--head", "feature", "--dry-run", "--format", "json"})
	require.NoError(t, command.Execute())
	var preparation model.PullRequestPreparation
	require.NoError(t, json.Unmarshal(output.Bytes(), &preparation))
	assert.True(t, preparation.DryRun)
	assert.Equal(t, "planned", preparation.Creation)
	assert.Len(t, provider.created, 1)
}

func TestPRCreateRejectsExistingPullRequestBeforeWriting(t *testing.T) {
	provider := &prProvider{repository: &model.Repository{DefaultBranch: "main", Permissions: &model.RepositoryPermissions{Push: true}}, comparison: &model.BranchComparison{State: "ahead", AheadBy: 1}, pullRequests: []*model.PullRequest{{Number: 7, Title: "Existing", State: "open"}}}
	command := newPRCmd(review.NewService(provider), git.NewRepositoryResolver("acme/project"))
	command.SetArgs([]string{"create", "--title", "Improve reviews", "--head", "feature"})

	err := command.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "already exists")
	assert.Empty(t, provider.created)
}

func TestPRCreateWritesAfterSafePreflight(t *testing.T) {
	provider := &prProvider{repository: &model.Repository{DefaultBranch: "main", Permissions: &model.RepositoryPermissions{Push: true}}, comparison: &model.BranchComparison{State: "ahead", AheadBy: 1}}
	command := newPRCmd(review.NewService(provider), git.NewRepositoryResolver("acme/project"))
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"create", "--title", "Improve reviews", "--head", "feature", "--format", "json"})

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
	repository      *model.Repository
	comparison      *model.BranchComparison
	pullRequests    []*model.PullRequest
	created         []*model.PullRequestInput
	userErr         error
	pullRequestsErr error
	issuesErr       error
}

func (p *prProvider) GetAuthenticatedUser(context.Context) (*model.User, error) {
	return &model.User{Login: "stephen"}, p.userErr
}
func (p *prProvider) ListRepositories(context.Context) ([]*model.Repository, error) { return nil, nil }
func (p *prProvider) GetRepository(context.Context, string, string) (*model.Repository, error) {
	return p.repository, nil
}
func (p *prProvider) ListReleases(context.Context, string, string, interfaces.ListReleasesOptions) ([]*model.Release, error) {
	return nil, nil
}
func (p *prProvider) ListPullRequests(context.Context, string, string, interfaces.ListPRsOptions) ([]*model.PullRequest, error) {
	return p.pullRequests, p.pullRequestsErr
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
	return nil, p.issuesErr
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
