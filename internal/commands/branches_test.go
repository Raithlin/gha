package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/internal/branch"
	"github.com/raithlin/gha/pkg/model"
)

type failingBranchLister struct{}

func (failingBranchLister) List(context.Context, int) (*model.BranchInventory, error) {
	return nil, errors.New("fatal: not a git repository")
}

func TestBranchesCommandShowsHelpWithoutConfiguration(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetArgs([]string{"branches", "--help"})

	require.NoError(t, root.Execute())
	assert.Contains(t, output.String(), "Inspect local branches and the remote-tracking branches for origin")
	assert.Contains(t, output.String(), "--limit")
}

func TestBranchesCommandValidatesLimitBeforeInspectingGit(t *testing.T) {
	command := newBranchesCmd(nil)
	command.SetArgs([]string{"--limit", "0"})

	err := command.Execute()

	require.Error(t, err)
	assert.ErrorContains(t, err, "limit must be between 1 and 100")
}

func TestBranchesCommandRendersStructuredErrors(t *testing.T) {
	command := newBranchesCmd(branch.NewService(failingBranchLister{}))
	var diagnostics bytes.Buffer
	command.SetErr(&diagnostics)
	command.SetArgs([]string{"--format", "json"})

	err := command.Execute()

	require.Error(t, err)
	assert.True(t, IsReportedError(err))
	var commandError model.CommandError
	require.NoError(t, json.Unmarshal(diagnostics.Bytes(), &commandError))
	assert.Equal(t, model.ErrorSchemaVersion, commandError.SchemaVersion)
	assert.Equal(t, "branch_inventory_failed", commandError.Code)
	assert.Contains(t, commandError.Message, "not a git repository")
}

func TestBranchesCommandRendersStructuredValidationErrors(t *testing.T) {
	command := newBranchesCmd(nil)
	var diagnostics bytes.Buffer
	command.SetErr(&diagnostics)
	command.SetArgs([]string{"--format", "json", "--limit", "0"})

	err := command.Execute()

	require.Error(t, err)
	assert.True(t, IsReportedError(err))
	var commandError model.CommandError
	require.NoError(t, json.Unmarshal(diagnostics.Bytes(), &commandError))
	assert.Equal(t, "invalid_argument", commandError.Code)
	assert.Contains(t, commandError.Message, "limit must be between 1 and 100")
}

func TestRootSuppressesCobraUsageAndDuplicateErrors(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)

	assert.True(t, root.SilenceUsage)
	assert.True(t, root.SilenceErrors)
}
