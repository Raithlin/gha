package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestCleanupWorktreePreviewPreservesUnsafeContent(t *testing.T) {
	for _, kind := range []string{"clean", "dirty", "untracked", "ignored", "locked", "unpublished", "primary", "current"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			runGit(t, root, "init", "-b", "main")
			runGit(t, root, "config", "user.name", "Test")
			runGit(t, root, "config", "user.email", "test@example.com")
			require.NoError(t, os.WriteFile(filepath.Join(root, "tracked"), []byte("original\n"), 0600))
			require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte("ignored\n"), 0600))
			runGit(t, root, "add", ".")
			runGit(t, root, "commit", "-m", "initial")
			runGit(t, root, "branch", "topic")
			path := filepath.Join(t.TempDir(), "linked")
			runGit(t, root, "worktree", "add", path, "topic")
			sha := strings.TrimSpace(runGit(t, root, "rev-parse", "topic"))
			repo := model.RepositoryRef{Owner: "acme", Name: "project"}
			pr := &model.PullRequest{Number: 7, State: "closed", MergedAt: "2026-01-01T00:00:00Z", Head: model.BranchRef{Ref: "topic", SHA: sha, Repo: &model.Repository{Owner: model.User{Login: "acme"}, Name: "project"}}}
			source := root
			branch := "topic"
			switch kind {
			case "dirty":
				require.NoError(t, os.WriteFile(filepath.Join(path, "tracked"), []byte("edited"), 0600))
			case "untracked", "ignored":
				require.NoError(t, os.WriteFile(filepath.Join(path, kind), []byte("keep"), 0600))
			case "locked":
				runGit(t, root, "worktree", "lock", path)
			case "unpublished":
				runGit(t, path, "commit", "--allow-empty", "-m", "local work")
			case "primary":
				path = root
				branch = "main"
			case "current":
				source = path
			}
			result, err := NewBranchLister(source).PlanCleanupWorktree(context.Background(), path, branch, repo, pr)
			require.NoError(t, err)
			require.True(t, result.DryRun)
			if kind == "clean" {
				require.Equal(t, "remove", result.Decision)
			} else {
				require.Equal(t, "keep", result.Decision)
				require.NotEmpty(t, result.Reasons)
			}
			require.DirExists(t, path)
			require.Equal(t, sha, strings.TrimSpace(runGit(t, root, "rev-parse", "main")))
		})
	}
}
