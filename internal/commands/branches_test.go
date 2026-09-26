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

type refreshingBranchLister struct {
	dryRun bool
}

func (l *refreshingBranchLister) List(context.Context, int) (*model.BranchInventory, error) {
	return &model.BranchInventory{SchemaVersion: model.BranchInventorySchemaVersion}, nil
}

func (l *refreshingBranchLister) RefreshOrigin(_ context.Context, _ int, dryRun bool) (*model.BranchInventory, error) {
	l.dryRun = dryRun
	state := "completed"
	if dryRun {
		state = "planned"
	}
	return &model.BranchInventory{
		SchemaVersion: model.BranchInventorySchemaVersion,
		OriginState:   "refreshed",
		OriginRefresh: model.OriginRefresh{State: state},
	}, nil
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

func TestBranchesCommandRejectsDryRunWithoutRefresh(t *testing.T) {
	command := newBranchesCmd(nil)
	command.SetArgs([]string{"--dry-run"})

	err := command.Execute()

	require.Error(t, err)
	assert.ErrorContains(t, err, "--dry-run requires --refresh-origin")
}

func TestBranchesCommandPlansOriginRefreshWithoutWriting(t *testing.T) {
	lister := &refreshingBranchLister{}
	command := newBranchesCmd(branch.NewService(lister))
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"--refresh-origin", "--dry-run", "--format", "json"})

	require.NoError(t, command.Execute())
	assert.True(t, lister.dryRun)
	var inventory model.BranchInventory
	require.NoError(t, json.Unmarshal(output.Bytes(), &inventory))
	assert.Equal(t, "planned", inventory.OriginRefresh.State)
}

func TestBranchesCommandRefreshesOriginByDefault(t *testing.T) {
	lister := &refreshingBranchLister{}
	command := newBranchesCmd(branch.NewService(lister))
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"--refresh-origin", "--format", "json"})

	require.NoError(t, command.Execute())
	assert.False(t, lister.dryRun)
	var inventory model.BranchInventory
	require.NoError(t, json.Unmarshal(output.Bytes(), &inventory))
	assert.Equal(t, "completed", inventory.OriginRefresh.State)
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

func TestBranchesCleanupCommandReturnsExplainableMergedCandidates(t *testing.T) {
	checkout := t.TempDir()
	require.NoError(t, exec.Command("git", "init", "--quiet", "-b", "main", checkout).Run())
	require.NoError(t, exec.Command("git", "-C", checkout, "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.Command("git", "-C", checkout, "config", "user.name", "Test User").Run())
	require.NoError(t, exec.Command("git", "-C", checkout, "commit", "--quiet", "--allow-empty", "-m", "initial").Run())
	require.NoError(t, exec.Command("git", "-C", checkout, "branch", "feature/merged").Run())
	require.NoError(t, exec.Command("git", "-C", checkout, "checkout", "--quiet", "-b", "feature/active").Run())
	require.NoError(t, exec.Command("git", "-C", checkout, "commit", "--quiet", "--allow-empty", "-m", "active").Run())
	require.NoError(t, exec.Command("git", "-C", checkout, "checkout", "--quiet", "main").Run())

	command := newBranchesCleanupCmd()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"--path", checkout, "--base", "main", "--format", "json"})

	require.NoError(t, command.Execute())
	var cleanup model.BranchCleanup
	require.NoError(t, json.Unmarshal(output.Bytes(), &cleanup))
	assert.Equal(t, model.BranchCleanupSchemaVersion, cleanup.SchemaVersion)
	assert.Equal(t, "main", cleanup.Base)
	assert.Equal(t, "tip_reachable_from_base", cleanup.Rule)
	require.Len(t, cleanup.Candidates, 1)
	assert.Equal(t, "feature/merged", cleanup.Candidates[0].Name)
	assert.Equal(t, "tip_reachable_from_base", cleanup.Candidates[0].Reason)
	require.Len(t, cleanup.Excluded, 2)
	assert.Equal(t, "feature/active", cleanup.Excluded[0].Name)
	assert.Equal(t, "not_reachable_from_base", cleanup.Excluded[0].Reason)
	assert.Equal(t, "main", cleanup.Excluded[1].Name)
	assert.Equal(t, "base_branch", cleanup.Excluded[1].Reason)
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
