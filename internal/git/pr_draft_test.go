package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDraftPullRequestSummarizesBaseToHead(t *testing.T) {
	repo := t.TempDir()
	gitTestRun(t, repo, "init", "--quiet")
	gitTestRun(t, repo, "config", "user.email", "test@example.com")
	gitTestRun(t, repo, "config", "user.name", "Test")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "base.txt"), []byte("base\n"), 0600))
	gitTestRun(t, repo, "add", ".")
	gitTestRun(t, repo, "commit", "--quiet", "-m", "base commit")
	gitTestRun(t, repo, "branch", "-M", "main")
	gitTestRun(t, repo, "checkout", "-b", "feature")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "feature.txt"), []byte("feature\n"), 0600))
	gitTestRun(t, repo, "add", ".")
	gitTestRun(t, repo, "commit", "--quiet", "-m", "Add feature")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "another.txt"), []byte("another\n"), 0600))
	gitTestRun(t, repo, "add", ".")
	gitTestRun(t, repo, "commit", "--quiet", "-m", "Add another file")

	commits, files, commitsTruncated, filesTruncated, err := DraftPullRequest(context.Background(), repo, "main", "feature", 50)
	require.NoError(t, err)
	assert.Equal(t, []string{"Add another file", "Add feature"}, commits)
	assert.Equal(t, []string{"another.txt", "feature.txt"}, files)
	assert.False(t, commitsTruncated)
	assert.False(t, filesTruncated)

	commits, files, commitsTruncated, filesTruncated, err = DraftPullRequest(context.Background(), repo, "main", "feature", 1)
	require.NoError(t, err)
	assert.Equal(t, []string{"Add another file"}, commits)
	assert.Equal(t, []string{"another.txt"}, files)
	assert.True(t, commitsTruncated)
	assert.True(t, filesTruncated)

	commits, files, commitsTruncated, filesTruncated, err = DraftPullRequest(context.Background(), repo, "main", "main", 0)
	require.NoError(t, err)
	assert.Empty(t, commits)
	assert.Empty(t, files)
	assert.False(t, commitsTruncated)
	assert.False(t, filesTruncated)

	_, _, _, _, err = DraftPullRequest(context.Background(), repo, "missing", "feature", 1)
	assert.ErrorContains(t, err, "summarize commits")
}

func TestDraftPullRequestReportsDiffFailureForUnrelatedBranches(t *testing.T) {
	repo := t.TempDir()
	gitTestRun(t, repo, "init", "--quiet")
	gitTestRun(t, repo, "config", "user.email", "test@example.com")
	gitTestRun(t, repo, "config", "user.name", "Test")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "base.txt"), []byte("base"), 0600))
	gitTestRun(t, repo, "add", ".")
	gitTestRun(t, repo, "commit", "--quiet", "-m", "base")
	gitTestRun(t, repo, "branch", "-M", "base")
	gitTestRun(t, repo, "checkout", "--orphan", "unrelated")
	gitTestRun(t, repo, "rm", "-rf", ".")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "unrelated.txt"), []byte("other"), 0600))
	gitTestRun(t, repo, "add", ".")
	gitTestRun(t, repo, "commit", "--quiet", "-m", "unrelated")

	_, _, _, _, err := DraftPullRequest(context.Background(), repo, "base", "unrelated", 10)
	assert.ErrorContains(t, err, "list changed files")
}

func gitTestRun(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, output)
}
