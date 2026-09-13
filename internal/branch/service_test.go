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
