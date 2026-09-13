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
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = w.workdir
	output, err := command.CombinedOutput()
	if err == nil {
		return nil
	}
	diagnostic := strings.TrimSpace(string(output))
	if diagnostic == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, diagnostic)
}
