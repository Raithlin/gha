package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/raithlin/gha/pkg/model"
)

// Cleanup returns a bounded, read-only review of local cleanup candidates.
// A candidate's tip must be reachable from the selected base; it is not an
// age-based or provider-derived "stale" classification.
func (l *BranchLister) Cleanup(ctx context.Context, base string, limit int) (*model.BranchCleanup, error) {
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("limit must be between 1 and 100")
	}
	resolvedBase, err := l.cleanupBase(ctx, base)
	if err != nil {
		return nil, err
	}
	current, err := l.run(ctx, "branch", "--show-current")
	if err != nil {
		return nil, fmt.Errorf("read current branch: %w", err)
	}
	local, err := l.listLocal(ctx, strings.TrimSpace(current), limit)
	if err != nil {
		return nil, err
	}
	cleanup := &model.BranchCleanup{
		SchemaVersion: model.BranchCleanupSchemaVersion,
		Rule:          "tip_reachable_from_base",
		Base:          resolvedBase,
		Limit:         limit,
		Truncated:     local.truncated,
		Candidates:    make([]*model.BranchCleanupCandidate, 0),
		Excluded:      make([]*model.BranchCleanupCandidate, 0),
	}
	for _, branch := range local.branches {
		candidate := &model.BranchCleanupCandidate{Name: branch.Name, Local: branch}
		switch {
		case branch.Name == resolvedBase:
			candidate.Reason = "base_branch"
			cleanup.Excluded = append(cleanup.Excluded, candidate)
		case branch.Current:
			candidate.Reason = "current_branch"
			cleanup.Excluded = append(cleanup.Excluded, candidate)
		default:
			reachable, err := l.isAncestor(ctx, branch.Name, resolvedBase)
			if err != nil {
				return nil, fmt.Errorf("compare branch %q to base %q: %w", branch.Name, resolvedBase, err)
			}
			if reachable {
				candidate.Reason = "tip_reachable_from_base"
				cleanup.Candidates = append(cleanup.Candidates, candidate)
			} else {
				candidate.Reason = "not_reachable_from_base"
				cleanup.Excluded = append(cleanup.Excluded, candidate)
			}
		}
	}
	return cleanup, nil
}

func (l *BranchLister) cleanupBase(ctx context.Context, base string) (string, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		output, err := l.run(ctx, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD")
		if err != nil {
			return "", fmt.Errorf("resolve cleanup base from cached origin/HEAD: %w; pass --base <branch> to select a local base", err)
		}
		base = strings.TrimPrefix(strings.TrimSpace(output), "origin/")
	}
	if base == "" {
		return "", fmt.Errorf("cleanup base is empty; pass --base <branch>")
	}
	if _, err := l.run(ctx, "show-ref", "--verify", "--quiet", "refs/heads/"+base); err != nil {
		return "", fmt.Errorf("cleanup base %q is not a local branch", base)
	}
	return base, nil
}

func (l *BranchLister) isAncestor(ctx context.Context, ancestor, descendant string) (bool, error) {
	command := exec.CommandContext(ctx, "git", "merge-base", "--is-ancestor", "refs/heads/"+ancestor, "refs/heads/"+descendant)
	command.Dir = l.workdir
	output, err := command.CombinedOutput()
	if err == nil {
		return true, nil
	}
	if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
		return false, nil
	}
	diagnostic := strings.TrimSpace(string(output))
	if diagnostic == "" {
		return false, err
	}
	return false, fmt.Errorf("%w: %s", err, diagnostic)
}
