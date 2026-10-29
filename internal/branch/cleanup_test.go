package branch

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/raithlin/gha/internal/interfaces"
	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/require"
)

type cleanupLister struct {
	fakeLister
	result *model.BranchCleanup
}

func (l *cleanupLister) Cleanup(context.Context, string, int) (*model.BranchCleanup, error) {
	return l.result, l.err
}

type cleanupProvider struct {
	fakeSafetyProvider
	prs                      []*model.PullRequest
	pr                       *model.PullRequest
	repo                     *model.Repository
	listErr, getErr, repoErr error
	calls                    int
}

func (p *cleanupProvider) GetRepository(context.Context, string, string) (*model.Repository, error) {
	return p.repo, p.repoErr
}
func (p *cleanupProvider) ListPullRequests(_ context.Context, owner, repo string, opts interfaces.ListPRsOptions) ([]*model.PullRequest, error) {
	if owner != "acme" || repo != "project" || opts.Head != "acme:topic" || opts.State != "all" || opts.PerPage != 2 || opts.Page != 1 {
		return nil, errors.New("unbounded or incorrect PR lookup")
	}
	p.calls++
	return p.prs, p.listErr
}
func (p *cleanupProvider) GetPullRequest(context.Context, string, string, int) (*model.PullRequest, error) {
	return p.pr, p.getErr
}

func cleanupFixture() (*cleanupLister, *cleanupProvider, model.RepositoryRef) {
	repo := &model.Repository{Name: "project", Owner: model.User{Login: "acme"}, DefaultBranch: "main"}
	pr := &model.PullRequest{Number: 7, State: "closed", MergedAt: "2026-10-28T12:00:00Z", Head: model.BranchRef{Ref: "topic", SHA: strings.Repeat("a", 40), Repo: repo}, Base: model.BranchRef{Ref: "main", Repo: repo}}
	l := &cleanupLister{result: &model.BranchCleanup{Base: "main", Candidates: []*model.BranchCleanupCandidate{}, Excluded: []*model.BranchCleanupCandidate{{Name: "topic", Local: &model.Branch{Name: "topic", SHA: pr.Head.SHA}, Reason: "not_reachable_from_base"}}}}
	return l, &cleanupProvider{repo: repo, pr: pr, prs: []*model.PullRequest{pr}}, model.RepositoryRef{Owner: "acme", Name: "project"}
}

func TestCleanupMergedEvidence(t *testing.T) {
	for _, strategy := range []string{"merge", "squash", "rebase"} {
		t.Run(strategy, func(t *testing.T) {
			l, p, repo := cleanupFixture()
			if strategy == "merge" {
				l.result.Candidates = l.result.Excluded
				l.result.Excluded = nil
				l.result.Candidates[0].Reason = "tip_reachable_from_base"
			}
			result, err := NewService(l, p).Cleanup(context.Background(), "main", 30, repo, nil, true)
			require.NoError(t, err)
			require.Len(t, result.Candidates, 1)
			require.Equal(t, "merged_pr_head_matches", result.Candidates[0].Reason)
			require.Equal(t, "fresh", result.Candidates[0].MergeEvidence.State)
			require.Equal(t, 7, result.Candidates[0].MergeEvidence.PullRequest.Number)
			require.NotEmpty(t, result.Candidates[0].MergeEvidence.ObservedAt)
			require.Equal(t, 1, p.calls)
		})
	}
}

func TestCleanupRetainsUnsafeEvidence(t *testing.T) {
	cases := []struct {
		name, reason string
		change       func(*cleanupLister, *cleanupProvider)
	}{
		{"post merge commits", "pr_head_mismatch", func(l *cleanupLister, _ *cleanupProvider) { l.result.Excluded[0].Local.SHA = strings.Repeat("b", 40) }},
		{"reused name", "ambiguous_pr_history", func(_ *cleanupLister, p *cleanupProvider) { p.prs = append(p.prs, p.pr) }},
		{"open", "pr_not_merged", func(_ *cleanupLister, p *cleanupProvider) { p.pr.State = "open" }},
		{"closed unmerged", "pr_not_merged", func(_ *cleanupLister, p *cleanupProvider) { p.pr.MergedAt = "" }},
		{"invalid merge time", "pr_not_merged", func(_ *cleanupLister, p *cleanupProvider) { p.pr.MergedAt = "yesterday" }},
		{"missing remote alone", "no_associated_pr", func(_ *cleanupLister, p *cleanupProvider) { p.prs = nil }},
		{"list failure", "provider_unavailable", func(_ *cleanupLister, p *cleanupProvider) { p.listErr = errors.New("offline") }},
		{"get failure", "provider_unavailable", func(_ *cleanupLister, p *cleanupProvider) { p.getErr = errors.New("offline") }},
		{"null get", "provider_unavailable", func(_ *cleanupLister, p *cleanupProvider) { p.pr = nil }},
		{"null list", "ambiguous_pr_history", func(_ *cleanupLister, p *cleanupProvider) { p.prs = []*model.PullRequest{nil} }},
		{"other repository", "pr_repository_mismatch", func(_ *cleanupLister, p *cleanupProvider) {
			p.pr.Head.Repo = &model.Repository{Name: "fork", Owner: model.User{Login: "acme"}}
		}},
		{"other base repository", "pr_repository_mismatch", func(_ *cleanupLister, p *cleanupProvider) { p.pr.Base.Repo = nil }},
		{"other branch", "pr_branch_mismatch", func(_ *cleanupLister, p *cleanupProvider) { p.pr.Head.Ref = "other" }},
		{"other base", "pr_base_mismatch", func(_ *cleanupLister, p *cleanupProvider) { p.pr.Base.Ref = "release" }},
		{"invalid head", "pr_head_mismatch", func(_ *cleanupLister, p *cleanupProvider) { p.pr.Head.SHA = "" }},
		{"default branch", "default_branch", func(_ *cleanupLister, p *cleanupProvider) { p.repo.DefaultBranch = "topic" }},
		{"unknown default", "provider_unavailable", func(_ *cleanupLister, p *cleanupProvider) { p.repo.DefaultBranch = "" }},
		{"repo unavailable", "provider_unavailable", func(_ *cleanupLister, p *cleanupProvider) { p.repoErr = errors.New("offline") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l, p, repo := cleanupFixture()
			tc.change(l, p)
			result, err := NewService(l, p).Cleanup(context.Background(), "main", 30, repo, nil, true)
			require.NoError(t, err)
			require.Empty(t, result.Candidates)
			require.Len(t, result.Excluded, 1)
			require.Equal(t, tc.reason, result.Excluded[0].Reason)
			require.NotEmpty(t, result.Excluded[0].MergeEvidence.Message)
		})
	}
}

func TestCleanupWorktreesAndLocalMode(t *testing.T) {
	l, p, repo := cleanupFixture()
	l.result.Excluded[0].Reason = "checked_out_in_worktree"
	l.result.Excluded[0].WorktreePath = "/tmp/linked"
	result, err := NewService(l, p).Cleanup(context.Background(), "main", 30, repo, nil, true)
	require.NoError(t, err)
	require.Empty(t, result.Candidates)
	require.Equal(t, "checked_out_in_worktree", result.Excluded[0].Reason)
	require.Equal(t, "merged_pr_head_matches", result.Excluded[0].MergeEvidence.Reason)
	l, p, repo = cleanupFixture()
	result, err = NewService(l, p).Cleanup(context.Background(), "main", 30, repo, nil, false)
	require.NoError(t, err)
	require.Equal(t, 0, p.calls)
	require.Nil(t, result.Excluded[0].MergeEvidence)
}

func TestCleanupServiceFailures(t *testing.T) {
	l, _, repo := cleanupFixture()
	_, err := NewService(&fakeLister{}).Cleanup(context.Background(), "main", 30, repo, nil, true)
	require.ErrorContains(t, err, "not supported")
	l.err = errors.New("local failed")
	_, err = NewService(l).Cleanup(context.Background(), "main", 30, repo, nil, true)
	require.ErrorContains(t, err, "local failed")
	l, _, repo = cleanupFixture()
	result, err := NewService(l).Cleanup(context.Background(), "main", 30, repo, nil, true)
	require.NoError(t, err)
	require.Equal(t, "provider_unavailable", result.Excluded[0].Reason)
	l, _, repo = cleanupFixture()
	result, err = NewService(l).Cleanup(context.Background(), "main", 30, repo, errors.New("origin mismatch"), true)
	require.NoError(t, err)
	require.Contains(t, result.Excluded[0].MergeEvidence.Message, "origin mismatch")
}

type cleanupPlanningLister struct {
	*cleanupLister
	planErr error
}

func (l *cleanupPlanningLister) PlanCleanupWorktree(context.Context, string, string, model.RepositoryRef, *model.PullRequest) (*model.PRWorktree, error) {
	return &model.PRWorktree{Decision: "keep", Reasons: []string{"local work"}, DryRun: true}, l.planErr
}
func TestCleanupWorktreePlanning(t *testing.T) {
	for _, fail := range []bool{false, true} {
		l, p, repo := cleanupFixture()
		l.result.Excluded[0].WorktreePath = "/tmp/linked"
		l.result.Excluded[0].Reason = "checked_out_in_worktree"
		planner := &cleanupPlanningLister{cleanupLister: l}
		if fail {
			planner.planErr = errors.New("status failed")
		}
		result, err := NewService(planner, p).Cleanup(context.Background(), "main", 30, repo, nil, true)
		require.NoError(t, err)
		require.Empty(t, result.Candidates)
		if fail {
			require.Contains(t, result.Excluded[0].MergeEvidence.Message, "status failed")
		} else {
			require.Equal(t, "keep", result.Excluded[0].WorktreeRetirement.Decision)
		}
	}
}

func TestCleanupEvidenceCannotReuseAnOldAssessment(t *testing.T) {
	l, p, repo := cleanupFixture()
	p.prs = []*model.PullRequest{{Number: 7}}
	first, err := NewService(l, p).Cleanup(context.Background(), "main", 30, repo, nil, true)
	require.NoError(t, err)
	require.Len(t, first.Candidates, 1)
	p.pr.State = "open"
	p.pr.Head.SHA = strings.Repeat("b", 40)
	second, err := NewService(l, p).Cleanup(context.Background(), "main", 30, repo, nil, true)
	require.NoError(t, err)
	require.Empty(t, second.Candidates)
	require.Equal(t, "pr_not_merged", second.Excluded[0].Reason)
	require.Equal(t, 2, p.calls)
}
