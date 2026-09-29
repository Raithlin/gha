package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/pkg/model"
)

func TestWorktreesCommandReturnsStructuredInventory(t *testing.T) {
	checkout := commandWorktreeRepository(t)
	var output bytes.Buffer
	command := newWorktreesCmd()
	command.SetOut(&output)
	command.SetArgs([]string{"--path", checkout, "--format", "json"})

	require.NoError(t, command.Execute())
	var result model.WorktreeInventory
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	assert.Equal(t, model.WorktreeInventorySchemaVersion, result.SchemaVersion)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Worktrees, 1)
	assert.True(t, result.Worktrees[0].Main)
	assert.True(t, result.Worktrees[0].Current)
	assert.Equal(t, "clean", result.Worktrees[0].StatusState)
}

func TestWorktreeCommandsReturnStructuredPreflightErrors(t *testing.T) {
	var diagnostics bytes.Buffer
	command := newWorktreesCmd()
	command.SetErr(&diagnostics)
	command.SetArgs([]string{"--format", "json", "--path", t.TempDir()})
	require.Error(t, command.Execute())
	var commandError model.CommandError
	require.NoError(t, json.Unmarshal(diagnostics.Bytes(), &commandError))
	assert.Equal(t, "worktree_inventory_failed", commandError.Code)

	command = newWorktreesCmd()
	command.SetArgs([]string{"--format", "xml"})
	assert.ErrorContains(t, command.Execute(), "unsupported format")

	command = newWorktreeAddCmd()
	command.SetArgs([]string{"/tmp/new-linked", "--branch", "feature/new", "--new-branch", "--dry-run", "--format", "xml"})
	assert.ErrorContains(t, command.Execute(), "unsupported format")

	command = newWorktreeAddCmd()
	diagnostics.Reset()
	command.SetErr(&diagnostics)
	command.SetArgs([]string{"/tmp/new-linked", "--branch", "bad branch", "--new-branch", "--dry-run", "--path", commandWorktreeRepository(t), "--format", "json"})
	require.Error(t, command.Execute())
	require.NoError(t, json.Unmarshal(diagnostics.Bytes(), &commandError))
	assert.Equal(t, "worktree_mutation_failed", commandError.Code)

	command = newWorktreeRemoveCmd()
	command.SetArgs([]string{"/tmp/linked", "--dry-run", "--format", "xml"})
	assert.ErrorContains(t, command.Execute(), "unsupported format")
}

func TestWorktreeAddPreviewsWithoutWriting(t *testing.T) {
	checkout := commandWorktreeRepository(t)
	target := filepath.Join(t.TempDir(), "linked")

	var output bytes.Buffer
	command := newWorktreeAddCmd()
	command.SetOut(&output)
	command.SetArgs([]string{target, "--branch", "feature/new", "--new-branch", "--from", "HEAD", "--dry-run", "--path", checkout, "--format", "json"})
	require.NoError(t, command.Execute())
	var plan model.WorktreeMutation
	require.NoError(t, json.Unmarshal(output.Bytes(), &plan))
	assert.Equal(t, model.WorktreeMutationSchemaVersion, plan.SchemaVersion)
	assert.Equal(t, "add", plan.Operation)
	assert.Equal(t, "planned", plan.State)
	assert.True(t, plan.DryRun)
	assert.NoDirExists(t, target)
}

func TestWorktreeAddRunsByDefault(t *testing.T) {
	checkout := commandWorktreeRepository(t)
	target := filepath.Join(t.TempDir(), "linked")
	command := newWorktreeAddCmd()
	command.SetArgs([]string{target, "--branch", "feature/new", "--new-branch", "--path", checkout, "--format", "json"})

	var output bytes.Buffer
	command.SetOut(&output)
	require.NoError(t, command.Execute())
	var result model.WorktreeMutation
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	assert.Equal(t, "completed", result.State)
	assert.False(t, result.DryRun)
	assert.Equal(t, "feature/new", strings.TrimSpace(runCommandGit(t, target, "branch", "--show-current")))
}

func TestWorktreeCommandsRejectConfirmFlagAndInvalidLimits(t *testing.T) {
	checkout := commandWorktreeRepository(t)
	target := filepath.Join(t.TempDir(), "linked")
	command := newWorktreeAddCmd()
	command.SetArgs([]string{target, "--branch", "feature/new", "--new-branch", "--confirm", "--path", checkout})
	assert.ErrorContains(t, command.Execute(), "unknown flag: --confirm")

	command = newWorktreeRemoveCmd()
	command.SetArgs([]string{target, "--confirm", "--path", checkout})
	assert.ErrorContains(t, command.Execute(), "unknown flag: --confirm")

	command = newWorktreesCmd()
	command.SetArgs([]string{"--path", checkout, "--limit", "0"})
	assert.ErrorContains(t, command.Execute(), "limit must be between 1 and 100")
}

func TestWorktreeRemovePreviewsThenRunsByDefault(t *testing.T) {
	checkout := commandWorktreeRepository(t)
	target := filepath.Join(t.TempDir(), "linked")
	runCommandGit(t, checkout, "branch", "feature/remove")
	runCommandGit(t, checkout, "worktree", "add", "--quiet", target, "feature/remove")

	var output bytes.Buffer
	command := newWorktreeRemoveCmd()
	command.SetOut(&output)
	command.SetArgs([]string{target, "--dry-run", "--path", checkout, "--format", "json"})
	require.NoError(t, command.Execute())
	var plan model.WorktreeMutation
	require.NoError(t, json.Unmarshal(output.Bytes(), &plan))
	assert.Equal(t, "remove", plan.Operation)
	assert.True(t, plan.DryRun)
	assert.Equal(t, "planned", plan.State)
	assert.DirExists(t, target)

	command = newWorktreeRemoveCmd()
	command.SetArgs([]string{target, "--path", checkout})
	require.NoError(t, command.Execute())
	assert.NoDirExists(t, target)
}

func TestWorktreeRemoveRejectsDirtyTargetsInPreviewAndDefaultModes(t *testing.T) {
	checkout := commandWorktreeRepository(t)
	target := filepath.Join(t.TempDir(), "linked")
	runCommandGit(t, checkout, "branch", "feature/remove")
	runCommandGit(t, checkout, "worktree", "add", "--quiet", target, "feature/remove")
	require.NoError(t, osWriteCommandFile(filepath.Join(target, "dirty.txt"), []byte("dirty")))

	command := newWorktreeRemoveCmd()
	command.SetArgs([]string{target, "--dry-run", "--path", checkout})
	err := command.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "dirty")
	assert.DirExists(t, target)

	command = newWorktreeRemoveCmd()
	command.SetArgs([]string{target, "--path", checkout})
	err = command.Execute()
	assert.ErrorContains(t, err, "dirty")
	assert.DirExists(t, target)
}

func commandWorktreeRepository(t *testing.T) string {
	t.Helper()
	checkout := t.TempDir()
	runCommandGit(t, checkout, "init", "--quiet", "-b", "main", checkout)
	runCommandGit(t, checkout, "config", "user.email", "test@example.com")
	runCommandGit(t, checkout, "config", "user.name", "Test User")
	runCommandGit(t, checkout, "commit", "--quiet", "--allow-empty", "-m", "initial")
	return checkout
}

func runCommandGit(t *testing.T, directory string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, output)
	return string(output)
}

func osWriteCommandFile(path string, content []byte) error {
	return os.WriteFile(path, content, 0o600)
}
