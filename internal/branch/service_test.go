package branch

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/pkg/model"
)

type fakeLister struct {
	limit     int
	inventory *model.BranchInventory
	err       error
}

type fakeInspector struct {
	inspection *model.BranchInspection
	err        error
}

func (f *fakeInspector) List(context.Context, int) (*model.BranchInventory, error) {
	return nil, errors.New("not used")
}

func (f *fakeInspector) Inspect(context.Context, string) (*model.BranchInspection, error) {
	return f.inspection, f.err
}

type fakeSafetyProvider struct {
	safety model.BranchSafety
	err    error
}

type fakeRefresher struct {
	*fakeLister
	dryRun bool
}

func (f *fakeRefresher) RefreshOrigin(_ context.Context, limit int, dryRun bool) (*model.BranchInventory, error) {
	f.limit = limit
	f.dryRun = dryRun
	return f.inventory, f.err
}

func (f fakeSafetyProvider) InspectBranchSafety(context.Context, model.RepositoryRef, string) (model.BranchSafety, error) {
	return f.safety, f.err
}

func (f *fakeLister) List(_ context.Context, limit int) (*model.BranchInventory, error) {
	f.limit = limit
	return f.inventory, f.err
}

func TestInventoryPassesLimitToGitLister(t *testing.T) {
	fake := &fakeLister{inventory: &model.BranchInventory{SchemaVersion: model.BranchInventorySchemaVersion}}

	inventory, err := NewService(fake).Inventory(context.Background(), 12)

	require.NoError(t, err)
	assert.Equal(t, 12, fake.limit)
	assert.Equal(t, model.BranchInventorySchemaVersion, inventory.SchemaVersion)
}

func TestInventoryWrapsGitErrors(t *testing.T) {
	_, err := NewService(&fakeLister{err: errors.New("not a repository")}).Inventory(context.Background(), 30)

	require.Error(t, err)
	assert.ErrorContains(t, err, "inspect branches: not a repository")
}

func TestShowKeepsLocalFactsWhenProviderFails(t *testing.T) {
	service := NewService(&fakeInspector{inspection: &model.BranchInspection{Name: "feature"}}, fakeSafetyProvider{err: errors.New("token rejected")})

	inspection, err := service.Show(context.Background(), "feature", model.RepositoryRef{Owner: "acme", Name: "project"}, nil)

	require.NoError(t, err)
	require.NotNil(t, inspection.Repository)
	assert.Equal(t, "acme/project", inspection.Repository.String())
	assert.Equal(t, "unavailable", inspection.Safety.Requests.State)
	assert.Contains(t, inspection.Safety.Requests.Message, "token rejected")
	assert.Equal(t, "unavailable", inspection.Safety.Merge.State)
}

func TestShowMarksSafetyUnavailableWhenRepositoryCannotBeResolved(t *testing.T) {
	service := NewService(&fakeInspector{inspection: &model.BranchInspection{Name: "feature"}})

	inspection, err := service.Show(context.Background(), "feature", model.RepositoryRef{}, errors.New("origin is not GitHub"))

	require.NoError(t, err)
	assert.Equal(t, "unavailable", inspection.Safety.Protection.State)
	assert.Equal(t, "origin is not GitHub", inspection.Safety.Protection.Message)
}

func TestInventoryAndRefreshValidateLimits(t *testing.T) {
	service := NewService(&fakeLister{})
	for _, limit := range []int{0, 101} {
		_, err := service.Inventory(context.Background(), limit)
		assert.ErrorContains(t, err, "limit must be between 1 and 100")
		_, err = service.RefreshOrigin(context.Background(), limit, true)
		assert.ErrorContains(t, err, "limit must be between 1 and 100")
	}
}

func TestRefreshOriginUsesRefresherAndReportsUnsupportedLister(t *testing.T) {
	refresher := &fakeRefresher{fakeLister: &fakeLister{inventory: &model.BranchInventory{OriginRefresh: model.OriginRefresh{State: "planned"}}}}
	inventory, err := NewService(refresher).RefreshOrigin(context.Background(), 10, true)
	require.NoError(t, err)
	assert.Equal(t, "planned", inventory.OriginRefresh.State)
	assert.Equal(t, 10, refresher.limit)
	assert.True(t, refresher.dryRun)

	_, err = NewService(&fakeLister{}).RefreshOrigin(context.Background(), 10, false)
	assert.ErrorContains(t, err, "not supported")
}

func TestWithListerAndShowCoverProviderOutcomes(t *testing.T) {
	inspection := &model.BranchInspection{Name: "feature"}
	provider := fakeSafetyProvider{safety: model.BranchSafety{Provider: "github"}}
	service := NewService(&fakeLister{}, provider).WithLister(&fakeInspector{inspection: inspection})
	result, err := service.Show(context.Background(), "feature", model.RepositoryRef{Owner: "acme", Name: "project"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "github", result.Safety.Provider)

	result, err = NewService(&fakeInspector{inspection: &model.BranchInspection{Name: "feature"}}).Show(context.Background(), "feature", model.RepositoryRef{}, nil)
	require.NoError(t, err)
	assert.ErrorContains(t, errors.New(result.Safety.Requests.Message), "repository is unavailable")

	_, err = NewService(&fakeLister{}).Show(context.Background(), "feature", model.RepositoryRef{}, nil)
	assert.ErrorContains(t, err, "not supported")
	_, err = NewService(&fakeInspector{err: errors.New("missing")}).Show(context.Background(), "feature", model.RepositoryRef{}, nil)
	assert.ErrorContains(t, err, "inspect branch: missing")
}

func TestBranchServiceCoversNilAndRefreshFailureOutcomes(t *testing.T) {
	service := (*Service)(nil).WithLister(&fakeInspector{inspection: &model.BranchInspection{Name: "feature"}})
	inspection, err := service.Show(context.Background(), "feature", model.RepositoryRef{Owner: "acme", Name: "project"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "unavailable", inspection.Safety.Permissions.State)

	_, err = NewService(&fakeRefresher{fakeLister: &fakeLister{err: errors.New("fetch failed")}}).RefreshOrigin(context.Background(), 10, false)
	assert.ErrorContains(t, err, "refresh origin: fetch failed")
}
