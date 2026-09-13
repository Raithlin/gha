package git

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/raithlin/gha/pkg/model"
)

// BranchLister reads local and origin remote-tracking branches from a Git worktree.
type BranchLister struct {
	workdir string
}

// NewBranchLister creates a lister rooted at workdir. An empty workdir uses the
// current directory.
func NewBranchLister(workdir string) *BranchLister {
	return &BranchLister{workdir: workdir}
}

// List returns at most limit branches from each source without changing Git state.
func (l *BranchLister) List(ctx context.Context, limit int) (*model.BranchInventory, error) {
	current, err := l.run(ctx, "branch", "--show-current")
	if err != nil {
		return nil, fmt.Errorf("read current branch: %w", err)
	}
	local, err := l.listLocal(ctx, strings.TrimSpace(current), limit)
	if err != nil {
		return nil, err
	}
	origin, err := l.originURL(ctx)
	if err != nil {
		return nil, err
	}
	originBranches, err := l.listOrigin(ctx, limit)
	if err != nil {
		return nil, err
	}

	return &model.BranchInventory{
		SchemaVersion:  model.BranchInventorySchemaVersion,
		Origin:         origin,
		Local:          local,
		OriginBranches: originBranches,
	}, nil
}

func (l *BranchLister) listLocal(ctx context.Context, current string, limit int) ([]*model.Branch, error) {
	output, err := l.run(ctx, "for-each-ref", "--format=%(refname:short)\t%(objectname)\t%(upstream:short)", "refs/heads")
	if err != nil {
		return nil, fmt.Errorf("list local branches: %w", err)
	}
	branches := make([]*model.Branch, 0)
	for _, line := range lines(output) {
		fields := strings.Split(line, "\t")
		if len(fields) != 3 {
			return nil, fmt.Errorf("parse local branch %q", line)
		}
		branch := &model.Branch{Name: fields[0], SHA: fields[1], Current: fields[0] == current, Upstream: fields[2]}
		if branch.Upstream != "" {
			ahead, behind, err := l.divergence(ctx, branch.Name, branch.Upstream)
			if err != nil {
				return nil, err
			}
			branch.Ahead = &ahead
			branch.Behind = &behind
		}
		branches = append(branches, branch)
		if len(branches) == limit {
			break
		}
	}
	return branches, nil
}

func (l *BranchLister) listOrigin(ctx context.Context, limit int) ([]*model.Branch, error) {
	output, err := l.run(ctx, "for-each-ref", "--format=%(refname:strip=3)\t%(objectname)", "refs/remotes/origin")
	if err != nil {
		return nil, fmt.Errorf("list origin branches: %w", err)
	}
	branches := make([]*model.Branch, 0)
	for _, line := range lines(output) {
		fields := strings.Split(line, "\t")
		if len(fields) != 2 {
			return nil, fmt.Errorf("parse origin branch %q", line)
		}
		if fields[0] == "HEAD" {
			continue
		}
		branches = append(branches, &model.Branch{Name: fields[0], SHA: fields[1]})
		if len(branches) == limit {
			break
		}
	}
	return branches, nil
}

func (l *BranchLister) originURL(ctx context.Context) (string, error) {
	output, err := l.run(ctx, "remote", "get-url", "origin")
	if err != nil {
		if isExitError(err) {
			return "", nil
		}
		return "", fmt.Errorf("read origin URL: %w", err)
	}
	return strings.TrimSpace(output), nil
}

func (l *BranchLister) divergence(ctx context.Context, branch, upstream string) (int, int, error) {
	output, err := l.run(ctx, "rev-list", "--left-right", "--count", branch+"..."+upstream)
	if err != nil {
		return 0, 0, fmt.Errorf("compare %s with %s: %w", branch, upstream, err)
	}
	fields := strings.Fields(output)
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("parse divergence for %s", branch)
	}
	ahead, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parse ahead count for %s: %w", branch, err)
	}
	behind, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse behind count for %s: %w", branch, err)
	}
	return ahead, behind, nil
}

func (l *BranchLister) run(ctx context.Context, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = l.workdir
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func lines(output string) []string {
	trimmed := strings.TrimSuffix(output, "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func isExitError(err error) bool {
	_, ok := err.(*exec.ExitError)
	return ok
}
