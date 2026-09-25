package git

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagWriterInspectsAndPublishesExactTag(t *testing.T) {
	workdir, remote := tagTestRepository(t)
	writer := NewTagWriter(workdir)
	inspection, err := writer.Inspect(context.Background(), "build-1", "HEAD")
	require.NoError(t, err)
	assert.NotEmpty(t, inspection.Commit)
	assert.False(t, inspection.LocalExists)
	assert.False(t, inspection.OriginExists)

	require.NoError(t, writer.Create(context.Background(), "build-1", inspection.Commit, ""))
	localExists, err := writer.LocalTagExists(context.Background(), "build-1")
	require.NoError(t, err)
	assert.True(t, localExists)
	require.NoError(t, writer.Push(context.Background(), "build-1"))
	assert.Equal(t, inspection.Commit, tagTestGit(t, remote, "--git-dir", remote, "rev-parse", "refs/tags/build-1"))

	inspection, err = writer.Inspect(context.Background(), "build-1", "HEAD")
	require.NoError(t, err)
	assert.True(t, inspection.LocalExists)
	assert.True(t, inspection.OriginExists)
}

func TestTagWriterCreatesAnnotatedTag(t *testing.T) {
	workdir, _ := tagTestRepository(t)
	writer := NewTagWriter(workdir)
	require.NoError(t, writer.Create(context.Background(), "release-1", "HEAD", "Release release-1"))
	assert.Equal(t, "Release release-1", tagTestGit(t, workdir, "-C", workdir, "for-each-ref", "--format=%(contents:subject)", "refs/tags/release-1"))
}

func TestTagWriterRejectsInvalidRefsAndCommits(t *testing.T) {
	workdir, _ := tagTestRepository(t)
	writer := NewTagWriter(workdir)
	_, err := writer.Inspect(context.Background(), "bad..tag", "HEAD")
	require.Error(t, err)
	_, err = writer.Inspect(context.Background(), "valid", "missing")
	require.Error(t, err)
	_, err = writer.LocalTagExists(context.Background(), "bad..tag")
	require.Error(t, err)
	assert.Error(t, writer.Create(context.Background(), "bad..tag", "HEAD", ""))
	assert.Error(t, writer.Push(context.Background(), "bad..tag"))
}

func TestTagWriterReportsOriginInspectionFailure(t *testing.T) {
	workdir, _ := tagTestRepository(t)
	tagTestGit(t, workdir, "remote", "set-url", "origin", t.TempDir()+"/missing.git")
	_, err := NewTagWriter(workdir).Inspect(context.Background(), "valid", "HEAD")
	require.ErrorContains(t, err, "inspect origin tag")
}

func tagTestRepository(t *testing.T) (string, string) {
	t.Helper()
	workdir, remote := t.TempDir(), t.TempDir()
	tagTestGit(t, "", "init", "--quiet", "--bare", remote)
	tagTestGit(t, workdir, "init", "--quiet", "-b", "main", workdir)
	tagTestGit(t, workdir, "config", "user.email", "test@example.com")
	tagTestGit(t, workdir, "config", "user.name", "Test User")
	tagTestGit(t, workdir, "commit", "--quiet", "--allow-empty", "-m", "initial")
	tagTestGit(t, workdir, "remote", "add", "origin", remote)
	tagTestGit(t, workdir, "push", "--quiet", "-u", "origin", "main")
	return workdir, remote
}

func tagTestGit(t *testing.T, directory string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = directory
	result, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, result)
	return strings.TrimSpace(string(result))
}
