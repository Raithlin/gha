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
