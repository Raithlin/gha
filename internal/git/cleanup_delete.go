package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/raithlin/gha/pkg/model"
)

// CleanupDeleteOptions is one selected branch bound to inspected commit IDs.
// The caller must verify fresh PR evidence or obtain branch-specific force
// confirmation before calling CleanupDelete. Git checks never override locks
// or primary/current checkout protection, even for forced cleanup.
type CleanupDeleteOptions struct {
	Branch, ExpectedSHA, Base, DefaultBranch, WorktreePath string
	Local, Origin, Force                                   bool
	ExpectedOriginSHA                                      *string
	PushURL                                                string
}

// PlanCleanupDelete assesses local checkouts and fresh origin refs, with no writes.
func (w *BranchWriter) PlanCleanupDelete(ctx context.Context, input CleanupDeleteOptions) (*model.CleanupBranchResult, error) {
	if !input.Local && !input.Origin {
		return nil, fmt.Errorf("select --local and/or --origin")
	}
	if err := w.validateName(ctx, input.Branch); err != nil {
		return nil, err
	}
	if err := w.validateCleanupTip(ctx, input); err != nil {
		return nil, err
	}
	result := &model.CleanupBranchResult{Name: input.Branch, SHA: input.ExpectedSHA, Forced: input.Force, State: "planned", Local: "not_requested", Origin: "not_requested", Worktree: "not_requested"}
	if input.Local {
		result.Local = "planned"
	}
	if err := w.planCleanupCheckout(ctx, input, result); err != nil {
		return nil, err
	}
	if input.Origin {
		if err := w.planCleanupOrigin(ctx, input, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (w *BranchWriter) planCleanupCheckout(ctx context.Context, input CleanupDeleteOptions, result *model.CleanupBranchResult) error {
	service := NewWorktreeService(w.workdir)
	entries, root, err := service.entries(ctx)
	if err != nil {
		return err
	}
	selected := ""
	if input.WorktreePath != "" {
		if !input.Local {
			return fmt.Errorf("worktree retirement requires --local")
		}
		selected, err = absolutePath(input.WorktreePath, root)
		if err != nil {
			return err
		}
	}
	found := false
	for _, entry := range entries {
		if entry.Branch != input.Branch {
			continue
		}
		if protectedPRCheckout(&entry) {
			return fmt.Errorf("branch uses a primary/current, locked or unavailable checkout at %s", entry.Path)
		}
		if !input.Local {
			return fmt.Errorf("branch has a linked checkout; select --local and its explicit --worktree before origin cleanup")
		}
		if selected != entry.Path || found {
			return fmt.Errorf("explicitly select this branch's single linked worktree: %s", entry.Path)
		}
		found = true
		if err := validateCleanupWorktree(ctx, input, entry); err != nil {
			return err
		}
		result.WorktreePath = entry.Path
		result.Worktree = "planned"
	}
	if selected != "" && !found {
		return fmt.Errorf("selected worktree does not belong to this branch")
	}
	return nil
}

func (w *BranchWriter) cleanupOriginTip(ctx context.Context, branch string) (string, string, error) {
	pushURL, err := w.CleanupPushTarget(ctx)
	if err != nil {
		return "", "", err
	}
	ref := "refs/heads/" + branch
	output, err := w.output(ctx, "ls-remote", "--heads", "--", pushURL, ref)
	if err != nil {
		return "", "", err
	}
	lines := nonemptyLines(output)
	if len(lines) == 0 {
		return "", pushURL, nil
	}
	fields := strings.Fields(lines[0])
	if len(lines) != 1 || len(fields) != 2 || fields[1] != ref || !commitID.MatchString(fields[0]) {
		return "", "", fmt.Errorf("origin branch identity is ambiguous")
	}
	return fields[0], pushURL, nil
}

// CleanupPushTarget resolves one immutable transport destination for a plan.
func (w *BranchWriter) CleanupPushTarget(ctx context.Context) (string, error) {
	urls, err := w.output(ctx, "remote", "get-url", "--push", "--all", "origin")
	if err != nil {
		return "", err
	}
	values := nonemptyLines(urls)
	if len(values) != 1 {
		return "", fmt.Errorf("origin must have exactly one push target")
	}
	return values[0], nil
}

// CleanupDelete repeats local checks, retires the selected checkout, then
// deletes origin with an exact lease and the local ref with compare-and-delete.
// It preserves completed effects and the original commit ID on partial failure.
func (w *BranchWriter) CleanupDelete(ctx context.Context, input CleanupDeleteOptions) (*model.CleanupBranchResult, error) {
	result, err := w.PlanCleanupDelete(ctx, input)
	if err != nil {
		return nil, err
	}
	result.Recovery = fmt.Sprintf("Preserved tip %s; recreate the branch at that commit if needed. Inspect origin and worktrees before retrying; completed effects are not rolled back.", input.ExpectedSHA)
	if result.Worktree == "planned" {
		args := []string{"worktree", "remove"}
		if input.Force {
			args = append(args, "--force")
		}
		args = append(args, result.WorktreePath)
		if err = w.run(ctx, args...); err != nil {
			return cleanupDeleteFailure(result, "worktree", err)
		}
		result.Worktree = "completed"
	}
	if result.Origin == "planned" {
		sha, url, readErr := w.cleanupOriginTip(ctx, input.Branch)
		if readErr != nil {
			return cleanupDeleteFailure(result, "origin", readErr)
		}
		if sha != result.OriginSHA || input.PushURL != "" && url != input.PushURL {
			return cleanupDeleteFailure(result, "origin", fmt.Errorf("origin changed after preflight; retain and inspect"))
		}
		ref := "refs/heads/" + input.Branch
		if err = w.run(ctx, "push", "--force-with-lease="+ref+":"+sha, "--", url, ":"+ref); err != nil {
			return cleanupDeleteFailure(result, "origin", err)
		}
		result.Origin = "completed"
	}
	if input.Local {
		// Recheck checkout protection after remote writes and worktree removal.
		recheck := input
		recheck.WorktreePath = ""
		recheck.Origin = false
		if _, err = w.PlanCleanupDelete(ctx, recheck); err != nil {
			return cleanupDeleteFailure(result, "local", err)
		}
		if err = w.run(ctx, "update-ref", "-d", "refs/heads/"+input.Branch, input.ExpectedSHA); err != nil {
			return cleanupDeleteFailure(result, "local", err)
		}
		result.Local = "completed"
		if err = w.removeCleanupBranchConfig(ctx, input.Branch); err != nil {
			result.State = "failed"
			result.Reason = "branch deleted but local branch configuration cleanup failed: " + err.Error()
			return result, err
		}
	}
	result.State = "completed"
	return result, nil
}

func (w *BranchWriter) removeCleanupBranchConfig(ctx context.Context, name string) error {
	_, err := w.output(ctx, "config", "--local", "--name-only", "--get-regexp", "^branch\\."+regexp.QuoteMeta(name)+"\\.")
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return nil
	}
	if err != nil {
		return err
	}
	return w.run(ctx, "config", "--local", "--remove-section", "branch."+name)
}

func cleanupDeleteFailure(result *model.CleanupBranchResult, effect string, err error) (*model.CleanupBranchResult, error) {
	result.State = "failed"
	result.Reason = err.Error()
	switch effect {
	case "worktree":
		result.Worktree = "failed"
	case "origin":
		result.Origin = "failed"
	case "local":
		result.Local = "failed"
	}
	if result.Local == "planned" {
		result.Local = "retained"
	}
	if result.Origin == "planned" {
		result.Origin = "retained"
	}
	return result, err
}

func (w *BranchWriter) validateCleanupTip(ctx context.Context, input CleanupDeleteOptions) error {
	if input.DefaultBranch == "" || input.Base == "" || input.Branch == input.Base || input.Branch == input.DefaultBranch {
		return fmt.Errorf("default/base branch is protected or its identity is unavailable")
	}
	current, err := w.output(ctx, "rev-parse", "--verify", "refs/heads/"+input.Branch)
	if err != nil || !commitID.MatchString(input.ExpectedSHA) || strings.TrimSpace(current) != input.ExpectedSHA {
		return fmt.Errorf("local branch tip changed or is unavailable; inspect and retry")
	}
	return nil
}

func validateCleanupWorktree(ctx context.Context, input CleanupDeleteOptions, entry model.Worktree) error {
	if entry.Head != input.ExpectedSHA {
		return fmt.Errorf("worktree head changed; inspect and retry")
	}
	if reason := prRetirementAmbiguity(ctx, entry.Path); reason != "" {
		return fmt.Errorf("retain checkout: %s", reason)
	}
	if !input.Force {
		status, statusErr := runGitAt(ctx, entry.Path, "status", "--porcelain=v1", "--untracked-files=all", "--ignored")
		if statusErr != nil || strings.TrimSpace(status) != "" {
			return fmt.Errorf("worktree contains dirty, untracked or ignored content, or status is unavailable")
		}
	}
	return nil
}

func (w *BranchWriter) planCleanupOrigin(ctx context.Context, input CleanupDeleteOptions, result *model.CleanupBranchResult) error {
	sha, pushURL, readErr := w.cleanupOriginTip(ctx, input.Branch)
	if readErr != nil {
		return readErr
	}
	if input.PushURL != "" && pushURL != input.PushURL {
		return fmt.Errorf("origin push target changed; inspect and retry")
	}
	if input.ExpectedOriginSHA != nil && sha != *input.ExpectedOriginSHA {
		return fmt.Errorf("origin tip changed after preview; inspect and retry")
	}
	if sha != "" && sha != input.ExpectedSHA && !input.Force {
		return fmt.Errorf("origin tip differs from the inspected local tip; retain the origin branch")
	}
	result.OriginSHA = sha
	result.Origin = "planned"
	if sha == "" {
		result.Origin = "absent"
	}
	return nil
}
