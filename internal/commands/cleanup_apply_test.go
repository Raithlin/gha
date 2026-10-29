package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raithlin/gha/internal/branch"
	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/interfaces"
	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/require"
)

type cleanupApplyProvider struct {
	mutationSafetyProvider
	repo  *model.Repository
	pr    *model.PullRequest
	onGet func()
}

func (p *cleanupApplyProvider) GetRepository(context.Context, string, string) (*model.Repository, error) {
	return p.repo, nil
}
func (p *cleanupApplyProvider) ListPullRequests(context.Context, string, string, interfaces.ListPRsOptions) ([]*model.PullRequest, error) {
	return []*model.PullRequest{p.pr}, nil
}
func (p *cleanupApplyProvider) GetPullRequest(context.Context, string, string, int) (*model.PullRequest, error) {
	if p.onGet != nil {
		p.onGet()
	}
	return p.pr, nil
}

func cleanupApplyFixture(t *testing.T) (string, *cleanupApplyProvider) {
	t.Helper()
	root := commandWorktreeRepository(t)
	runCommandGit(t, root, "remote", "add", "origin", "https://github.com/acme/project.git")
	runCommandGit(t, root, "checkout", "-b", "topic")
	runCommandGit(t, root, "commit", "--allow-empty", "-m", "PR head")
	sha := strings.TrimSpace(runCommandGit(t, root, "rev-parse", "HEAD"))
	runCommandGit(t, root, "checkout", "main")
	repo := &model.Repository{Owner: model.User{Login: "acme"}, Name: "project", DefaultBranch: "main"}
	return root, &cleanupApplyProvider{repo: repo, pr: &model.PullRequest{Number: 7, State: "closed", MergedAt: "2026-01-01T00:00:00Z", Head: model.BranchRef{Ref: "topic", SHA: sha, Repo: repo}, Base: model.BranchRef{Ref: "main", Repo: repo}}}
}
func executeCleanupApply(t *testing.T, root string, p *cleanupApplyProvider, input string, args ...string) (model.CleanupExecution, string, error) {
	t.Helper()
	cmd := newBranchesCmd(branch.NewService(git.NewBranchLister(root), p))
	var out, diagnostics bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&diagnostics)
	cmd.SetIn(strings.NewReader(input))
	cmd.SetArgs(append([]string{"cleanup", "apply", "topic", "--path", root, "--base", "main", "--format", "json"}, args...))
	err := cmd.Execute()
	var result model.CleanupExecution
	if out.Len() > 0 {
		require.NoError(t, json.Unmarshal(out.Bytes(), &result))
	}
	return result, diagnostics.String(), err
}
func TestCleanupApplyVerifiedSquashHeadNeedsNoForce(t *testing.T) {
	root, p := cleanupApplyFixture(t)
	plan, diagnostics, err := executeCleanupApply(t, root, p, "", "--local", "--dry-run")
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Equal(t, "planned", plan.Results[0].Local)
	result, _, err := executeCleanupApply(t, root, p, "", "--local")
	require.NoError(t, err)
	require.Equal(t, "completed", result.Results[0].Local)
	require.False(t, result.Results[0].Forced)
}
func TestCleanupForceRequiresSpecificNameConfirmation(t *testing.T) {
	for _, answer := range []string{"", "yes\n", "other\n", "topic\n"} {
		t.Run(strings.TrimSpace(answer), func(t *testing.T) {
			root, p := cleanupApplyFixture(t)
			p.pr.MergedAt = ""
			path := filepath.Join(t.TempDir(), "linked")
			runCommandGit(t, root, "worktree", "add", path, "topic")
			require.NoError(t, os.WriteFile(filepath.Join(path, "untracked"), []byte("discard only after confirmation"), 0600))
			result, diagnostics, err := executeCleanupApply(t, root, p, answer, "--local", "--worktree", path, "--force", "topic")
			require.Contains(t, diagnostics, "topic")
			require.Contains(t, diagnostics, path)
			require.Contains(t, diagnostics, "untracked")
			if answer == "topic\n" {
				require.NoError(t, err)
				require.Equal(t, "completed", result.Results[0].Local)
				require.NoDirExists(t, path)
			} else {
				require.Error(t, err)
				require.DirExists(t, path)
				require.NotEmpty(t, runCommandGit(t, root, "rev-parse", "topic"))
			}
		})
	}
}
func TestCleanupForceDryRunDoesNotPromptOrDelete(t *testing.T) {
	root, p := cleanupApplyFixture(t)
	p.pr.MergedAt = ""
	result, diagnostics, err := executeCleanupApply(t, root, p, "", "--local", "--force", "topic", "--dry-run")
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.True(t, result.Results[0].Forced)
	require.Equal(t, "planned", result.Results[0].Local)
	_, _, err = executeCleanupApply(t, root, p, "topic\n", "--local", "--force", "other")
	require.ErrorContains(t, err, "selected branch")
}
func TestCleanupApplyRetainsUnmergedAndChangedHeads(t *testing.T) {
	root, p := cleanupApplyFixture(t)
	p.pr.State = "open"
	result, _, err := executeCleanupApply(t, root, p, "", "--local")
	require.NoError(t, err)
	require.Equal(t, "retained", result.Results[0].State)
	p.pr.State = "closed"
	calls := 0
	p.onGet = func() {
		calls++
		if calls == 2 {
			runCommandGit(t, root, "branch", "-f", "topic", "main")
		}
	}
	result, _, err = executeCleanupApply(t, root, p, "", "--local")
	require.Error(t, err)
	require.Equal(t, "retained", result.Results[0].Local)
}

func TestCleanupApplyValidatesSelectionsBeforeWriting(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"target", nil, "select --local"},
		{"force", []string{"--local", "--force", "other"}, "selected branch"},
		{"worktree", []string{"--local", "--worktree", "/missing"}, "does not belong"},
		{"origin worktree", []string{"--origin", "--worktree", "/missing"}, "requires --local"},
		{"format", []string{"--local", "--format", "bad"}, "format"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, p := cleanupApplyFixture(t)
			_, _, err := executeCleanupApply(t, root, p, "", tc.args...)
			require.ErrorContains(t, err, tc.want)
			require.NotEmpty(t, runCommandGit(t, root, "rev-parse", "topic"))
		})
	}
}

func TestCleanupForceAllConfirmationsPrecedeAnyWrites(t *testing.T) {
	root, p := cleanupApplyFixture(t)
	runCommandGit(t, root, "branch", "second", "topic")
	cmd := newBranchesCmd(branch.NewService(git.NewBranchLister(root), p))
	var out, diagnostics bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&diagnostics)
	cmd.SetIn(strings.NewReader("topic\nwrong\n"))
	cmd.SetArgs([]string{"cleanup", "apply", "topic", "second", "--local", "--force", "topic", "--force", "second", "--path", root, "--base", "main", "--format", "json"})
	require.Error(t, cmd.Execute())
	require.NotEmpty(t, runCommandGit(t, root, "rev-parse", "topic"))
	require.NotEmpty(t, runCommandGit(t, root, "rev-parse", "second"))
	var result model.CleanupExecution
	require.NoError(t, json.Unmarshal(out.Bytes(), &result))
	require.Len(t, result.Results, 2)
	for _, branch := range result.Results {
		require.Equal(t, "retained", branch.Local)
	}
}

func TestCleanupForceRemainsScopedToItsNamedBranch(t *testing.T) {
	root, p := cleanupApplyFixture(t)
	p.pr.State = "open"
	runCommandGit(t, root, "branch", "second", "topic")
	cmd := newBranchesCmd(branch.NewService(git.NewBranchLister(root), p))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader("topic\n"))
	cmd.SetArgs([]string{"cleanup", "apply", "topic", "second", "--local", "--force", "topic", "--path", root, "--base", "main", "--format", "json"})
	require.NoError(t, cmd.Execute())
	var result model.CleanupExecution
	require.NoError(t, json.Unmarshal(out.Bytes(), &result))
	require.Equal(t, "completed", result.Results[0].Local)
	require.Equal(t, "retained", result.Results[1].Local)
	require.False(t, result.Results[1].Forced)
	require.NotEmpty(t, runCommandGit(t, root, "rev-parse", "second"))
}

func TestCleanupApplyRetiresCleanSelectedWorktree(t *testing.T) {
	root, p := cleanupApplyFixture(t)
	path := filepath.Join(t.TempDir(), "linked")
	runCommandGit(t, root, "worktree", "add", path, "topic")
	result, _, err := executeCleanupApply(t, root, p, "", "--local", "--worktree", path)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Results[0].Worktree)
	require.Equal(t, "completed", result.Results[0].Local)
	require.NoDirExists(t, path)
}

func cleanupApplyOrigin(t *testing.T, root string, p *cleanupApplyProvider) string {
	t.Helper()
	remote := filepath.Join(t.TempDir(), "origin.git")
	runCommandGit(t, root, "init", "--bare", remote)
	// Keep provider identity canonical while routing this test's push transport locally.
	runCommandGit(t, root, "config", "remote.origin.pushurl", "ssh://git@github.com/acme/project.git")
	runCommandGit(t, root, "config", "url."+remote+".insteadOf", "ssh://git@github.com/acme/project.git")
	runCommandGit(t, root, "push", "origin", "topic")
	yes, no := true, false
	p.safety = model.BranchSafety{Permissions: model.ProviderSignal{State: "available"}, CanPush: &yes, DefaultBranch: model.ProviderSignal{State: "available"}, IsDefault: &no, DefaultBranchName: "main", Protection: model.ProviderSignal{State: "available"}, Protected: &no}
	return remote
}

func TestCleanupApplyOriginAndPartialFailure(t *testing.T) {
	for _, reject := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "partial failure"}[reject], func(t *testing.T) {
			root, p := cleanupApplyFixture(t)
			remote := cleanupApplyOrigin(t, root, p)
			path := filepath.Join(t.TempDir(), "linked")
			runCommandGit(t, root, "worktree", "add", path, "topic")
			if reject {
				require.NoError(t, os.WriteFile(filepath.Join(remote, "hooks", "pre-receive"), []byte("#!/bin/sh\nexit 1\n"), 0700))
			}
			result, diagnostics, err := executeCleanupApply(t, root, p, "", "--local", "--origin", "--worktree", path)
			require.NoDirExists(t, path)
			require.Equal(t, "completed", result.Results[0].Worktree)
			if reject {
				require.Error(t, err)
				require.Contains(t, diagnostics, "branch_cleanup_failed")
				require.Equal(t, "failed", result.Results[0].Origin)
				require.Equal(t, "retained", result.Results[0].Local)
				require.NotEmpty(t, result.Results[0].Recovery)
			} else {
				require.NoError(t, err)
				require.Equal(t, "completed", result.Results[0].Origin)
				require.Equal(t, "completed", result.Results[0].Local)
			}
		})
	}
}

func TestCleanupForceCannotBypassOriginPolicy(t *testing.T) {
	for _, kind := range []string{"permission", "default", "protection", "wrong target"} {
		t.Run(kind, func(t *testing.T) {
			root, p := cleanupApplyFixture(t)
			cleanupApplyOrigin(t, root, p)
			switch kind {
			case "permission":
				p.safety.CanPush = boolPointer(false)
			case "default":
				p.safety.IsDefault = boolPointer(true)
			case "protection":
				p.safety.Protected = boolPointer(true)
			case "wrong target":
				runCommandGit(t, root, "config", "remote.origin.pushurl", "https://github.com/other/project.git")
			}
			result, diagnostics, err := executeCleanupApply(t, root, p, "topic\n", "--origin", "--force", "topic")
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			require.Equal(t, "retained", result.Results[0].Origin)
			require.NotEmpty(t, result.Results[0].Reason)
		})
	}
}

func TestCleanupApplyRejectsMissingOrDuplicateBranches(t *testing.T) {
	for _, names := range [][]string{nil, {"topic", "topic"}, {"missing"}} {
		root, p := cleanupApplyFixture(t)
		cmd := newCleanupApplyCmd(branch.NewService(git.NewBranchLister(root), p))
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(append(names, "--local", "--path", root, "--base", "main"))
		require.Error(t, cmd.Execute())
	}
	root, p := cleanupApplyFixture(t)
	runCommandGit(t, root, "remote", "remove", "origin")
	_, _, err := executeCleanupApply(t, root, p, "", "--local")
	require.Error(t, err)
	root, p = cleanupApplyFixture(t)
	p.repo.DefaultBranch = ""
	result, _, err := executeCleanupApply(t, root, p, "topic\n", "--local", "--force", "topic")
	require.NoError(t, err)
	require.Equal(t, "retained", result.Results[0].Local)
}

func TestCleanupRevalidationRetainsPlanWhenOriginIdentityChanges(t *testing.T) {
	root, p := cleanupApplyFixture(t)
	cmd := newCleanupApplyCmd(branch.NewService(git.NewBranchLister(root), p))
	cmd.SetContext(context.Background())
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	app, err := prepareCleanupApplication(cmd, branch.NewService(git.NewBranchLister(root), p), []string{"topic"}, cleanupApplyOptions{path: root, base: "main", local: true})
	require.NoError(t, err)
	runCommandGit(t, root, "remote", "set-url", "origin", "https://github.com/other/project.git")
	require.Error(t, app.execute(cmd))
	require.Equal(t, "retained", app.plan.Results[0].Local)
}
