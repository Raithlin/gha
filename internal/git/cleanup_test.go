package git

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchListerCleanupUsesCachedOriginDefaultWhenBaseIsOmitted(t *testing.T) {
	workdir := t.TempDir()
	runGit(t, workdir, "init", "-b", "main")
	runGit(t, workdir, "config", "user.email", "test@example.com")
	runGit(t, workdir, "config", "user.name", "Test User")
	runGit(t, workdir, "commit", "--allow-empty", "-m", "initial")
	mainSHA := strings.TrimSpace(runGit(t, workdir, "rev-parse", "main"))
	runGit(t, workdir, "branch", "feature/merged")
	runGit(t, workdir, "remote", "add", "origin", "https://example.com/acme/project.git")
	runGit(t, workdir, "update-ref", "refs/remotes/origin/main", mainSHA)
	runGit(t, workdir, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")

	cleanup, err := NewBranchLister(workdir).Cleanup(context.Background(), "", 30)

	require.NoError(t, err)
	assert.Equal(t, "main", cleanup.Base)
	require.Len(t, cleanup.Candidates, 1)
	assert.Equal(t, "feature/merged", cleanup.Candidates[0].Name)
}

func TestBranchListerCleanupClassifiesBaseCurrentAndUnmergedBranches(t *testing.T) {
	workdir := t.TempDir()
	runGit(t, workdir, "init", "-b", "main")
	runGit(t, workdir, "config", "user.email", "test@example.com")
	runGit(t, workdir, "config", "user.name", "Test User")
	runGit(t, workdir, "commit", "--allow-empty", "-m", "initial")
	runGit(t, workdir, "branch", "merged")
	runGit(t, workdir, "checkout", "-b", "active")
	runGit(t, workdir, "commit", "--allow-empty", "-m", "active")
	runGit(t, workdir, "checkout", "main")
	runGit(t, workdir, "checkout", "-b", "current")

	cleanup, err := NewBranchLister(workdir).Cleanup(context.Background(), "main", 30)
	require.NoError(t, err)
	assert.Equal(t, []string{"merged"}, []string{cleanup.Candidates[0].Name})
	assert.Len(t, cleanup.Excluded, 3)
	assert.Contains(t, []string{cleanup.Excluded[0].Reason, cleanup.Excluded[1].Reason, cleanup.Excluded[2].Reason}, "base_branch")
	assert.Contains(t, []string{cleanup.Excluded[0].Reason, cleanup.Excluded[1].Reason, cleanup.Excluded[2].Reason}, "current_branch")
	assert.Contains(t, []string{cleanup.Excluded[0].Reason, cleanup.Excluded[1].Reason, cleanup.Excluded[2].Reason}, "not_reachable_from_base")
}

func TestBranchListerCleanupValidatesLimitsAndBase(t *testing.T) {
	workdir := t.TempDir()
	runGit(t, workdir, "init", "-b", "main")
	for _, limit := range []int{0, 101} {
		_, err := NewBranchLister(workdir).Cleanup(context.Background(), "main", limit)
		assert.ErrorContains(t, err, "limit must be between 1 and 100")
	}
	_, err := NewBranchLister(workdir).Cleanup(context.Background(), "missing", 1)
	assert.ErrorContains(t, err, "is not a local branch")
	_, err = NewBranchLister(workdir).Cleanup(context.Background(), "", 1)
	assert.ErrorContains(t, err, "resolve cleanup base")
}
