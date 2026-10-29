package branch

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/raithlin/gha/internal/interfaces"
	"github.com/raithlin/gha/pkg/model"
)

type localCleanupReader interface {
	Cleanup(context.Context, string, int) (*model.BranchCleanup, error)
}

type cleanupWorktreePlanner interface {
	PlanCleanupWorktree(context.Context, string, string, model.RepositoryRef, *model.PullRequest) (*model.PRWorktree, error)
}

type cleanupEvidenceProvider interface {
	GetRepository(context.Context, string, string) (*model.Repository, error)
	ListPullRequests(context.Context, string, string, interfaces.ListPRsOptions) ([]*model.PullRequest, error)
	GetPullRequest(context.Context, string, string, int) (*model.PullRequest, error)
}

// Cleanup preserves offline reachability inspection or requires fresh, exact PR
// head evidence for every candidate. Neither mode writes Git or provider state.
func (s *Service) Cleanup(ctx context.Context, base string, limit int, repository model.RepositoryRef, repositoryError error, mergedPRs bool) (*model.BranchCleanup, error) {
	lister, ok := s.lister.(localCleanupReader)
	if !ok {
		return nil, fmt.Errorf("local cleanup inspection is not supported")
	}
	result, err := lister.Cleanup(ctx, base, limit)
	if err != nil || !mergedPRs {
		return result, err
	}
	result.Rule = "fresh_merged_pr_head_matches"
	entries := append(append([]*model.BranchCleanupCandidate{}, result.Candidates...), result.Excluded...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	result.Candidates = make([]*model.BranchCleanupCandidate, 0)
	result.Excluded = make([]*model.BranchCleanupCandidate, 0)
	provider, _ := s.provider.(cleanupEvidenceProvider)
	defaultBranch, repositoryError := cleanupDefaultBranch(ctx, provider, repository, repositoryError)
	result.DefaultBranch = defaultBranch
	for _, entry := range entries {
		evidence := &model.CleanupMergeEvidence{State: "unavailable", Reason: "provider_unavailable", Repository: repository, ObservedAt: time.Now().UTC().Format(time.RFC3339)}
		entry.MergeEvidence = evidence
		if repositoryError != nil {
			evidence.Message = repositoryError.Error()
		} else {
			inspectCleanupPR(ctx, provider, result.Base, defaultBranch, entry, evidence)
		}
		s.planCleanupRetirement(ctx, entry, evidence)
		protected := entry.Reason == "base_branch" || entry.Reason == "current_branch" || entry.Reason == "checked_out_in_worktree"
		if !protected {
			entry.Reason = evidence.Reason
		}
		if !protected && evidence.Reason == "merged_pr_head_matches" {
			result.Candidates = append(result.Candidates, entry)
		} else {
			result.Excluded = append(result.Excluded, entry)
		}
	}
	return result, nil
}

func inspectCleanupPR(ctx context.Context, provider cleanupEvidenceProvider, base, defaultBranch string, entry *model.BranchCleanupCandidate, evidence *model.CleanupMergeEvidence) {
	evidence.State = "fresh"
	if entry.Name == defaultBranch {
		setCleanupReason(evidence, "default_branch", "repository default branch must be retained")
		return
	}
	repo := evidence.Repository
	prs, err := provider.ListPullRequests(ctx, repo.Owner, repo.Name, interfaces.ListPRsOptions{State: "all", Head: repo.Owner + ":" + entry.Name, PerPage: 2, Page: 1, Sort: "updated", Direction: "desc"})
	if err != nil {
		evidence.State = "unavailable"
		evidence.Message = err.Error()
		return
	}
	if len(prs) == 0 {
		setCleanupReason(evidence, "no_associated_pr", "no PR found; a missing origin branch does not prove a merge")
		return
	}
	if len(prs) != 1 || prs[0] == nil || prs[0].Number < 1 {
		setCleanupReason(evidence, "ambiguous_pr_history", "multiple or incomplete PR identities; inspect branch reuse and select a PR explicitly before cleanup")
		return
	}
	pr, err := provider.GetPullRequest(ctx, repo.Owner, repo.Name, prs[0].Number)
	if err != nil || pr == nil || pr.Number != prs[0].Number {
		evidence.State = "unavailable"
		evidence.Message = "fresh PR identity unavailable; retry inspection"
		if err != nil {
			evidence.Message = err.Error()
		}
		return
	}
	evidence.PullRequest = pr
	classifyCleanupPR(base, entry, evidence, pr)
}

func classifyCleanupPR(base string, entry *model.BranchCleanupCandidate, evidence *model.CleanupMergeEvidence, pr *model.PullRequest) {
	repo := evidence.Repository
	switch {
	case !cleanupRepositoryMatches(repo, pr.Head.Repo) || !cleanupRepositoryMatches(repo, pr.Base.Repo):
		setCleanupReason(evidence, "pr_repository_mismatch", "PR head and base must belong to the checkout fetch repository; inspect fork or deleted repository identity")
	case pr.Head.Ref != entry.Name:
		setCleanupReason(evidence, "pr_branch_mismatch", "provider head branch differs from the local selection; inspect the PR")
	case pr.Base.Ref != base:
		setCleanupReason(evidence, "pr_base_mismatch", "PR merged into a different base; select that base explicitly")
	case !cleanupMerged(pr):
		setCleanupReason(evidence, "pr_not_merged", "PR is open, unmerged or has unavailable merge evidence; retain its branch")
	case entry.Local == nil || len(pr.Head.SHA) != 40 && len(pr.Head.SHA) != 64 || pr.Head.SHA != entry.Local.SHA:
		setCleanupReason(evidence, "pr_head_mismatch", "local tip differs from the merged PR head; inspect unpublished commits or branch reuse")
	default:
		setCleanupReason(evidence, "merged_pr_head_matches", "fresh merged PR matches this repository, branch and exact local tip; merge strategy does not affect this evidence")
	}
}

func cleanupRepositoryMatches(target model.RepositoryRef, repo *model.Repository) bool {
	return repo != nil && strings.EqualFold(repo.Owner.Login, target.Owner) && strings.EqualFold(repo.Name, target.Name)
}
func cleanupMerged(pr *model.PullRequest) bool {
	merged, err := time.Parse(time.RFC3339, pr.MergedAt)
	return pr.State == "closed" && err == nil && !merged.IsZero() && !merged.After(time.Now())
}
func setCleanupReason(evidence *model.CleanupMergeEvidence, reason, message string) {
	evidence.Reason = reason
	evidence.Message = message
}

func cleanupDefaultBranch(ctx context.Context, provider cleanupEvidenceProvider, repository model.RepositoryRef, previousError error) (string, error) {
	if previousError != nil {
		return "", previousError
	}
	if provider == nil || repository.Owner == "" || repository.Name == "" {
		return "", fmt.Errorf("provider or repository identity unavailable; inspect the checkout fetch origin")
	}
	repo, err := provider.GetRepository(ctx, repository.Owner, repository.Name)
	if err != nil {
		return "", err
	}
	if !cleanupRepositoryMatches(repository, repo) || repo.DefaultBranch == "" {
		return "", fmt.Errorf("repository identity or default branch unavailable; retain branches until provider inspection succeeds")
	}
	return repo.DefaultBranch, nil
}

func (s *Service) planCleanupRetirement(ctx context.Context, entry *model.BranchCleanupCandidate, evidence *model.CleanupMergeEvidence) {
	if evidence.Reason != "merged_pr_head_matches" || entry.WorktreePath == "" {
		return
	}
	planner, supported := s.lister.(cleanupWorktreePlanner)
	if !supported {
		return
	}
	plan, err := planner.PlanCleanupWorktree(ctx, entry.WorktreePath, entry.Name, evidence.Repository, evidence.PullRequest)
	if err != nil {
		evidence.Message += "; worktree assessment unavailable: " + err.Error()
		return
	}
	entry.WorktreeRetirement = plan
}
