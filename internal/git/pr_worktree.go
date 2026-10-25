package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/raithlin/gha/pkg/model"
)

// PRWorktreeOptions binds one explicit local target to a freshly inspected PR.
type PRWorktreeOptions struct {
	Operation   string
	Path        string
	Branch      string
	Repository  model.RepositoryRef
	PullRequest *model.PullRequest
	DryRun      bool
}

var commitID = regexp.MustCompile(`^[0-9a-fA-F]{40}([0-9a-fA-F]{24})?$`)

// PRWorktree preflights checkout/resume or retirement, then executes unless
// DryRun is selected. Failures after execution begins retain observed effects.
func (s *WorktreeService) PRWorktree(ctx context.Context, input PRWorktreeOptions) (*model.PRWorktree, error) {
	plan, options, err := s.planPRWorktree(ctx, input)
	if err != nil {
		return nil, err
	}
	if input.DryRun {
		return plan, nil
	}
	switch plan.Decision {
	case "keep":
		plan.State = "kept"
	case "resume":
		plan.State = "resumed"
		if err = s.recordPRAssociation(ctx, input, plan); err != nil {
			return plan, err
		}
	case "remove":
		// Repeat the whole local assessment, including ignored files and commits.
		current, _, checkErr := s.planPRWorktree(ctx, input)
		if checkErr != nil {
			return plan, checkErr
		}
		if current.Decision != "remove" {
			current.State = "kept"
			return current, nil
		}
		_, err = s.Remove(ctx, plan.Path)
		if err != nil {
			plan.State = "failed"
			plan.WorktreeState = "failed"
			return plan, err
		}
		plan.State = "completed"
		plan.WorktreeState = "completed"
		plan.Worktree = nil
	case "create":
		return s.createPRWorktree(ctx, input, plan, options)
	}
	return plan, nil
}

func (s *WorktreeService) planPRWorktree(ctx context.Context, input PRWorktreeOptions) (*model.PRWorktree, WorktreeAddOptions, error) {
	options := WorktreeAddOptions{}
	if err := validatePRWorktree(input); err != nil {
		return nil, options, err
	}
	entries, root, err := s.entries(ctx)
	if err != nil {
		return nil, options, err
	}
	target, err := absolutePath(input.Path, root)
	if err != nil {
		return nil, options, err
	}
	branch := input.Branch
	if branch == "" {
		branch = defaultPRBranch(input)
	}
	if _, err = s.run(ctx, "check-ref-format", "--branch", branch); err != nil {
		return nil, options, fmt.Errorf("invalid branch name %q", branch)
	}
	plan := newPRWorktreePlan(input, target, branch)
	options = WorktreeAddOptions{Path: target, Branch: branch}
	for index := range entries {
		entry := &entries[index]
		if entry.Path == target {
			plan.Worktree = entry
			if input.Branch == "" && input.Operation == "retire" {
				plan.Branch = entry.Branch
			}
		}
	}
	if input.Operation == "retire" {
		return s.planPRRetirement(ctx, input, plan), options, nil
	}
	if current := currentEntry(entries); current == nil || current.Bare {
		return nil, options, fmt.Errorf("PR checkout requires a non-bare local checkout")
	}
	if err = s.planPRCheckout(ctx, input, plan, entries, &options); err != nil {
		return nil, options, err
	}
	return plan, options, nil
}

func validatePRWorktree(input PRWorktreeOptions) error {
	if input.Operation != "checkout" && input.Operation != "retire" {
		return fmt.Errorf("operation must be checkout or retire")
	}
	if strings.TrimSpace(input.Path) == "" {
		return fmt.Errorf("select an explicit worktree path with --worktree")
	}
	pr := input.PullRequest
	if pr == nil || pr.Number < 1 || !commitID.MatchString(pr.Head.SHA) || pr.Head.Ref == "" {
		return fmt.Errorf("PR identity or head is unavailable; inspect the pull request before retrying")
	}
	if input.Repository.Owner == "" || input.Repository.Name == "" {
		return fmt.Errorf("repository identity is required")
	}
	return nil
}

func defaultPRBranch(input PRWorktreeOptions) string {
	head := input.PullRequest.Head.Repo
	if head != nil && strings.EqualFold(head.Owner.Login, input.Repository.Owner) && strings.EqualFold(head.Name, input.Repository.Name) {
		return input.PullRequest.Head.Ref
	}
	return fmt.Sprintf("pr/%d", input.PullRequest.Number)
}

func newPRWorktreePlan(input PRWorktreeOptions, path, branch string) *model.PRWorktree {
	return &model.PRWorktree{SchemaVersion: model.PRWorktreeSchemaVersion, Operation: input.Operation, Repository: input.Repository, PullRequest: input.PullRequest, Branch: branch, Path: path, DryRun: input.DryRun, State: "planned", AssociationState: "not_needed", Fetch: "not_needed", BranchState: "retained", WorktreeState: "not_needed", ProviderState: "fresh", Comparison: model.BranchComparison{State: "unavailable"}, ReviewState: model.ProviderSignal{State: "unavailable"}, Push: prPushTarget(input.PullRequest), Reasons: []string{}, RecommendedActions: []model.RecommendedAction{}}
}

func prPushTarget(pr *model.PullRequest) model.PRPushTarget {
	target := model.PRPushTarget{State: "unavailable"}
	if pr.Head.Repo != nil && pr.Head.Repo.Name != "" && pr.Head.Repo.Owner.Login != "" {
		target.State = "available"
		target.Repository = model.RepositoryRef{Owner: pr.Head.Repo.Owner.Login, Name: pr.Head.Repo.Name}
		target.URL = "https://github.com/" + target.Repository.String() + ".git"
		target.Refspec = "HEAD:refs/heads/" + pr.Head.Ref
	}
	return target
}

func (s *WorktreeService) planPRCheckout(ctx context.Context, input PRWorktreeOptions, plan *model.PRWorktree, entries []model.Worktree, options *WorktreeAddOptions) error {
	if input.PullRequest.State != "open" {
		return fmt.Errorf("PR is not open; inspect it or use gha pr retire")
	}
	for _, entry := range entries {
		if entry.Branch == plan.Branch && entry.Path != plan.Path {
			return fmt.Errorf("branch %q is already checked out at %s; resume with that --worktree path", plan.Branch, entry.Path)
		}
	}
	if err := validatePRResume(plan); err != nil {
		return err
	}
	exists, err := s.localBranchExists(ctx, plan.Branch)
	if err != nil {
		return err
	}
	if exists {
		if err = s.inspectPRBranch(ctx, input, plan); err != nil {
			return err
		}
	}
	if plan.Worktree != nil {
		plan.Decision = "resume"
		plan.RecommendedActions = append(plan.RecommendedActions, model.RecommendedAction{Action: "resume", Reason: "continue in the existing checkout; no reset, fetch, or switch was performed"})
		return nil
	}
	if err = validateNewWorktreePath(plan.Path); err != nil {
		return err
	}
	options.NewBranch = !exists
	if exists {
		if _, err = s.PlanAdd(ctx, *options); err != nil {
			return err
		}
	} else {
		if err = NewBranchWriter(s.workdir).ValidateCreate(ctx, plan.Branch, "HEAD", false); err != nil {
			return err
		}
		plan.Fetch = "planned"
		plan.BranchState = "planned"
		plan.AssociationState = "planned"
	}
	plan.Decision = "create"
	plan.WorktreeState = "planned"
	return nil
}

func (s *WorktreeService) prComparison(ctx context.Context, published, local string) model.BranchComparison {
	comparison := model.BranchComparison{State: "unavailable", Message: "commit comparison unavailable; preserve local work"}
	shallow, err := s.run(ctx, "rev-parse", "--is-shallow-repository")
	if err != nil || strings.TrimSpace(shallow) != "false" {
		comparison.Message = "shallow or unavailable history cannot prove publication"
		return comparison
	}
	ahead, behind, err := NewBranchLister(s.workdir).divergence(ctx, local, published)
	if err != nil {
		return comparison
	}
	comparison.State = "identical"
	comparison.Message = "compared with the fresh provider PR head"
	comparison.AheadBy = ahead
	comparison.BehindBy = behind
	if ahead > 0 {
		comparison.State = "ahead"
	}
	if behind > 0 {
		comparison.State = "behind"
	}
	if ahead > 0 && behind > 0 {
		comparison.State = "diverged"
	}
	return comparison
}

func (s *WorktreeService) planPRRetirement(ctx context.Context, input PRWorktreeOptions, plan *model.PRWorktree) *model.PRWorktree {
	plan.Decision = "keep"
	pr := input.PullRequest
	if pr.State != "closed" {
		plan.Reasons = append(plan.Reasons, "PR is open or its terminal state is unknown")
	}
	entry := plan.Worktree
	if entry == nil {
		plan.Reasons = append(plan.Reasons, "target is not a registered worktree")
		return plan
	}
	if protectedPRCheckout(entry) {
		plan.Reasons = append(plan.Reasons, "main, selected, detached, locked, bare or unavailable checkout must be kept")
	}
	if entry.Branch != plan.Branch || entry.Branch == "" {
		plan.Reasons = append(plan.Reasons, "worktree branch does not match the explicit selection")
	}
	binding, bindingErr := s.prBranchBinding(ctx, entry.Branch)
	if bindingErr != nil || !matchingPRBinding(input, entry.Branch, binding) {
		plan.Reasons = append(plan.Reasons, "branch is not associated with this PR or its association is unavailable")
	}
	status, err := runGitAt(ctx, entry.Path, "status", "--porcelain=v1", "--untracked-files=all", "--ignored")
	if err != nil || strings.TrimSpace(status) != "" {
		plan.Reasons = append(plan.Reasons, "checkout contains changes, untracked/ignored files, or unavailable status")
	}
	if reason := prRetirementAmbiguity(ctx, entry.Path); reason != "" {
		plan.Reasons = append(plan.Reasons, reason)
	}
	plan.Comparison = s.prComparison(ctx, pr.Head.SHA, entry.Head)
	if plan.Comparison.State == "unavailable" || plan.Comparison.AheadBy > 0 {
		plan.Reasons = append(plan.Reasons, "local commits are unpublished to this PR or history is ambiguous")
	}
	if len(plan.Reasons) == 0 {
		plan.Decision = "remove"
		plan.WorktreeState = "planned"
	} else {
		plan.RecommendedActions = append(plan.RecommendedActions, model.RecommendedAction{Action: "keep", Reason: strings.Join(plan.Reasons, "; ")})
	}
	return plan
}

func prBinding(input PRWorktreeOptions) string {
	return fmt.Sprintf("%s#%d", strings.ToLower(input.Repository.String()), input.PullRequest.Number)
}

func (s *WorktreeService) createPRWorktree(ctx context.Context, input PRWorktreeOptions, plan *model.PRWorktree, options WorktreeAddOptions) (*model.PRWorktree, error) {
	if plan.Fetch == "planned" {
		ref := fmt.Sprintf("refs/gha/pull/%d/head", input.PullRequest.Number)
		source := fmt.Sprintf("+refs/pull/%d/head:%s", input.PullRequest.Number, ref)
		if _, err := s.run(ctx, "-c", "fetch.recurseSubmodules=false", "fetch", "--no-tags", "--no-write-fetch-head", "--refmap=", "origin", source); err != nil {
			plan.Fetch = "failed"
			plan.State = "failed"
			return plan, fmt.Errorf("fetch PR head: %w", err)
		}
		plan.Fetch = "completed"
		sha, err := s.run(ctx, "rev-parse", "--verify", ref+"^{commit}")
		if err != nil || strings.TrimSpace(sha) != input.PullRequest.Head.SHA {
			plan.State = "failed"
			return plan, fmt.Errorf("PR head changed during fetch; inspect and retry; only the private cached PR ref was updated")
		}
		options.From = input.PullRequest.Head.SHA
	}
	_, err := s.Add(ctx, options)
	if err != nil {
		plan.State = "failed"
		plan.WorktreeState = "failed"
		if options.NewBranch {
			exists, inspectErr := s.localBranchExists(ctx, plan.Branch)
			if inspectErr != nil {
				plan.BranchState = "unavailable"
			} else if exists {
				plan.BranchState = "completed"
			} else {
				plan.BranchState = "failed"
			}
		}
		return plan, err
	}
	plan.WorktreeState = "completed"
	if options.NewBranch {
		plan.BranchState = "completed"
	}
	plan.State = "completed"
	entries, _, inspectErr := s.entries(ctx)
	if inspectErr != nil {
		plan.State = "failed"
		return plan, fmt.Errorf("checkout created but status could not be read: %w", inspectErr)
	}
	for index := range entries {
		if entries[index].Path == plan.Path {
			plan.Worktree = &entries[index]
		}
	}
	if plan.Worktree != nil {
		plan.Comparison = s.prComparison(ctx, input.PullRequest.Head.SHA, plan.Worktree.Head)
	}
	if err = s.recordPRAssociation(ctx, input, plan); err != nil {
		return plan, err
	}
	plan.RecommendedActions = append(plan.RecommendedActions, model.RecommendedAction{Action: "work_in_checkout", Reason: "use the returned path; commit there and push HEAD to the explicit PR push destination"})
	return plan, nil
}

// PRFetchRepository binds PR operations to exactly one fetch origin, regardless
// of provider overrides. No fetch, remote creation, or checkout switch occurs.
func PRFetchRepository(ctx context.Context, path, override string) (model.RepositoryRef, error) {
	urls, err := runGitAt(ctx, path, "remote", "get-url", "--all", "origin")
	if err != nil {
		return model.RepositoryRef{}, fmt.Errorf("read checkout fetch origin: %w", err)
	}
	values := nonemptyLines(urls)
	if len(values) != 1 {
		return model.RepositoryRef{}, fmt.Errorf("origin must have exactly one fetch URL")
	}
	repo, err := ParseRepositoryRemote(values[0])
	if err != nil {
		return repo, err
	}
	if override != "" {
		selected, parseErr := ParseRepository(override)
		if parseErr != nil {
			return repo, parseErr
		}
		if !strings.EqualFold(selected.String(), repo.String()) {
			return repo, fmt.Errorf("selected repository %s does not match fetch origin %s", selected.String(), repo.String())
		}
	}
	return repo, nil
}

func protectedPRCheckout(entry *model.Worktree) bool {
	return entry.Main || entry.Current || entry.Bare || entry.Detached || entry.Locked || entry.Prunable
}

func validatePRResume(plan *model.PRWorktree) error {
	entry := plan.Worktree
	if entry == nil {
		return nil
	}
	if entry.Branch != plan.Branch || entry.Detached || entry.Bare || entry.Prunable || entry.Locked || entry.StatusState == "unavailable" {
		return fmt.Errorf("worktree path has a conflicting branch, lock, or unavailable checkout; inspect gha worktrees")
	}
	return nil
}

func (s *WorktreeService) prBranchBinding(ctx context.Context, branch string) (string, error) {
	value, err := s.run(ctx, "config", "--get", "branch."+branch+".gha-pr")
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return "", nil
	}
	return strings.TrimSpace(value), err
}

func matchingPRBinding(input PRWorktreeOptions, branch, binding string) bool {
	if binding != "" {
		return binding == prBinding(input)
	}
	return branch == defaultPRBranch(input)
}

func (s *WorktreeService) inspectPRBranch(ctx context.Context, input PRWorktreeOptions, plan *model.PRWorktree) error {
	plan.Comparison = s.prComparison(ctx, input.PullRequest.Head.SHA, "refs/heads/"+plan.Branch)
	binding, err := s.prBranchBinding(ctx, plan.Branch)
	if err != nil {
		return fmt.Errorf("inspect PR branch association: %w", err)
	}
	if binding == "" {
		plan.AssociationState = "planned"
	}
	if binding != "" && binding != prBinding(input) {
		return fmt.Errorf("local branch is associated with another PR; select another --branch")
	}
	if plan.Comparison.State == "unavailable" || (plan.Comparison.BehindBy > 0 && binding == "") {
		return fmt.Errorf("local branch %q is unrelated or ambiguous for this PR; select another --branch or inspect its commits", plan.Branch)
	}
	return nil
}

func prRetirementAmbiguity(ctx context.Context, path string) string {
	flags, err := runGitAt(ctx, path, "ls-files", "-v", "-z")
	if err != nil {
		return "tracked-file safety flags are unavailable"
	}
	for _, field := range strings.Split(flags, "\x00") {
		if field != "" && field[0] != 'H' {
			return "index flags can hide local changes; inspect assume-unchanged or skip-worktree entries"
		}
	}
	paths, err := runGitAt(ctx, path, "rev-parse", "--path-format=absolute", "--git-path", "MERGE_HEAD", "--git-path", "rebase-merge", "--git-path", "rebase-apply", "--git-path", "CHERRY_PICK_HEAD", "--git-path", "REVERT_HEAD", "--git-path", "sequencer", "--git-path", "BISECT_START")
	if err != nil {
		return "in-progress Git operations could not be inspected"
	}
	for _, file := range nonemptyLines(paths) {
		if _, statErr := os.Lstat(file); !os.IsNotExist(statErr) {
			return "an in-progress Git operation or unavailable metadata requires keeping the checkout"
		}
	}
	return ""
}

func (s *WorktreeService) recordPRAssociation(ctx context.Context, input PRWorktreeOptions, plan *model.PRWorktree) error {
	if plan.AssociationState != "planned" {
		return nil
	}
	if _, err := s.run(ctx, "config", "branch."+plan.Branch+".gha-pr", prBinding(input)); err != nil {
		plan.State = "failed"
		plan.AssociationState = "failed"
		return fmt.Errorf("checkout exists but PR association could not be recorded; retry checkout to associate it: %w", err)
	}
	plan.AssociationState = "completed"
	return nil
}
