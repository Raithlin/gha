package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
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
