package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func cleanupDeleteFixture(t *testing.T) (string, CleanupDeleteOptions) {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "-b", "main")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "commit", "--allow-empty", "-m", "initial")
	runGit(t, root, "checkout", "-b", "topic")
	runGit(t, root, "commit", "--allow-empty", "-m", "unreachable PR head")
	sha := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))
	runGit(t, root, "checkout", "main")
	return root, CleanupDeleteOptions{Branch: "topic", ExpectedSHA: sha, Base: "main", DefaultBranch: "main", Local: true}
}

func TestCleanupDeleteUsesExactVerifiedTip(t *testing.T) {
	root, input := cleanupDeleteFixture(t)
	plan, err := NewBranchWriter(root).PlanCleanupDelete(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, "planned", plan.Local)
	result, err := NewBranchWriter(root).CleanupDelete(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Local)
	_, err = NewBranchLister(root).Inspect(context.Background(), "topic")
	require.Error(t, err)
	require.Equal(t, input.ExpectedSHA, strings.TrimSpace(runGit(t, root, "rev-parse", input.ExpectedSHA)))
}

func TestCleanupDeleteRejectsChangedHeadAndProtectedBranches(t *testing.T) {
	for _, kind := range []string{"changed", "base", "default", "current", "primary", "locked", "unselected", "dirty"} {
		t.Run(kind, func(t *testing.T) {
			root, input := cleanupDeleteFixture(t)
			switch kind {
			case "changed":
				input.ExpectedSHA = strings.Repeat("b", 40)
			case "base":
				input.Base = "topic"
			case "default":
				input.DefaultBranch = "topic"
			case "current":
				runGit(t, root, "checkout", "topic")
			case "primary":
				runGit(t, root, "checkout", "topic")
				other := filepath.Join(t.TempDir(), "source")
				runGit(t, root, "worktree", "add", other, "main")
				root = other
			default:
				path := filepath.Join(t.TempDir(), "linked")
				runGit(t, root, "worktree", "add", path, "topic")
				if kind != "unselected" {
					input.WorktreePath = path
				}
				if kind == "locked" {
					runGit(t, root, "worktree", "lock", path)
				}
				if kind == "dirty" {
					require.NoError(t, os.WriteFile(filepath.Join(path, "untracked"), []byte("keep"), 0600))
				}
			}
			if kind != "dirty" {
				input.Force = true
			}
			_, err := NewBranchWriter(root).PlanCleanupDelete(context.Background(), input)
			require.Error(t, err)
			require.NotEmpty(t, runGit(t, root, "rev-parse", "topic"))
		})
	}
}

func TestForcedCleanupRemovesOnlySelectedLinkedWorktree(t *testing.T) {
	root, input := cleanupDeleteFixture(t)
	path := filepath.Join(t.TempDir(), "linked")
	runGit(t, root, "worktree", "add", path, "topic")
	require.NoError(t, os.WriteFile(filepath.Join(path, "untracked"), []byte("discard explicitly"), 0600))
	input.WorktreePath = path
	input.Force = true
	plan, err := NewBranchWriter(root).PlanCleanupDelete(context.Background(), input)
	require.NoError(t, err)
	require.True(t, plan.Forced)
	require.DirExists(t, path)
	result, err := NewBranchWriter(root).CleanupDelete(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Worktree)
	require.Equal(t, "completed", result.Local)
	require.NoDirExists(t, path)
}

func TestCleanupOriginEffectsAndPartialFailure(t *testing.T) {
	for _, kind := range []string{"success", "absent", "rejected", "lease", "different_tip"} {
		t.Run(kind, func(t *testing.T) {
			root, input := cleanupDeleteFixture(t)
			remote := filepath.Join(t.TempDir(), "origin.git")
			runGit(t, root, "init", "--bare", remote)
			runGit(t, root, "remote", "add", "origin", remote)
			if kind != "absent" {
				runGit(t, root, "push", "origin", "topic")
			}
			input.Origin = true
			if kind == "different_tip" {
				runGit(t, remote, "update-ref", "refs/heads/topic", strings.TrimSpace(runGit(t, root, "rev-parse", "main")))
				input.Force = true
			}
			writer := NewBranchWriter(root)
			plan, err := writer.PlanCleanupDelete(context.Background(), input)
			require.NoError(t, err)
			input.ExpectedOriginSHA = &plan.OriginSHA
			if kind == "rejected" {
				require.NoError(t, os.WriteFile(filepath.Join(remote, "hooks", "pre-receive"), []byte("#!/bin/sh\nexit 1\n"), 0700))
				path := filepath.Join(t.TempDir(), "linked")
				runGit(t, root, "worktree", "add", path, "topic")
				input.WorktreePath = path
			}
			if kind == "lease" {
				changed := strings.TrimSpace(runGit(t, root, "rev-parse", "main"))
				// The pre-push hook advances origin after ls-remote but before its lease is applied.
				hook := "#!/bin/sh\ngit -C '" + remote + "' update-ref refs/heads/topic " + changed + "\n"
				require.NoError(t, os.WriteFile(filepath.Join(root, ".git", "hooks", "pre-push"), []byte(hook), 0700))
			}
			result, err := writer.CleanupDelete(context.Background(), input)
			if kind == "rejected" || kind == "lease" {
				require.Error(t, err)
				require.Equal(t, "failed", result.Origin)
				require.Equal(t, "retained", result.Local)
				require.NotEmpty(t, result.Recovery)
				require.Equal(t, input.ExpectedSHA, strings.TrimSpace(runGit(t, root, "rev-parse", "topic")))
				if kind == "rejected" {
					require.Equal(t, "completed", result.Worktree)
					require.NoDirExists(t, input.WorktreePath)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, "completed", result.Local)
				if kind == "absent" {
					require.Equal(t, "absent", result.Origin)
				} else {
					require.Equal(t, "completed", result.Origin)
				}
			}
		})
	}
}

func TestCleanupOriginBindsTipAndPushTarget(t *testing.T) {
	root, input := cleanupDeleteFixture(t)
	remote := filepath.Join(t.TempDir(), "origin.git")
	runGit(t, root, "init", "--bare", remote)
	runGit(t, root, "remote", "add", "origin", remote)
	runGit(t, root, "push", "origin", "topic")
	input.Origin = true
	writer := NewBranchWriter(root)
	expected := ""
	input.ExpectedOriginSHA = &expected
	_, err := writer.PlanCleanupDelete(context.Background(), input)
	require.ErrorContains(t, err, "origin tip changed")
	input.ExpectedOriginSHA = nil
	input.PushURL = "different"
	_, err = writer.PlanCleanupDelete(context.Background(), input)
	require.ErrorContains(t, err, "push target changed")
	input.PushURL = ""
	input.ExpectedSHA = strings.TrimSpace(runGit(t, root, "rev-parse", "main"))
	runGit(t, root, "branch", "-f", "topic", "main")
	_, err = writer.PlanCleanupDelete(context.Background(), input)
	require.ErrorContains(t, err, "origin tip differs")
}

func TestCleanupRemovesBranchConfiguration(t *testing.T) {
	root, input := cleanupDeleteFixture(t)
	runGit(t, root, "config", "branch.topic.gha-pr", "acme/project#7")
	result, err := NewBranchWriter(root).CleanupDelete(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Local)
	output := runGit(t, root, "config", "--local", "--list")
	require.NotContains(t, output, "branch.topic")
}

func TestCleanupDeletionFailuresPreserveObservedEffects(t *testing.T) {
	for _, kind := range []string{"ref lock", "config lock"} {
		t.Run(kind, func(t *testing.T) {
			root, input := cleanupDeleteFixture(t)
			lock := filepath.Join(root, ".git", "refs", "heads", "topic.lock")
			if kind == "config lock" {
				runGit(t, root, "config", "branch.topic.gha-pr", "acme/project#7")
				lock = filepath.Join(root, ".git", "config.lock")
			}
			require.NoError(t, os.WriteFile(lock, []byte("held by another operation"), 0600))
			result, err := NewBranchWriter(root).CleanupDelete(context.Background(), input)
			require.Error(t, err)
			require.Equal(t, "failed", result.State)
			require.NotEmpty(t, result.Recovery)
			if kind == "ref lock" {
				require.Equal(t, "failed", result.Local)
				require.NotEmpty(t, runGit(t, root, "rev-parse", "topic"))
			} else {
				require.Equal(t, "completed", result.Local)
				require.Contains(t, result.Reason, "configuration cleanup failed")
			}
		})
	}
}

func TestCleanupPlanRejectsIncompleteOrUnsafeTargets(t *testing.T) {
	for _, kind := range []string{"no target", "bad name", "worktree without local", "wrong worktree", "origin worktree", "missing origin", "multiple origins", "unavailable origin", "in progress"} {
		t.Run(kind, func(t *testing.T) {
			root, input := cleanupDeleteFixture(t)
			switch kind {
			case "no target":
				input.Local = false
			case "bad name":
				input.Branch = "bad name"
			case "wrong worktree":
				input.WorktreePath = filepath.Join(t.TempDir(), "missing")
			case "worktree without local":
				input.Local = false
				input.Origin = true
				input.WorktreePath = "missing"
			case "origin worktree":
				path := filepath.Join(t.TempDir(), "linked")
				runGit(t, root, "worktree", "add", path, "topic")
				input.Local = false
				input.Origin = true
			case "missing origin":
				input.Origin = true
			case "multiple origins":
				input.Origin = true
				runGit(t, root, "remote", "add", "origin", root)
				runGit(t, root, "config", "--add", "remote.origin.pushurl", root)
				runGit(t, root, "config", "--add", "remote.origin.pushurl", root+"/other")
			case "unavailable origin":
				input.Origin = true
				runGit(t, root, "remote", "add", "origin", filepath.Join(t.TempDir(), "missing.git"))
			case "in progress":
				path := filepath.Join(t.TempDir(), "linked")
				runGit(t, root, "worktree", "add", path, "topic")
				input.WorktreePath = path
				input.Force = true
				gitPath := strings.TrimSpace(runGit(t, path, "rev-parse", "--git-path", "MERGE_HEAD"))
				require.NoError(t, os.WriteFile(gitPath, []byte(input.ExpectedSHA), 0600))
			}
			_, err := NewBranchWriter(root).PlanCleanupDelete(context.Background(), input)
			require.Error(t, err)
			if kind == "no target" {
				_, err = NewBranchWriter(root).CleanupDelete(context.Background(), input)
				require.Error(t, err)
			}
		})
	}
}
