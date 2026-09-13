package git

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchListerListsLocalOriginAndDivergence(t *testing.T) {
	workdir := t.TempDir()
	runGit(t, workdir, "init", "-b", "main")
	runGit(t, workdir, "config", "user.email", "test@example.com")
	runGit(t, workdir, "config", "user.name", "Test User")
	runGit(t, workdir, "commit", "--allow-empty", "-m", "initial")
	mainSHA := strings.TrimSpace(runGit(t, workdir, "rev-parse", "main"))
	runGit(t, workdir, "remote", "add", "origin", "https://example.com/acme/project.git")
	runGit(t, workdir, "update-ref", "refs/remotes/origin/main", mainSHA)
	runGit(t, workdir, "checkout", "-b", "feature")
	runGit(t, workdir, "commit", "--allow-empty", "-m", "feature")
	runGit(t, workdir, "update-ref", "refs/remotes/origin/feature", mainSHA)
	runGit(t, workdir, "branch", "--set-upstream-to=origin/feature", "feature")

	inventory, err := NewBranchLister(workdir).List(context.Background(), 30)

	require.NoError(t, err)
	assert.Equal(t, "v1", inventory.SchemaVersion)
	assert.Equal(t, "https://example.com/acme/project.git", inventory.Origin)
	require.Len(t, inventory.Local, 2)
	assert.Equal(t, "feature", inventory.Local[0].Name)
	assert.True(t, inventory.Local[0].Current)
	assert.Equal(t, "origin/feature", inventory.Local[0].Upstream)
	require.NotNil(t, inventory.Local[0].Ahead)
	require.NotNil(t, inventory.Local[0].Behind)
	assert.Equal(t, 1, *inventory.Local[0].Ahead)
	assert.Zero(t, *inventory.Local[0].Behind)
	require.Len(t, inventory.OriginBranches, 2)
	assert.Equal(t, "feature", inventory.OriginBranches[0].Name)
	assert.Equal(t, mainSHA, inventory.OriginBranches[0].SHA)
}

func TestBranchListerWorksWithoutOrigin(t *testing.T) {
	workdir := t.TempDir()
	runGit(t, workdir, "init", "-b", "main")
	runGit(t, workdir, "config", "user.email", "test@example.com")
	runGit(t, workdir, "config", "user.name", "Test User")
	runGit(t, workdir, "commit", "--allow-empty", "-m", "initial")

	inventory, err := NewBranchLister(workdir).List(context.Background(), 30)

	require.NoError(t, err)
	assert.Empty(t, inventory.Origin)
	require.Len(t, inventory.Local, 1)
	assert.Empty(t, inventory.OriginBranches)
}

func TestBranchListerLimitsEachSource(t *testing.T) {
	workdir := t.TempDir()
	runGit(t, workdir, "init", "-b", "main")
	runGit(t, workdir, "config", "user.email", "test@example.com")
	runGit(t, workdir, "config", "user.name", "Test User")
	runGit(t, workdir, "commit", "--allow-empty", "-m", "initial")
	sha := strings.TrimSpace(runGit(t, workdir, "rev-parse", "main"))
	runGit(t, workdir, "remote", "add", "origin", "https://example.com/acme/project.git")
	runGit(t, workdir, "branch", "feature")
	runGit(t, workdir, "update-ref", "refs/remotes/origin/main", sha)
	runGit(t, workdir, "update-ref", "refs/remotes/origin/feature", sha)

	inventory, err := NewBranchLister(workdir).List(context.Background(), 1)

	require.NoError(t, err)
	assert.Len(t, inventory.Local, 1)
	assert.Len(t, inventory.OriginBranches, 1)
}

func runGit(t *testing.T, workdir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = workdir
	output, err := command.CombinedOutput()
	require.NoError(t, err, "git %s: %s", strings.Join(args, " "), output)
	return string(output)
}
