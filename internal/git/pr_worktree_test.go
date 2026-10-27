package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func prWorktreeFixture(t *testing.T) (*WorktreeService, PRWorktreeOptions, string) {
	t.Helper()
	root := worktreeTestRepository(t)
	origin := filepath.Join(t.TempDir(), "origin.git")
	runGit(t, root, "init", "--bare", "--quiet", origin)
	runGit(t, root, "remote", "add", "origin", origin)
	runGit(t, root, "checkout", "-b", "feature")
	runGit(t, root, "commit", "--allow-empty", "-m", "PR work")
	sha := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))
	runGit(t, root, "push", "origin", "HEAD:refs/pull/42/head")
	runGit(t, root, "checkout", "main")
	runGit(t, root, "branch", "-D", "feature")
	return NewWorktreeService(root), PRWorktreeOptions{Operation: "checkout", Path: filepath.Join(t.TempDir(), "PR checkout"), Repository: model.RepositoryRef{Owner: "acme", Name: "project"}, PullRequest: &model.PullRequest{Number: 42, State: "open", Head: model.BranchRef{Ref: "feature", SHA: sha, Label: "acme:feature", Repo: &model.Repository{Name: "project", Owner: model.User{Login: "acme"}}}}}, root
}

func TestPRWorktreeCheckoutPreviewCreateAndResume(t *testing.T) {
	ctx := context.Background()
	service, input, root := prWorktreeFixture(t)
	input.DryRun = true
	plan, err := service.PRWorktree(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, "create", plan.Decision)
	assert.Equal(t, "planned", plan.Fetch)
	assert.NoDirExists(t, input.Path)
	assert.NotContains(t, runGit(t, root, "for-each-ref", "refs/heads"), "feature")
	input.DryRun = false
	result, err := service.PRWorktree(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, "completed", result.State)
	assert.Equal(t, "completed", result.Fetch)
	assert.Equal(t, "main", strings.TrimSpace(runGit(t, root, "branch", "--show-current")))
	assert.Equal(t, "feature", strings.TrimSpace(runGit(t, input.Path, "branch", "--show-current")))
	require.NoError(t, os.WriteFile(filepath.Join(input.Path, "local.txt"), []byte("keep"), 0600))
	result, err = service.PRWorktree(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, "resume", result.Decision)
	assert.Equal(t, "not_needed", result.Fetch)
	assert.Equal(t, "dirty", result.Worktree.StatusState)
	assert.FileExists(t, filepath.Join(input.Path, "local.txt"))
}

func TestPRWorktreeRetirementProtectsWork(t *testing.T) {
	for _, scenario := range []string{"open", "dirty", "ignored", "unpublished", "locked", "closed", "merged", "unknown", "shallow"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			service, input, root := prWorktreeFixture(t)
			_, err := service.PRWorktree(ctx, input)
			require.NoError(t, err)
			input.Operation = "retire"
			input.PullRequest.State = "closed"
			switch scenario {
			case "open":
				input.PullRequest.State = "open"
			case "dirty":
				require.NoError(t, os.WriteFile(filepath.Join(input.Path, "local"), []byte("keep"), 0600))
			case "ignored":
				runGit(t, input.Path, "config", "core.excludesFile", filepath.Join(root, ".git", "ignored"))
				require.NoError(t, os.WriteFile(filepath.Join(root, ".git", "ignored"), []byte("cache\n"), 0600))
				require.NoError(t, os.WriteFile(filepath.Join(input.Path, "cache"), []byte("keep"), 0600))
			case "unpublished":
				runGit(t, input.Path, "commit", "--allow-empty", "-m", "unpublished")
			case "locked":
				runGit(t, root, "worktree", "lock", input.Path)
			case "merged":
				input.PullRequest.MergedAt = "2026-10-25T12:00:00Z"
			case "unknown":
				input.PullRequest.State = "mystery"
			case "shallow":
				require.NoError(t, os.WriteFile(filepath.Join(root, ".git", "shallow"), []byte(input.PullRequest.Head.SHA+"\n"), 0600))
			}
			input.DryRun = true
			plan, err := service.PRWorktree(ctx, input)
			require.NoError(t, err)
			remove := scenario == "closed" || scenario == "merged"
			if remove {
				assert.Equal(t, "remove", plan.Decision)
			} else {
				assert.Equal(t, "keep", plan.Decision)
				assert.NotEmpty(t, plan.Reasons)
			}
			assert.DirExists(t, input.Path)
			input.DryRun = false
			result, err := service.PRWorktree(ctx, input)
			require.NoError(t, err)
			if remove {
				assert.NoDirExists(t, input.Path)
				assert.Equal(t, "completed", result.State)
				assert.Contains(t, runGit(t, root, "branch", "--list"), "feature")
			} else {
				assert.DirExists(t, input.Path)
				assert.Equal(t, "kept", result.State)
			}
		})
	}
}

func TestPRWorktreeRejectsConflictsAndUnrelatedBranches(t *testing.T) {
	ctx := context.Background()
	service, input, root := prWorktreeFixture(t)
	runGit(t, root, "branch", "feature")
	_, err := service.PRWorktree(ctx, input)
	require.ErrorContains(t, err, "unrelated")
	runGit(t, root, "branch", "-D", "feature")
	require.NoError(t, os.Mkdir(input.Path, 0700))
	_, err = service.PRWorktree(ctx, input)
	require.ErrorContains(t, err, "already exists")
	require.NoError(t, os.Remove(input.Path))
	_, err = service.PRWorktree(ctx, input)
	require.NoError(t, err)
	input.Path = filepath.Join(t.TempDir(), "another")
	_, err = service.PRWorktree(ctx, input)
	require.ErrorContains(t, err, "already checked out")
}

func TestPRWorktreeForkAndHeadChanged(t *testing.T) {
	ctx := context.Background()
	service, input, root := prWorktreeFixture(t)
	input.PullRequest.Head.Repo = &model.Repository{Name: "renamed-fork", Owner: model.User{Login: "contributor"}}
	input.PullRequest.Head.Label = "contributor:feature"
	plan, err := service.PRWorktree(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, "pr/42", plan.Branch)
	assert.Equal(t, "contributor/renamed-fork", plan.Push.Repository.String())
	assert.Equal(t, "HEAD:refs/heads/feature", plan.Push.Refspec)
	assert.Empty(t, strings.TrimSpace(runGit(t, input.Path, "for-each-ref", "--format=%(upstream)", "refs/heads/pr/42")))
	// A provider head that no longer matches the fetched PR ref must not create a branch.
	runGit(t, root, "worktree", "remove", input.Path)
	runGit(t, root, "branch", "-D", "pr/42")
	input.PullRequest.Head.SHA = strings.TrimSpace(runGit(t, root, "rev-parse", "main"))
	result, err := service.PRWorktree(ctx, input)
	require.ErrorContains(t, err, "head changed")
	require.NotNil(t, result)
	assert.Equal(t, "completed", result.Fetch)
	assert.Equal(t, "failed", result.State)
	assert.NoDirExists(t, input.Path)
}

func TestPRWorktreeValidatesRequests(t *testing.T) {
	for _, scenario := range []string{"operation", "empty path", "nil PR", "invalid SHA", "empty ref", "empty repo", "bad branch", "not repo", "bare", "branch mismatch", "prunable", "status unavailable", "different binding"} {
		t.Run(scenario, func(t *testing.T) {
			service, input, root := prWorktreeFixture(t)
			input.DryRun = true
			switch scenario {
			case "operation":
				input.Operation = "erase"
			case "empty path":
				input.Path = " "
			case "nil PR":
				input.PullRequest = nil
			case "invalid SHA":
				input.PullRequest.Head.SHA = "--help"
			case "empty ref":
				input.PullRequest.Head.Ref = ""
			case "empty repo":
				input.Repository.Owner = ""
			case "bad branch":
				input.Branch = "bad branch"
			case "not repo":
				service = NewWorktreeService(t.TempDir())
			case "bare":
				bare := filepath.Join(t.TempDir(), "bare.git")
				runGit(t, root, "init", "--bare", bare)
				service = NewWorktreeService(bare)
			case "branch mismatch":
				runGit(t, root, "worktree", "add", "-b", "other", input.Path)
			case "prunable":
				input.DryRun = false
				_, err := service.PRWorktree(context.Background(), input)
				require.NoError(t, err)
				require.NoError(t, os.RemoveAll(input.Path))
				input.DryRun = true
			case "status unavailable":
				input.DryRun = false
				_, err := service.PRWorktree(context.Background(), input)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(input.Path, ".git"), []byte("invalid"), 0600))
				input.DryRun = true
			case "different binding":
				runGit(t, root, "branch", "feature", input.PullRequest.Head.SHA)
				runGit(t, root, "config", "branch.feature.gha-pr", "acme/project#99")
			}
			_, err := service.PRWorktree(context.Background(), input)
			require.Error(t, err)
		})
	}
}

func TestPRWorktreeLocalAheadAndBoundBehindResume(t *testing.T) {
	ctx := context.Background()
	service, input, root := prWorktreeFixture(t)
	_, err := service.PRWorktree(ctx, input)
	require.NoError(t, err)
	runGit(t, input.Path, "commit", "--allow-empty", "-m", "new local work")
	result, err := service.PRWorktree(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, "ahead", result.Comparison.State)
	assert.Equal(t, 1, result.Comparison.AheadBy)
	// Simulate a newer published PR head while preserving the existing checkout.
	runGit(t, root, "branch", "published", input.PullRequest.Head.SHA)
	remoteWork := filepath.Join(t.TempDir(), "remote-work")
	runGit(t, root, "worktree", "add", remoteWork, "published")
	runGit(t, remoteWork, "commit", "--allow-empty", "-m", "published update")
	input.PullRequest.Head.SHA = strings.TrimSpace(runGit(t, remoteWork, "rev-parse", "HEAD"))
	result, err = service.PRWorktree(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, "diverged", result.Comparison.State)
	assert.Equal(t, "resume", result.Decision)
	assert.Equal(t, "not_needed", result.Fetch)
	// An unpublished local tip can never be retired even with a prior binding.
	input.Operation = "retire"
	input.PullRequest.State = "closed"
	result, err = service.PRWorktree(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, "keep", result.Decision)
}

func TestPRWorktreeRetirementPreservesAmbiguity(t *testing.T) {
	for _, scenario := range []string{"missing", "main", "current", "detached", "branch mismatch", "unrelated", "missing history", "binding conflict"} {
		t.Run(scenario, func(t *testing.T) {
			service, input, root := prWorktreeFixture(t)
			_, err := service.PRWorktree(context.Background(), input)
			require.NoError(t, err)
			input.Operation = "retire"
			input.PullRequest.State = "closed"
			switch scenario {
			case "missing":
				input.Path = filepath.Join(t.TempDir(), "missing")
			case "main":
				input.Path = root
			case "current":
				service = NewWorktreeService(input.Path)
			case "detached":
				runGit(t, input.Path, "checkout", "--detach")
			case "branch mismatch":
				input.Branch = "other"
			case "unrelated":
				runGit(t, input.Path, "branch", "-m", "unrelated")
				runGit(t, root, "config", "--unset", "branch.unrelated.gha-pr")
			case "missing history":
				input.PullRequest.Head.SHA = strings.Repeat("f", 40)
			case "binding conflict":
				runGit(t, root, "config", "branch.feature.gha-pr", "acme/project#99")
			}
			result, err := service.PRWorktree(context.Background(), input)
			require.NoError(t, err)
			assert.Equal(t, "keep", result.Decision)
		})
	}
}

func TestPRFetchRepositoryMatchesActualFetchOrigin(t *testing.T) {
	ctx := context.Background()
	root := worktreeTestRepository(t)
	_, err := PRFetchRepository(ctx, root, "")
	require.ErrorContains(t, err, "fetch origin")
	runGit(t, root, "remote", "add", "origin", "https://github.com/acme/project.git")
	repo, err := PRFetchRepository(ctx, root, "")
	require.NoError(t, err)
	assert.Equal(t, "acme/project", repo.String())
	_, err = PRFetchRepository(ctx, root, "ACME/PROJECT")
	require.NoError(t, err)
	_, err = PRFetchRepository(ctx, root, "other/project")
	require.ErrorContains(t, err, "does not match")
	_, err = PRFetchRepository(ctx, root, "bad")
	require.ErrorContains(t, err, "invalid repository")
	runGit(t, root, "remote", "set-url", "origin", "https://other.example/acme/project.git")
	_, err = PRFetchRepository(ctx, root, "")
	require.ErrorContains(t, err, "not a GitHub")
	runGit(t, root, "remote", "set-url", "origin", "https://github.com/acme/project.git")
	runGit(t, root, "config", "--add", "remote.origin.url", "https://github.com/acme/another.git")
	_, err = PRFetchRepository(ctx, root, "")
	require.ErrorContains(t, err, "exactly one")
}

func prGitFailure(t *testing.T, mode string) {
	t.Helper()
	gitExecutable, err := exec.LookPath("git")
	require.NoError(t, err)
	directory := t.TempDir()
	script := `#!/bin/sh
case "$GHA_TEST_PR_FAILURE:$*" in
 fetch:*fetch\ *) echo simulated-fetch-failure >&2; exit 2;;
 add:worktree\ add*) echo simulated-add-failure >&2; exit 2;;
 partial-add:worktree\ add*) "$GHA_TEST_REAL_GIT" branch feature; echo simulated-add-failure >&2; exit 2;;
 bind:config\ branch.*) echo simulated-binding-failure >&2; exit 2;;
 association:config\ --get\ branch.*) echo simulated-association-failure >&2; exit 2;;
 remove:worktree\ remove*) echo simulated-removal-failure >&2; exit 2;;
 inspect:show-ref*) echo simulated-inspection-failure >&2; exit 2;;
esac
exec "$GHA_TEST_REAL_GIT" "$@"
`
	require.NoError(t, os.WriteFile(filepath.Join(directory, "git"), []byte(script), 0700))
	t.Setenv("GHA_TEST_REAL_GIT", gitExecutable)
	t.Setenv("GHA_TEST_PR_FAILURE", mode)
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestPRWorktreeReportsPartialFailures(t *testing.T) {
	for _, mode := range []string{"fetch", "add", "partial-add", "bind", "remove", "association", "inspect"} {
		t.Run(mode, func(t *testing.T) {
			service, input, root := prWorktreeFixture(t)
			if mode == "remove" || mode == "association" {
				_, err := service.PRWorktree(context.Background(), input)
				require.NoError(t, err)
			}
			if mode == "remove" {
				input.Operation = "retire"
				input.PullRequest.State = "closed"
			}
			prGitFailure(t, mode)
			result, err := service.PRWorktree(context.Background(), input)
			if mode == "association" {
				require.ErrorContains(t, err, "association")
				return
			}
			require.Error(t, err)
			if mode == "inspect" {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Equal(t, "failed", result.State)
			if mode == "fetch" {
				assert.Equal(t, "failed", result.Fetch)
			}
			if mode == "partial-add" {
				assert.Equal(t, "completed", result.BranchState)
				assert.NoDirExists(t, input.Path)
			}
			if mode == "bind" {
				assert.Equal(t, "completed", result.WorktreeState)
				assert.DirExists(t, input.Path)
			}
			if mode == "remove" {
				assert.DirExists(t, input.Path)
			}
			assert.Equal(t, "main", strings.TrimSpace(runGit(t, root, "branch", "--show-current")))
		})
	}
}

func TestPRWorktreeReusesUncheckedOutPublishedBranch(t *testing.T) {
	service, input, root := prWorktreeFixture(t)
	runGit(t, root, "branch", "feature", input.PullRequest.Head.SHA)
	result, err := service.PRWorktree(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "not_needed", result.Fetch)
	assert.Equal(t, "retained", result.BranchState)
	assert.DirExists(t, input.Path)
}

func TestPRCheckoutBranchPrefixConflictDoesNotFetch(t *testing.T) {
	service, input, root := prWorktreeFixture(t)
	runGit(t, root, "branch", "feature/other")
	_, err := service.PRWorktree(context.Background(), input)
	require.ErrorContains(t, err, "conflict")
	assert.Empty(t, strings.TrimSpace(runGit(t, root, "for-each-ref", "refs/gha")))
}

func TestPRWorktreeCreationReportsActualExistingBranchHead(t *testing.T) {
	service, input, root := prWorktreeFixture(t)
	runGit(t, root, "branch", "feature", input.PullRequest.Head.SHA)
	runGit(t, root, "checkout", "feature")
	runGit(t, root, "commit", "--allow-empty", "-m", "local unpublished")
	head := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))
	runGit(t, root, "checkout", "main")
	result, err := service.PRWorktree(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, result.Worktree)
	assert.Equal(t, head, result.Worktree.Head)
	assert.Equal(t, "ahead", result.Comparison.State)
}

func TestPRRetirementProtectsHiddenChangesAndInProgressOperations(t *testing.T) {
	for _, mode := range []string{"assume unchanged", "skip worktree", "merge in progress", "rebase in progress"} {
		t.Run(mode, func(t *testing.T) {
			service, input, root := prWorktreeFixture(t)
			// Put one tracked file on the published PR head.
			runGit(t, root, "checkout", "-b", "publish", input.PullRequest.Head.SHA)
			require.NoError(t, os.WriteFile(filepath.Join(root, "tracked"), []byte("published"), 0600))
			runGit(t, root, "add", "tracked")
			runGit(t, root, "commit", "-m", "tracked file")
			input.PullRequest.Head.SHA = strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))
			runGit(t, root, "push", "origin", "+HEAD:refs/pull/42/head")
			runGit(t, root, "checkout", "main")
			_, err := service.PRWorktree(context.Background(), input)
			require.NoError(t, err)
			switch mode {
			case "assume unchanged":
				runGit(t, input.Path, "update-index", "--assume-unchanged", "tracked")
				require.NoError(t, os.WriteFile(filepath.Join(input.Path, "tracked"), []byte("hidden work"), 0600))
			case "skip worktree":
				runGit(t, input.Path, "update-index", "--skip-worktree", "tracked")
				require.NoError(t, os.WriteFile(filepath.Join(input.Path, "tracked"), []byte("hidden work"), 0600))
			case "merge in progress":
				path := strings.TrimSpace(runGit(t, input.Path, "rev-parse", "--git-path", "MERGE_HEAD"))
				require.NoError(t, os.WriteFile(path, []byte(input.PullRequest.Head.SHA+"\n"), 0600))
			case "rebase in progress":
				path := strings.TrimSpace(runGit(t, input.Path, "rev-parse", "--git-path", "rebase-merge"))
				require.NoError(t, os.Mkdir(path, 0700))
			}
			input.Operation = "retire"
			input.PullRequest.State = "closed"
			input.DryRun = true
			result, err := service.PRWorktree(context.Background(), input)
			require.NoError(t, err)
			assert.Equal(t, "keep", result.Decision)
			assert.DirExists(t, input.Path)
		})
	}
}

func TestPRResumeRecordsAssociationAndRecoversFailedBinding(t *testing.T) {
	service, input, root := prWorktreeFixture(t)
	runGit(t, root, "worktree", "add", "-b", "feature", input.Path, input.PullRequest.Head.SHA)
	input.DryRun = true
	result, err := service.PRWorktree(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "planned", result.AssociationState)
	input.DryRun = false
	result, err = service.PRWorktree(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "completed", result.AssociationState)
	assert.Equal(t, "acme/project#42", strings.TrimSpace(runGit(t, root, "config", "--get", "branch.feature.gha-pr")))
	result, err = service.PRWorktree(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "not_needed", result.AssociationState)
}
