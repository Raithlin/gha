package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// BranchWriter performs explicit local and origin branch operations from one
// checkout. It never fetches implicitly.
type BranchWriter struct {
	workdir string
}

// NewBranchWriter creates a writer rooted at workdir. An empty workdir uses
// the current directory.
func NewBranchWriter(workdir string) *BranchWriter {
	return &BranchWriter{workdir: workdir}
}

// CreateLocal creates name at from, or HEAD when from is empty.
func (w *BranchWriter) CreateLocal(ctx context.Context, name, from string) error {
	args := []string{"branch", name}
	if from != "" {
		args = append(args, from)
	}
	return w.run(ctx, args...)
}

// Publish creates or updates origin/name and configures it as the local
// branch's upstream.
func (w *BranchWriter) Publish(ctx context.Context, name string) error {
	return w.run(ctx, "push", "--set-upstream", "origin", name)
}

// RenameLocal renames one local branch without switching the worktree.
func (w *BranchWriter) RenameLocal(ctx context.Context, oldName, newName string) error {
	return w.run(ctx, "branch", "-m", oldName, newName)
}

// CurrentBranch returns the currently checked-out branch. It is empty when
// HEAD is detached.
func (w *BranchWriter) CurrentBranch(ctx context.Context) (string, error) {
	output, err := w.output(ctx, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// DefaultBranch reads the cached origin default branch without contacting the
// remote. The branch must already exist locally before Switch can use it.
func (w *BranchWriter) DefaultBranch(ctx context.Context) (string, error) {
	output, err := w.output(ctx, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD")
	if err != nil {
		return "", fmt.Errorf("read cached origin default branch: %w", err)
	}
	branch := strings.TrimPrefix(strings.TrimSpace(output), "origin/")
	if branch == "" {
		return "", fmt.Errorf("read cached origin default branch: empty branch name")
	}
	return branch, nil
}

// Switch checks out an existing local branch without changing remote state.
func (w *BranchWriter) Switch(ctx context.Context, name string) error {
	return w.run(ctx, "switch", name)
}

// RenameOrigin creates the new remote name from its already-renamed local
// branch and deletes the old remote name in a single push invocation.
func (w *BranchWriter) RenameOrigin(ctx context.Context, oldName, newName string) error {
	return w.run(ctx, "push", "origin", newName+":refs/heads/"+newName, ":refs/heads/"+oldName)
}

// DeleteLocal removes name. Git's normal merged-branch guard is retained
// unless force is explicitly selected.
func (w *BranchWriter) DeleteLocal(ctx context.Context, name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	return w.run(ctx, "branch", flag, name)
}

// DeleteOrigin removes origin/name.
func (w *BranchWriter) DeleteOrigin(ctx context.Context, name string) error {
	return w.run(ctx, "push", "origin", "--delete", name)
}

func (w *BranchWriter) run(ctx context.Context, args ...string) error {
	_, err := w.output(ctx, args...)
	return err
}

func (w *BranchWriter) output(ctx context.Context, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = w.workdir
	output, err := command.CombinedOutput()
	if err == nil {
		return string(output), nil
	}
	diagnostic := strings.TrimSpace(string(output))
	if diagnostic == "" {
		return "", err
	}
	return "", fmt.Errorf("%w: %s", err, diagnostic)
}
