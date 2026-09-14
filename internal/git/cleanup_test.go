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
