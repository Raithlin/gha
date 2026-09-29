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

func TestBranchWriterValidateCreateChecksStartAndRefCollisions(t *testing.T) {
	workdir, _ := branchValidationRepository(t)
	writer := NewBranchWriter(workdir)
	ctx := context.Background()

	require.NoError(t, writer.ValidateCreate(ctx, "new-feature", "main", true))
	require.NoError(t, writer.ValidateCreate(ctx, "from-head", "", false))

	err := writer.ValidateCreate(ctx, "new-feature", "missing", false)
	assert.ErrorContains(t, err, "start point")
	err = writer.ValidateCreate(ctx, "bad name", "main", false)
	assert.ErrorContains(t, err, "invalid branch name")

	runGit(t, workdir, "branch", "existing")
	err = writer.ValidateCreate(ctx, "existing", "main", false)
	assert.ErrorContains(t, err, "already exists locally")

	runGit(t, workdir, "branch", "nested/child")
	err = writer.ValidateCreate(ctx, "nested", "main", false)
	assert.ErrorContains(t, err, "conflicts with existing ref")

	runGit(t, workdir, "branch", "remote-existing")
	runGit(t, workdir, "push", "origin", "remote-existing")
	runGit(t, workdir, "branch", "-D", "remote-existing")
	err = writer.ValidateCreate(ctx, "remote-existing", "main", true)
	assert.ErrorContains(t, err, "already exists on origin")

	noOrigin := t.TempDir()
	runGit(t, noOrigin, "init", "-b", "main")
	runGit(t, noOrigin, "config", "user.email", "test@example.com")
	runGit(t, noOrigin, "config", "user.name", "Test User")
	runGit(t, noOrigin, "commit", "--allow-empty", "-m", "initial")
	err = NewBranchWriter(noOrigin).ValidateCreate(ctx, "new-feature", "HEAD", true)
	assert.ErrorContains(t, err, "read origin branch refs")
	assert.ErrorContains(t, NewBranchWriter(t.TempDir()).ValidateCreate(ctx, "new-feature", "HEAD", false), "inspect local branches")
	_, err = writer.DefaultBranch(ctx)
	assert.ErrorContains(t, err, "read cached origin default branch")
}

func TestBranchWriterValidateRenameChecksLocalAndOriginRefs(t *testing.T) {
	workdir, _ := branchValidationRepository(t)
	writer := NewBranchWriter(workdir)
	ctx := context.Background()
	runGit(t, workdir, "branch", "feature")
	runGit(t, workdir, "push", "origin", "feature")

	require.NoError(t, writer.ValidateRename(ctx, "feature", "renamed", true))
	assert.ErrorContains(t, writer.ValidateRename(ctx, "missing", "renamed", false), "not found locally")
	assert.ErrorContains(t, writer.ValidateRename(ctx, "bad name", "renamed", false), "invalid branch name")
	assert.ErrorContains(t, writer.ValidateRename(ctx, "feature", "bad name", false), "invalid branch name")
	assert.ErrorContains(t, writer.ValidateRename(ctx, "feature", "feature", false), "identical")

	runGit(t, workdir, "branch", "occupied")
	assert.ErrorContains(t, writer.ValidateRename(ctx, "feature", "occupied", false), "already exists locally")

	runGit(t, workdir, "branch", "local-only")
	assert.ErrorContains(t, writer.ValidateRename(ctx, "local-only", "unused", true), "does not exist on origin")

	runGit(t, workdir, "branch", "remote-occupied")
	runGit(t, workdir, "push", "origin", "remote-occupied")
	runGit(t, workdir, "branch", "-D", "remote-occupied")
	assert.ErrorContains(t, writer.ValidateRename(ctx, "feature", "remote-occupied", true), "already exists on origin")

	noOrigin := t.TempDir()
	runGit(t, noOrigin, "init", "-b", "main")
	runGit(t, noOrigin, "config", "user.email", "test@example.com")
	runGit(t, noOrigin, "config", "user.name", "Test User")
	runGit(t, noOrigin, "commit", "--allow-empty", "-m", "initial")
	runGit(t, noOrigin, "branch", "feature")
	assert.ErrorContains(t, NewBranchWriter(noOrigin).ValidateRename(ctx, "feature", "renamed", true), "read origin branch refs")
	assert.ErrorContains(t, NewBranchWriter(t.TempDir()).ValidateRename(ctx, "feature", "renamed", false), "inspect local branches")
}

func TestBranchWriterValidateDeleteChecksSelectedRefsAndMergeState(t *testing.T) {
	workdir, _ := branchValidationRepository(t)
	writer := NewBranchWriter(workdir)
	ctx := context.Background()
	runGit(t, workdir, "branch", "feature")
	runGit(t, workdir, "push", "origin", "feature")

	require.NoError(t, writer.ValidateDelete(ctx, "feature", true, true))
	assert.ErrorContains(t, writer.ValidateDelete(ctx, "feature", false, false), "select at least one")
	assert.ErrorContains(t, writer.ValidateDelete(ctx, "bad name", true, false), "invalid branch name")
	assert.ErrorContains(t, writer.ValidateDelete(ctx, "missing", true, false), "not found locally")
	assert.ErrorContains(t, writer.ValidateDelete(ctx, "local-only", false, true), "does not exist on origin")
	assert.ErrorContains(t, NewBranchWriter(t.TempDir()).ValidateDelete(ctx, "feature", true, false), "inspect local branches")
	assert.NoError(t, writer.ValidateLocalDelete(ctx, "main", false, "main"))

	noOrigin := t.TempDir()
	runGit(t, noOrigin, "init", "-b", "main")
	runGit(t, noOrigin, "config", "user.email", "test@example.com")
	runGit(t, noOrigin, "config", "user.name", "Test User")
	runGit(t, noOrigin, "commit", "--allow-empty", "-m", "initial")
	assert.ErrorContains(t, NewBranchWriter(noOrigin).ValidateDelete(ctx, "main", false, true), "read origin branch refs")

	runGit(t, workdir, "switch", "feature")
	runGit(t, workdir, "commit", "--allow-empty", "-m", "unmerged")
	runGit(t, workdir, "switch", "main")
	assert.ErrorContains(t, writer.ValidateLocalDelete(ctx, "feature", false, "main"), "not fully merged")
	assert.NoError(t, writer.ValidateLocalDelete(ctx, "feature", true, "main"))
	assert.ErrorContains(t, writer.ValidateLocalDelete(ctx, "feature", false, "missing"), "does not resolve to a commit")
	assert.ErrorContains(t, NewBranchWriter(t.TempDir()).ValidateLocalDelete(ctx, "feature", false, ""), "check whether branch")
}

func branchValidationRepository(t *testing.T) (string, string) {
	t.Helper()
	workdir := t.TempDir()
	remote := t.TempDir()
	runGit(t, remote, "init", "--bare")
	runGit(t, workdir, "init", "-b", "main")
	runGit(t, workdir, "config", "user.email", "test@example.com")
	runGit(t, workdir, "config", "user.name", "Test User")
	runGit(t, workdir, "commit", "--allow-empty", "-m", "initial")
	runGit(t, workdir, "remote", "add", "origin", remote)
	runGit(t, workdir, "push", "-u", "origin", "main")
	return workdir, remote
}
