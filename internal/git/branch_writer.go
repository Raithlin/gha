package git

import (
	"context"
	"errors"
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

// ValidateCreate checks a proposed branch creation without changing local or
// origin refs. Origin refs are read from the remote when publish is selected.
func (w *BranchWriter) ValidateCreate(ctx context.Context, name, from string, publish bool) error {
	if err := w.validateName(ctx, name); err != nil {
		return err
	}
	local, err := w.localBranches(ctx)
	if err != nil {
		return err
	}
	if err := rejectBranchConflict(name, local, "locally"); err != nil {
		return err
	}
	start := from
	if start == "" {
		start = "HEAD"
	}
	if _, err := w.output(ctx, "rev-parse", "--verify", "--quiet", "--end-of-options", start+"^{commit}"); err != nil {
		return fmt.Errorf("start point %q does not resolve to a commit: %w", start, err)
	}
	if publish {
		origin, err := w.originBranches(ctx)
		if err != nil {
			return fmt.Errorf("inspect origin branches before creating: %w", err)
		}
		if err := rejectBranchConflict(name, origin, "on origin"); err != nil {
			return err
		}
	}
	return nil
}

// ValidateRename checks the local and selected origin refs before a rename.
func (w *BranchWriter) ValidateRename(ctx context.Context, oldName, newName string, origin bool) error {
	if err := w.validateName(ctx, oldName); err != nil {
		return err
	}
	if err := w.validateName(ctx, newName); err != nil {
		return err
	}
	if oldName == newName {
		return fmt.Errorf("old and new branch names are identical")
	}
	local, err := w.localBranches(ctx)
	if err != nil {
		return err
	}
	if !containsBranch(local, oldName) {
		return fmt.Errorf("branch %q was not found locally", oldName)
	}
	if err := rejectBranchConflict(newName, local, "locally"); err != nil {
		return err
	}
	if origin {
		remote, err := w.originBranches(ctx)
		if err != nil {
			return fmt.Errorf("inspect origin branches before renaming: %w", err)
		}
		if !containsBranch(remote, oldName) {
			return fmt.Errorf("branch %q does not exist on origin", oldName)
		}
		if err := rejectBranchConflict(newName, remote, "on origin"); err != nil {
			return err
		}
	}
	return nil
}

// ValidateDelete checks that every selected ref exists. It does not inspect
// provider policy or local checkout and merge safety.
func (w *BranchWriter) ValidateDelete(ctx context.Context, name string, local, origin bool) error {
	if !local && !origin {
		return fmt.Errorf("select at least one branch target")
	}
	if err := w.validateName(ctx, name); err != nil {
		return err
	}
	if local {
		branches, err := w.localBranches(ctx)
		if err != nil {
			return err
		}
		if !containsBranch(branches, name) {
			return fmt.Errorf("branch %q was not found locally", name)
		}
	}
	if origin {
		branches, err := w.originBranches(ctx)
		if err != nil {
			return fmt.Errorf("inspect origin branches before deleting: %w", err)
		}
		if !containsBranch(branches, name) {
			return fmt.Errorf("branch %q does not exist on origin", name)
		}
	}
	return nil
}

// ValidateLocalDelete mirrors git branch -d: a configured upstream takes
// precedence over HEAD (or the branch selected for a pending switch).
func (w *BranchWriter) ValidateLocalDelete(ctx context.Context, name string, force bool, mergedInto string) error {
	if force {
		return nil
	}
	target := "HEAD"
	if mergedInto != "" {
		target = "refs/heads/" + mergedInto
		if _, err := w.output(ctx, "rev-parse", "--verify", "--quiet", target+"^{commit}"); err != nil {
			return fmt.Errorf("safe branch %q does not resolve to a commit: %w", mergedInto, err)
		}
	}
	upstream, err := w.output(ctx, "for-each-ref", "--format=%(upstream)", "refs/heads/"+name)
	if err != nil {
		return fmt.Errorf("check whether branch %q is merged: read upstream: %w", name, err)
	}
	if upstream = strings.TrimSpace(upstream); upstream != "" {
		command := exec.CommandContext(ctx, "git", "show-ref", "--verify", "--quiet", upstream)
		command.Dir = w.workdir
		if err := command.Run(); err == nil {
			target = upstream
		} else {
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
				return fmt.Errorf("check upstream ref %q for branch %q: %w", upstream, name, err)
			}
		}
	}
	command := exec.CommandContext(ctx, "git", "merge-base", "--is-ancestor", "refs/heads/"+name, target)
	command.Dir = w.workdir
	if err := command.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return fmt.Errorf("branch %q is not fully merged into %s; use --force only after independent verification", name, target)
		}
		return fmt.Errorf("check whether branch %q is merged into %s: %w", name, target, err)
	}
	return nil
}

func (w *BranchWriter) validateName(ctx context.Context, name string) error {
	if _, err := w.output(ctx, "check-ref-format", "--branch", name); err != nil {
		return fmt.Errorf("invalid branch name %q: %w", name, err)
	}
	return nil
}

func (w *BranchWriter) localBranches(ctx context.Context) ([]string, error) {
	output, err := w.output(ctx, "for-each-ref", "--format=%(refname:lstrip=2)", "refs/heads")
	if err != nil {
		return nil, fmt.Errorf("inspect local branches: %w", err)
	}
	return nonemptyLines(output), nil
}

func (w *BranchWriter) originBranches(ctx context.Context) ([]string, error) {
	resolved, err := w.output(ctx, "remote", "get-url", "--push", "--all", "origin")
	if err != nil {
		return nil, fmt.Errorf("read origin branch refs: resolve push URL: %w", err)
	}
	pushURLs := nonemptyLines(resolved)
	if len(pushURLs) == 0 {
		return nil, fmt.Errorf("read origin branch refs: origin has no push URL")
	}
	if len(pushURLs) > 1 {
		return nil, fmt.Errorf("read origin branch refs: origin has multiple push URLs (%d); select one write target", len(pushURLs))
	}
	output, err := w.output(ctx, "ls-remote", "--heads", "--", pushURLs[0])
	if err != nil {
		return nil, fmt.Errorf("read origin branch refs: %w", err)
	}
	branches := make([]string, 0)
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.HasPrefix(fields[1], "refs/heads/") {
			branches = append(branches, strings.TrimPrefix(fields[1], "refs/heads/"))
		}
	}
	return branches, nil
}

func rejectBranchConflict(name string, existing []string, location string) error {
	for _, branch := range existing {
		if branchRefConflict(name, branch) {
			if name == branch {
				return fmt.Errorf("branch target %q already exists %s", name, location)
			}
			return fmt.Errorf("branch target %q conflicts with existing ref %q %s", name, branch, location)
		}
	}
	return nil
}

func branchRefConflict(first, second string) bool {
	return first == second || strings.HasPrefix(first, second+"/") || strings.HasPrefix(second, first+"/")
}

func containsBranch(branches []string, name string) bool {
	for _, branch := range branches {
		if branch == name {
			return true
		}
	}
	return false
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
