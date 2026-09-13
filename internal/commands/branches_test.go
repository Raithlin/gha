package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/internal/branch"
	"github.com/raithlin/gha/internal/git"
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

func TestBranchesCommandInspectsExplicitLocalPath(t *testing.T) {
	checkout := t.TempDir()
	require.NoError(t, exec.Command("git", "init", "--quiet", checkout).Run())
	require.NoError(t, exec.Command("git", "-C", checkout, "remote", "add", "origin", "git@github.com:Raithlin/gha.git").Run())

	command := newBranchesCmd(nil)
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"--path", checkout, "--format", "json"})

	require.NoError(t, command.Execute())
	var inventory model.BranchInventory
	require.NoError(t, json.Unmarshal(output.Bytes(), &inventory))
	assert.Equal(t, "git@github.com:Raithlin/gha.git", inventory.Origin)
}

func TestBranchShowCommandReturnsLocalFactsWhenProviderIsUnavailable(t *testing.T) {
	checkout := t.TempDir()
	require.NoError(t, exec.Command("git", "init", "--quiet", "-b", "main", checkout).Run())
	require.NoError(t, exec.Command("git", "-C", checkout, "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.Command("git", "-C", checkout, "config", "user.name", "Test User").Run())
	require.NoError(t, exec.Command("git", "-C", checkout, "commit", "--quiet", "--allow-empty", "-m", "initial").Run())

	command := newBranchShowCmd(branch.NewService(git.NewBranchLister(checkout)), git.NewRepositoryResolver("acme/project"))
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"main", "--format", "json"})

	require.NoError(t, command.Execute())
	var inspection model.BranchInspection
	require.NoError(t, json.Unmarshal(output.Bytes(), &inspection))
	assert.Equal(t, model.BranchInspectionSchemaVersion, inspection.SchemaVersion)
	require.NotNil(t, inspection.Local)
	assert.Equal(t, "main", inspection.Local.Name)
	assert.Equal(t, "unavailable", inspection.Safety.Requests.State)
}

func TestRootSuppressesCobraUsageAndDuplicateErrors(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)

	assert.True(t, root.SilenceUsage)
	assert.True(t, root.SilenceErrors)
}
