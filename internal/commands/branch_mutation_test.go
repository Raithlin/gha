package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/internal/branch"
	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/pkg/model"
)

func TestBranchCreateDryRunReportsBothTargetsWithoutWriting(t *testing.T) {
	checkout, _ := mutationRepository(t)
	command := newBranchCreateCmd()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"feature", "--publish", "--dry-run", "--path", checkout, "--format", "json"})

	require.NoError(t, command.Execute())
	var result model.BranchMutation
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	assert.Equal(t, model.BranchMutationSchemaVersion, result.SchemaVersion)
	assert.True(t, result.DryRun)
	assert.Equal(t, "planned", result.Local)
	assert.Equal(t, "planned", result.Origin)
	assertBranchMissing(t, checkout, "feature")
}

func TestBranchCreatePublishesOnlyWithExplicitOriginConfirmation(t *testing.T) {
	checkout, remote := mutationRepository(t)
	command := newBranchCreateCmd()
	command.SetArgs([]string{"feature", "--publish", "--path", checkout})
	err := command.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "--confirm-origin")
	assertBranchMissing(t, checkout, "feature")

	command = newBranchCreateCmd()
	command.SetArgs([]string{"feature", "--publish", "--confirm-origin", "--path", checkout})
	require.NoError(t, command.Execute())
	assertBranchExists(t, checkout, "feature")
	assertBranchExists(t, remote, "feature")
}

func TestBranchPublishPreflightsAnUnpublishedLocalBranchWithoutWriting(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")

	safety := model.BranchSafety{
		Permissions: model.ProviderSignal{State: "available"},
		CanPush:     boolPointer(true),
	}
	command := newBranchPublishCmd(branch.NewService(git.NewBranchLister(checkout), mutationSafetyProvider{safety: safety}), git.NewRepositoryResolver("acme/project"))
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"feature", "--dry-run", "--repo", "acme/project", "--path", checkout, "--format", "json"})

	require.NoError(t, command.Execute())
	var result model.BranchPublication
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	assert.Equal(t, model.BranchPublicationSchemaVersion, result.SchemaVersion)
	assert.Equal(t, "origin/feature", result.Target)
	assert.Equal(t, "planned", result.Publication)
	assert.True(t, result.DryRun)
	require.NotNil(t, result.Local)
	assert.Empty(t, result.Local.Upstream)
	assert.Equal(t, "not_tracked", result.Local.DivergenceState)
	assert.Equal(t, "available", result.Permissions.State)
	assert.True(t, *result.CanPush)
	assertBranchMissing(t, remote, "feature")
}

func TestBranchPublishRequiresOriginConfirmationBeforeWriting(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	safety := model.BranchSafety{Permissions: model.ProviderSignal{State: "available"}, CanPush: boolPointer(true)}
	command := newBranchPublishCmd(branch.NewService(git.NewBranchLister(checkout), mutationSafetyProvider{safety: safety}), git.NewRepositoryResolver("acme/project"))
	command.SetArgs([]string{"feature", "--repo", "acme/project", "--path", checkout})

	err := command.Execute()

	require.Error(t, err)
	assert.ErrorContains(t, err, "--confirm-origin")
	assertBranchMissing(t, remote, "feature")
}

func TestBranchPublishSetsUpstreamOnlyAfterConfirmedSafePreflight(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	safety := model.BranchSafety{Permissions: model.ProviderSignal{State: "available"}, CanPush: boolPointer(true)}
	command := newBranchPublishCmd(branch.NewService(git.NewBranchLister(checkout), mutationSafetyProvider{safety: safety}), git.NewRepositoryResolver("acme/project"))
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"feature", "--confirm-origin", "--repo", "acme/project", "--path", checkout, "--format", "json"})

	require.NoError(t, command.Execute())
	var result model.BranchPublication
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	assert.Equal(t, "completed", result.Publication)
	assertBranchExists(t, remote, "feature")
	assert.Equal(t, "origin/feature", upstreamMutationBranch(t, checkout, "feature"))
}

func TestBranchPublishUsesGitPushWhenProviderPermissionIsUnavailable(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	safety := model.BranchSafety{Permissions: model.ProviderSignal{State: "unavailable", Message: "token rejected"}}
	command := newBranchPublishCmd(branch.NewService(git.NewBranchLister(checkout), mutationSafetyProvider{safety: safety}), git.NewRepositoryResolver("acme/project"))
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"feature", "--confirm-origin", "--repo", "acme/project", "--path", checkout, "--format", "json"})

	require.NoError(t, command.Execute())

	var result model.BranchPublication
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	assert.Equal(t, "completed", result.Publication)
	assert.Equal(t, "unavailable", result.Permissions.State)
	assertBranchExists(t, remote, "feature")
}

func TestBranchPublishRejectsAnAlreadyTrackedBranch(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	runMutationGit(t, checkout, "push", "-u", "origin", "feature")
	safety := model.BranchSafety{Permissions: model.ProviderSignal{State: "available"}, CanPush: boolPointer(true)}
	command := newBranchPublishCmd(branch.NewService(git.NewBranchLister(checkout), mutationSafetyProvider{safety: safety}), git.NewRepositoryResolver("acme/project"))
	command.SetArgs([]string{"feature", "--dry-run", "--repo", "acme/project", "--path", checkout})

	err := command.Execute()

	require.Error(t, err)
	assert.ErrorContains(t, err, "already tracks origin/feature")
	assertBranchExists(t, remote, "feature")
}

func TestBranchRenameOriginRequiresConfirmationAndCanBeExplicitlyForced(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	runMutationGit(t, checkout, "push", "-u", "origin", "feature")

	command := newBranchRenameCmd(nil, nil)
	command.SetArgs([]string{"feature", "better", "--origin", "--path", checkout})
	err := command.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "--confirm-origin")
	assertBranchExists(t, checkout, "feature")

	command = newBranchRenameCmd(nil, nil)
	command.SetArgs([]string{"feature", "better", "--origin", "--confirm-origin", "--force", "--path", checkout})
	require.NoError(t, command.Execute())
	assertBranchMissing(t, checkout, "feature")
	assertBranchExists(t, checkout, "better")
	assertBranchMissing(t, remote, "feature")
	assertBranchExists(t, remote, "better")
}

func TestBranchDeleteRequiresAnExplicitTargetAndSupportsBothTargets(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	runMutationGit(t, checkout, "push", "-u", "origin", "feature")

	command := newBranchDeleteCmd(nil, nil)
	command.SetArgs([]string{"feature", "--path", checkout})
	err := command.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "--local and/or --origin")
	assertBranchExists(t, checkout, "feature")

	command = newBranchDeleteCmd(nil, nil)
	command.SetArgs([]string{"feature", "--local", "--origin", "--confirm-origin", "--force", "--path", checkout})
	require.NoError(t, command.Execute())
	assertBranchMissing(t, checkout, "feature")
	assertBranchMissing(t, remote, "feature")
}

func TestBranchDeleteOriginGuardsTheDefaultBranchBeforeWriting(t *testing.T) {
	checkout, remote := mutationRepository(t)
	safety := model.BranchSafety{
		Permissions:   model.ProviderSignal{State: "available"},
		CanPush:       boolPointer(true),
		DefaultBranch: model.ProviderSignal{State: "available"},
		IsDefault:     boolPointer(true),
		Protection:    model.ProviderSignal{State: "available"},
		Protected:     boolPointer(false),
	}
	service := branch.NewService(git.NewBranchLister(checkout), mutationSafetyProvider{safety: safety})
	command := newBranchDeleteCmd(service, git.NewRepositoryResolver("acme/project"))
	command.SetArgs([]string{"main", "--origin", "--confirm-origin", "--repo", "acme/project", "--path", checkout})

	err := command.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "default branch")
	assertBranchExists(t, remote, "main")
}

func TestBranchDeleteCurrentMergedBranchSwitchesToDefaultBeforeDeleting(t *testing.T) {
	checkout, _ := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	runMutationGit(t, checkout, "switch", "feature")
	runMutationGit(t, checkout, "commit", "--allow-empty", "-m", "feature")
	runMutationGit(t, checkout, "switch", "main")
	runMutationGit(t, checkout, "merge", "--ff-only", "feature")
	runMutationGit(t, checkout, "switch", "feature")

	command := newBranchDeleteCmd(nil, nil)
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"feature", "--local", "--path", checkout, "--format", "json"})

	require.NoError(t, command.Execute())
	var result model.BranchMutation
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	assert.Equal(t, "main", result.CheckedOut)
	assert.Equal(t, "completed", result.Local)
	assertBranchMissing(t, checkout, "feature")
	assert.Equal(t, "main", currentMutationBranch(t, checkout))
}

func TestBranchDeleteCurrentDryRunReportsCheckoutWithoutChangingBranches(t *testing.T) {
	checkout, _ := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	runMutationGit(t, checkout, "switch", "feature")

	command := newBranchDeleteCmd(nil, nil)
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"feature", "--local", "--dry-run", "--path", checkout, "--format", "json"})

	require.NoError(t, command.Execute())
	var result model.BranchMutation
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	assert.True(t, result.DryRun)
	assert.Equal(t, "planned", result.Local)
	assert.Equal(t, "main", result.CheckedOut)
	assertBranchExists(t, checkout, "feature")
	assert.Equal(t, "feature", currentMutationBranch(t, checkout))
}

func TestBranchDeleteRefusesTheCurrentDefaultBranch(t *testing.T) {
	checkout, _ := mutationRepository(t)
	command := newBranchDeleteCmd(nil, nil)
	command.SetArgs([]string{"main", "--local", "--path", checkout})

	err := command.Execute()

	require.Error(t, err)
	assert.ErrorContains(t, err, "current default branch")
	assertBranchExists(t, checkout, "main")
}

type mutationSafetyProvider struct {
	safety model.BranchSafety
}

func (p mutationSafetyProvider) InspectBranchSafety(_ context.Context, _ model.RepositoryRef, _ string) (model.BranchSafety, error) {
	return p.safety, nil
}

func boolPointer(value bool) *bool { return &value }

func mutationRepository(t *testing.T) (string, string) {
	t.Helper()
	checkout := filepath.Join(t.TempDir(), "checkout")
	remote := filepath.Join(t.TempDir(), "origin.git")
	runMutationGit(t, "", "init", "--quiet", "--bare", remote)
	runMutationGit(t, "", "init", "--quiet", "-b", "main", checkout)
	runMutationGit(t, checkout, "config", "user.email", "test@example.com")
	runMutationGit(t, checkout, "config", "user.name", "Test User")
	runMutationGit(t, checkout, "commit", "--quiet", "--allow-empty", "-m", "initial")
	runMutationGit(t, checkout, "remote", "add", "origin", remote)
	runMutationGit(t, checkout, "push", "-u", "origin", "main")
	runMutationGit(t, checkout, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	return checkout, remote
}

func assertBranchExists(t *testing.T, directory, name string) {
	t.Helper()
	command := exec.Command("git", "-C", directory, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	if filepath.Ext(directory) == ".git" {
		command = exec.Command("git", "--git-dir", directory, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	}
	require.NoError(t, command.Run())
}

func assertBranchMissing(t *testing.T, directory, name string) {
	t.Helper()
	command := exec.Command("git", "-C", directory, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	if filepath.Ext(directory) == ".git" {
		command = exec.Command("git", "--git-dir", directory, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	}
	require.Error(t, command.Run())
}

func runMutationGit(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, output)
}

func currentMutationBranch(t *testing.T, directory string) string {
	t.Helper()
	command := exec.Command("git", "branch", "--show-current")
	command.Dir = directory
	output, err := command.Output()
	require.NoError(t, err)
	return strings.TrimSpace(string(output))
}

func upstreamMutationBranch(t *testing.T, directory, name string) string {
	t.Helper()
	command := exec.Command("git", "for-each-ref", "--format=%(upstream:short)", "refs/heads/"+name)
	command.Dir = directory
	output, err := command.Output()
	require.NoError(t, err)
	return strings.TrimSpace(string(output))
}
