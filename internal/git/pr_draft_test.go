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

	commits, files, commitsTruncated, filesTruncated, err := DraftPullRequest(context.Background(), repo, "main", "feature", 50)
	require.NoError(t, err)
	assert.Equal(t, []string{"Add feature"}, commits)
	assert.Equal(t, []string{"feature.txt"}, files)
	assert.False(t, commitsTruncated)
	assert.False(t, filesTruncated)
}

func gitTestRun(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, output)
}
