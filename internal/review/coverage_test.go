package review

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raithlin/gha/internal/interfaces"
	"github.com/raithlin/gha/pkg/model"
)

func TestPreparationAndListingFailuresRemainExplicit(t *testing.T) {
	repository := model.RepositoryRef{Owner: "acme", Name: "project"}
	_, err := (*Service)(nil).PreparePullRequest(context.Background(), PreparePullRequestInput{})
	assert.ErrorContains(t, err, "not configured")
	_, err = NewService(&fakeProvider{}).PreparePullRequest(context.Background(), PreparePullRequestInput{Repository: repository})
	assert.ErrorContains(t, err, "get repository")
	_, err = NewService(&fakeProvider{repository: &model.Repository{}}).PreparePullRequest(context.Background(), PreparePullRequestInput{Repository: repository})
	assert.ErrorContains(t, err, "default branch")

	preparation, err := NewService(&fakeProvider{repository: &model.Repository{DefaultBranch: "main"}}).PreparePullRequest(context.Background(), PreparePullRequestInput{Repository: repository, Head: "main"})
	assert.NoError(t, err)
	assert.Equal(t, "not_applicable", preparation.Comparison.State)
	assert.ErrorContains(t, validatePullRequestPreparation(preparation), "same branch")

	_, err = NewService(&fakeProvider{prsErr: errors.New("down")}).ListMatching(context.Background(), repository, interfaces.ListPRsOptions{}, 1, nil)
	assert.ErrorContains(t, err, "list pull requests")
}

func TestReviewHelpersCoverUnavailableAndEmptySignals(t *testing.T) {
	assert.Equal(t, "unavailable", summarizeCheckRuns([]*model.CheckRun{nil}))
	assert.Equal(t, "unresolved", summarizeReviewThreads([]*model.ReviewThread{nil}))
	assert.Equal(t, "resolved", summarizeReviewThreads([]*model.ReviewThread{{IsResolved: true}}))
	assert.Empty(t, latestReviewsByUser([]*model.Review{nil}))
	assert.False(t, func() bool { _, ok := parseMergedAt(nil); return ok }())
	assert.False(t, func() bool { _, ok := parseMergedAt(&model.PullRequest{MergedAt: "bad"}); return ok }())
	assert.True(t, func() bool {
		_, ok := parseMergedAt(&model.PullRequest{MergedAt: time.Now().UTC().Format(time.RFC3339)})
		return ok
	}())

	provider := &fakeProvider{userErr: errors.New("token expired")}
	_, err := NewService(provider).AuthenticatedUser(context.Background())
	assert.ErrorContains(t, err, "token expired")
	_, err = NewService(&fakeProvider{}).AuthenticatedUser(context.Background())
	assert.ErrorContains(t, err, "no user")
}
