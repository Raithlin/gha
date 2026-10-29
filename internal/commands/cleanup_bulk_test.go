package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

type bulkCleanupProvider struct {
	*cleanupApplyProvider
	records     map[string]*model.PullRequest
	safetyReads int
}

func (p *bulkCleanupProvider) ListPullRequests(_ context.Context, owner, _ string, options interfaces.ListPRsOptions) ([]*model.PullRequest, error) {
	if pr := p.records[strings.TrimPrefix(options.Head, owner+":")]; pr != nil {
		return []*model.PullRequest{pr}, nil
	}
	return []*model.PullRequest{}, nil
}
func (p *bulkCleanupProvider) GetPullRequest(_ context.Context, _, _ string, number int) (*model.PullRequest, error) {
	for _, pr := range p.records {
		if pr.Number == number {
			return pr, nil
		}
	}
	return nil, fmt.Errorf("unknown PR")
}
func (p *bulkCleanupProvider) InspectBranchSafety(ctx context.Context, repo model.RepositoryRef, name string) (model.BranchSafety, error) {
	p.safetyReads++
	return p.cleanupApplyProvider.InspectBranchSafety(ctx, repo, name)
}
func (p *bulkCleanupProvider) addBranch(t *testing.T, root, name string, number int) {
	t.Helper()
	runCommandGit(t, root, "branch", name, "topic")
	pr := *p.pr
	pr.Number = number
	pr.Head.Ref = name
	p.records[name] = &pr
}
func runBulkCleanup(t *testing.T, root string, p *bulkCleanupProvider, args ...string) (model.CleanupExecution, error) {
	t.Helper()
	cmd := newBranchesCmd(branch.NewService(git.NewBranchLister(root), p))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs(append([]string{"cleanup", "apply", "--path", root, "--base", "main", "--format", "json"}, args...))
	err := cmd.Execute()
	var result model.CleanupExecution
	if out.Len() > 0 {
		require.NoError(t, json.Unmarshal(out.Bytes(), &result))
	}
	return result, err
}
func bulkResult(t *testing.T, result model.CleanupExecution, name string) *model.CleanupBranchResult {
	t.Helper()
	for _, entry := range result.Results {
		if entry.Name == name {
			return entry
		}
	}
	t.Fatalf("missing branch %s", name)
	return nil
}

func TestCleanupNamedBatchContinuesAfterFirstRemoval(t *testing.T) {
	root, base := cleanupApplyFixture(t)
	p := &bulkCleanupProvider{cleanupApplyProvider: base, records: map[string]*model.PullRequest{"topic": base.pr}}
	p.addBranch(t, root, "second", 8)
	result, err := runBulkCleanup(t, root, p, "topic", "second", "--local")
	require.NoError(t, err)
	require.Equal(t, "completed", bulkResult(t, result, "topic").Local)
	require.Equal(t, "completed", bulkResult(t, result, "second").Local)
}

func TestCleanupBulkLocalBranchesAndWorktreesWithDeletedOriginHeads(t *testing.T) {
	root, base := cleanupApplyFixture(t)
	remote := cleanupApplyOrigin(t, root, base)
	runCommandGit(t, root, "push", "origin", ":refs/heads/topic")
	require.Empty(t, strings.TrimSpace(runCommandGit(t, remote, "for-each-ref", "--format=%(refname)", "refs/heads")))
	p := &bulkCleanupProvider{cleanupApplyProvider: base, records: map[string]*model.PullRequest{"topic": base.pr}}
	p.addBranch(t, root, "second", 8)
	p.addBranch(t, root, "active", 9)
	p.records["active"].State = "open"
	p.records["active"].MergedAt = ""
	p.addBranch(t, root, "leftover", 10)
	clean := filepath.Join(t.TempDir(), "clean")
	dirty := filepath.Join(t.TempDir(), "dirty")
	runCommandGit(t, root, "worktree", "add", clean, "second")
	runCommandGit(t, root, "worktree", "add", dirty, "leftover")
	require.NoError(t, os.WriteFile(filepath.Join(dirty, "keep.txt"), []byte("local work"), 0600))
	before := runCommandGit(t, root, "show-ref")
	plan, err := runBulkCleanup(t, root, p, "--all", "--local", "--dry-run")
	require.NoError(t, err)
	require.True(t, plan.DryRun)
	require.Equal(t, "all", plan.Selection)
	require.Len(t, plan.Results, 5)
	require.Equal(t, "planned", bulkResult(t, plan, "second").Worktree)
	require.Equal(t, "retained", bulkResult(t, plan, "leftover").Local)
	require.Equal(t, before, runCommandGit(t, root, "show-ref"))
	require.DirExists(t, clean)
	require.FileExists(t, filepath.Join(dirty, "keep.txt"))
	result, err := runBulkCleanup(t, root, p, "--all", "--local")
	require.NoError(t, err)
	for _, name := range []string{"topic", "second"} {
		require.Equal(t, "completed", bulkResult(t, result, name).Local)
	}
	for _, name := range []string{"main", "active", "leftover"} {
		require.Equal(t, "retained", bulkResult(t, result, name).Local)
	}
	for _, entry := range result.Results {
		require.Equal(t, "not_requested", entry.Origin)
	}
	require.NoDirExists(t, clean)
	require.FileExists(t, filepath.Join(dirty, "keep.txt"))
	require.Equal(t, 0, p.safetyReads)
}

func TestCleanupAlreadyDeletedOriginDoesNotBlockProvenLocalMerge(t *testing.T) {
	root, base := cleanupApplyFixture(t)
	cleanupApplyOrigin(t, root, base)
	runCommandGit(t, root, "push", "origin", ":refs/heads/topic")
	base.safety = model.BranchSafety{} // Deleted refs cannot supply branch protection.
	p := &bulkCleanupProvider{cleanupApplyProvider: base, records: map[string]*model.PullRequest{"topic": base.pr}}
	result, err := runBulkCleanup(t, root, p, "topic", "--local", "--origin")
	require.NoError(t, err)
	require.Equal(t, "completed", result.Results[0].Local)
	require.Equal(t, "absent", result.Results[0].Origin)
	require.False(t, result.Results[0].Forced)
	require.Equal(t, 0, p.safetyReads)
}

func TestCleanupBulkRejectsIncompleteOrMixedSelections(t *testing.T) {
	root, base := cleanupApplyFixture(t)
	p := &bulkCleanupProvider{cleanupApplyProvider: base, records: map[string]*model.PullRequest{"topic": base.pr}}
	_, err := runBulkCleanup(t, root, p, "--all", "topic", "--local")
	require.ErrorContains(t, err, "cannot combine")
	for i := 0; i < 100; i++ {
		runCommandGit(t, root, "branch", fmt.Sprintf("extra-%03d", i))
	}
	_, err = runBulkCleanup(t, root, p, "--all", "--local", "--dry-run")
	require.ErrorContains(t, err, "incomplete")
	require.NotEmpty(t, runCommandGit(t, root, "rev-parse", "topic"))
}

func TestCleanupBulkForceNeedsExplicitWorktreeAndConfirmation(t *testing.T) {
	root, base := cleanupApplyFixture(t)
	p := &bulkCleanupProvider{cleanupApplyProvider: base, records: map[string]*model.PullRequest{"topic": base.pr}}
	path := filepath.Join(t.TempDir(), "dirty")
	runCommandGit(t, root, "worktree", "add", path, "topic")
	require.NoError(t, os.WriteFile(filepath.Join(path, "keep.txt"), []byte("keep until exact approval"), 0600))
	plan, err := runBulkCleanup(t, root, p, "--all", "--local", "--force", "topic", "--dry-run")
	require.NoError(t, err)
	require.Equal(t, "retained", bulkResult(t, plan, "topic").Local)
	plan, err = runBulkCleanup(t, root, p, "--all", "--local", "--force", "topic", "--worktree", path, "--dry-run")
	require.NoError(t, err)
	require.Equal(t, "planned", bulkResult(t, plan, "topic").Local)
	_, err = runBulkCleanup(t, root, p, "--all", "--local", "--force", "topic", "--worktree", path)
	require.ErrorContains(t, err, "confirmation")
	require.FileExists(t, filepath.Join(path, "keep.txt"))
}
