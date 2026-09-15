package git

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchWriterLocalBranchLifecycle(t *testing.T) {
	workdir := t.TempDir()
	runGit(t, workdir, "init", "-b", "main")
	runGit(t, workdir, "config", "user.email", "test@example.com")
	runGit(t, workdir, "config", "user.name", "Test User")
	runGit(t, workdir, "commit", "--allow-empty", "-m", "initial")

	writer := NewBranchWriter(workdir)
	require.NoError(t, writer.CreateLocal(context.Background(), "feature", ""))
	require.NoError(t, writer.RenameLocal(context.Background(), "feature", "renamed"))
	require.NoError(t, writer.Switch(context.Background(), "renamed"))

	branch, err := writer.CurrentBranch(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "renamed", branch)

	require.NoError(t, writer.Switch(context.Background(), "main"))
	require.NoError(t, writer.DeleteLocal(context.Background(), "renamed", false))
	assert.NotContains(t, runGit(t, workdir, "branch", "--format=%(refname:short)"), "renamed")
}

func TestBranchWriterCreatesFromExplicitRevisionAndReportsGitDiagnostics(t *testing.T) {
	workdir := t.TempDir()
	runGit(t, workdir, "init", "-b", "main")
	runGit(t, workdir, "config", "user.email", "test@example.com")
	runGit(t, workdir, "config", "user.name", "Test User")
	runGit(t, workdir, "commit", "--allow-empty", "-m", "initial")

	writer := NewBranchWriter(workdir)
	require.NoError(t, writer.CreateLocal(context.Background(), "from-main", "main"))
	err := writer.Switch(context.Background(), "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

func TestBranchWriterDefaultBranchAndDetachedHead(t *testing.T) {
	workdir := t.TempDir()
	runGit(t, workdir, "init", "-b", "main")
	runGit(t, workdir, "config", "user.email", "test@example.com")
	runGit(t, workdir, "config", "user.name", "Test User")
	runGit(t, workdir, "commit", "--allow-empty", "-m", "initial")
	sha := strings.TrimSpace(runGit(t, workdir, "rev-parse", "HEAD"))
	runGit(t, workdir, "remote", "add", "origin", "https://example.com/acme/project.git")
	runGit(t, workdir, "update-ref", "refs/remotes/origin/main", sha)
	runGit(t, workdir, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")

	writer := NewBranchWriter(workdir)
	branch, err := writer.DefaultBranch(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "main", branch)

	runGit(t, workdir, "checkout", "--detach")
	current, err := writer.CurrentBranch(context.Background())
	require.NoError(t, err)
	assert.Empty(t, current)
}

func TestCurrentBranchRejectsDetachedHead(t *testing.T) {
	workdir := t.TempDir()
	runGit(t, workdir, "init", "-b", "main")
	runGit(t, workdir, "config", "user.email", "test@example.com")
	runGit(t, workdir, "config", "user.name", "Test User")
	runGit(t, workdir, "commit", "--allow-empty", "-m", "initial")
	runGit(t, workdir, "checkout", "--detach")

	_, err := CurrentBranch(context.Background(), workdir)
	require.Error(t, err)
	assert.ErrorContains(t, err, "detached")
}

func TestBranchWriterPublishesRenamesAndDeletesOriginBranches(t *testing.T) {
	workdir := t.TempDir()
	remote := t.TempDir()
	runGit(t, remote, "init", "--bare")
	runGit(t, workdir, "init", "-b", "main")
	runGit(t, workdir, "config", "user.email", "test@example.com")
	runGit(t, workdir, "config", "user.name", "Test User")
	runGit(t, workdir, "commit", "--allow-empty", "-m", "initial")
	runGit(t, workdir, "remote", "add", "origin", remote)
	runGit(t, workdir, "push", "-u", "origin", "main")
	runGit(t, workdir, "branch", "feature")

	writer := NewBranchWriter(workdir)
	require.NoError(t, writer.Publish(context.Background(), "feature"))
	require.NoError(t, writer.RenameLocal(context.Background(), "feature", "better"))
	require.NoError(t, writer.RenameOrigin(context.Background(), "feature", "better"))
	assert.Contains(t, runGit(t, workdir, "ls-remote", "--heads", "origin", "better"), "refs/heads/better")
	require.NoError(t, writer.DeleteOrigin(context.Background(), "better"))
	assert.Empty(t, strings.TrimSpace(runGit(t, workdir, "ls-remote", "--heads", "origin", "better")))
}
