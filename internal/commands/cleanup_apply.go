package commands

import (
	"bufio"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/raithlin/gha/internal/branch"
	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/pkg/model"
	"github.com/spf13/cobra"
)

type cleanupApplyOptions struct {
	path, base, format    string
	local, origin, dryRun bool
	force, worktrees      []string
}

type cleanupApplication struct {
	options    cleanupApplyOptions
	service    *branch.Service
	repository model.RepositoryRef
	names      []string
	plan       *model.CleanupExecution
	inputs     map[string]git.CleanupDeleteOptions
}

func newCleanupApplyCmd(service *branch.Service) *cobra.Command {
	options := cleanupApplyOptions{}
	cmd := &cobra.Command{Use: "apply <branch...>", Short: "Clean up explicitly selected merged PR branches and linked worktrees", SilenceUsage: true, SilenceErrors: true}
	cmd.Long = `Delete only explicitly named branches with --local, --origin, or both.
Fresh merged-PR evidence must match the exact local tip, including squash and
rebase merges. --worktree explicitly selects a linked checkout to retire first.
Default/base branches, primary/current checkouts and locked worktrees stay protected.

--dry-run returns the complete plan without prompting or writing. Otherwise the
operation executes after fresh revalidation. --force <branch> bypasses merge and
publication checks for that selected name and permits discarding dirty,
untracked or ignored files in its explicitly selected linked worktree. Repeat
--force for each intended override; it never applies to other branches.
Forced execution shows targets on stderr and requires typing each exact branch
name on stdin before any writes. Missing or mismatched confirmation stops all writes.
Origin writes retain permission/default/protection checks and use exact tip leases.
Results include completed, retained and failed effects and recovery guidance.`
	cmd.Args = func(_ *cobra.Command, args []string) error {
		if len(args) < 1 || len(args) > 100 {
			return fmt.Errorf("select between 1 and 100 explicit branches")
		}
		return nil
	}
	cmd.RunE = func(cmd *cobra.Command, names []string) error {
		format, err := output.ParseFormat(options.format)
		if err != nil {
			return err
		}
		application, err := prepareCleanupApplication(cmd, service, names, options)
		if err != nil {
			return renderCommandError(cmd, format, "branch_cleanup_failed", err)
		}
		if !options.dryRun {
			if err = application.confirm(cmd); err == nil {
				err = application.execute(cmd)
			}
		}
		if writeErr := output.CleanupExecution(cmd.OutOrStdout(), format, application.plan); writeErr != nil {
			return errors.Join(err, writeErr)
		}
		if err != nil {
			return renderCommandError(cmd, format, "branch_cleanup_failed", err)
		}
		return nil
	}
	cmd.Flags().StringVar(&options.path, "path", "", "Local source checkout")
	cmd.Flags().StringVar(&options.base, "base", "", "Local integration base (defaults to cached origin/HEAD)")
	cmd.Flags().StringVarP(&options.format, "format", "f", "text", "Output format (text, json, yaml)")
	cmd.Flags().BoolVar(&options.local, "local", false, "Delete selected local refs")
	cmd.Flags().BoolVar(&options.origin, "origin", false, "Delete selected origin refs with exact tip leases")
	cmd.Flags().BoolVar(&options.dryRun, "dry-run", false, "Preview without prompting or writing")
	cmd.Flags().StringArrayVar(&options.force, "force", nil, "Named selected branch to force; repeat per override; execution requires branch-name confirmation")
	cmd.Flags().StringArrayVar(&options.worktrees, "worktree", nil, "Explicit linked checkout to retire before deleting its selected branch; repeat per checkout")
	return cmd
}

func prepareCleanupApplication(cmd *cobra.Command, service *branch.Service, names []string, options cleanupApplyOptions) (*cleanupApplication, error) {
	if !options.local && !options.origin {
		return nil, fmt.Errorf("select --local and/or --origin")
	}
	selected := map[string]bool{}
	for _, name := range names {
		if selected[name] {
			return nil, fmt.Errorf("duplicate selected branch %q", name)
		}
		selected[name] = true
	}
	for _, name := range options.force {
		if !selected[name] {
			return nil, fmt.Errorf("--force %q must name a selected branch", name)
		}
	}
	repository, err := git.PRFetchRepository(cmd.Context(), options.path, "")
	if err != nil {
		return nil, err
	}
	local := git.NewBranchLister(options.path).WithCleanupSelection(names)
	application := &cleanupApplication{options: options, service: service.WithLister(local), repository: repository, names: names, plan: &model.CleanupExecution{SchemaVersion: model.CleanupExecutionSchemaVersion, DryRun: options.dryRun, Results: []*model.CleanupBranchResult{}}, inputs: map[string]git.CleanupDeleteOptions{}}
	inspection, err := application.inspect(cmd)
	if err != nil {
		return nil, err
	}
	paths, err := cleanupSelectedPaths(cmd, options, inspection)
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		candidate := cleanupEntry(inspection, name)
		result, input := application.planBranch(cmd, inspection, candidate, paths)
		application.plan.Results = append(application.plan.Results, result)
		application.inputs[name] = input
	}
	return application, nil
}

func (a *cleanupApplication) inspect(cmd *cobra.Command) (*model.BranchCleanup, error) {
	if _, err := git.PRFetchRepository(cmd.Context(), a.options.path, a.repository.String()); err != nil {
		return nil, err
	}
	return a.service.Cleanup(cmd.Context(), a.options.base, len(a.names), a.repository, nil, true)
}

func cleanupEntry(inspection *model.BranchCleanup, name string) *model.BranchCleanupCandidate {
	for _, entry := range append(append([]*model.BranchCleanupCandidate{}, inspection.Candidates...), inspection.Excluded...) {
		if entry.Name == name {
			return entry
		}
	}
	return nil
}

func cleanupSelectedPaths(cmd *cobra.Command, options cleanupApplyOptions, inspection *model.BranchCleanup) (map[string]bool, error) {
	paths := map[string]bool{}
	if len(options.worktrees) == 0 {
		return paths, nil
	}
	if !options.local {
		return nil, fmt.Errorf("--worktree requires --local")
	}
	inventory, err := git.NewWorktreeService(options.path).List(cmd.Context(), 100)
	if err != nil {
		return nil, err
	}
	for _, path := range options.worktrees {
		if !filepath.IsAbs(path) {
			path = filepath.Join(inventory.Path, path)
		}
		path = filepath.Clean(path)
		found := false
		for _, entry := range inspection.Excluded {
			if entry.WorktreePath == path {
				found = true
			}
		}
		if !found || paths[path] {
			return nil, fmt.Errorf("worktree %q is duplicate or does not belong to a selected branch", path)
		}
		paths[path] = true
	}
	return paths, nil
}

func (a *cleanupApplication) planBranch(cmd *cobra.Command, inspection *model.BranchCleanup, entry *model.BranchCleanupCandidate, paths map[string]bool) (*model.CleanupBranchResult, git.CleanupDeleteOptions) {
	result := &model.CleanupBranchResult{Name: entry.Name, SHA: entry.Local.SHA, Forced: containsName(a.options.force, entry.Name), State: "retained", Local: "not_requested", Origin: "not_requested", Worktree: "not_requested", Evidence: entry.MergeEvidence}
	if a.options.local {
		result.Local = "retained"
	}
	if a.options.origin {
		result.Origin = "retained"
	}
	input := git.CleanupDeleteOptions{Branch: entry.Name, ExpectedSHA: entry.Local.SHA, Base: inspection.Base, DefaultBranch: inspection.DefaultBranch, Local: a.options.local, Origin: a.options.origin, Force: result.Forced}
	if paths[entry.WorktreePath] {
		input.WorktreePath = entry.WorktreePath
		result.WorktreePath = entry.WorktreePath
		result.Worktree = "retained"
	}
	if inspection.DefaultBranch == "" {
		result.Reason = "default branch unavailable; retry provider inspection"
		return result, input
	}
	if !result.Forced && (entry.MergeEvidence == nil || entry.MergeEvidence.Reason != "merged_pr_head_matches") {
		result.Reason = entry.Reason
		return result, input
	}
	if a.options.origin {
		if err := a.verifyOriginPolicy(cmd, entry.Name); err != nil {
			result.Reason = err.Error()
			return result, input
		}
		pushURL, err := git.NewBranchWriter(a.options.path).CleanupPushTarget(cmd.Context())
		if err != nil {
			result.Reason = err.Error()
			return result, input
		}
		input.PushURL = pushURL
	}
	plan, err := git.NewBranchWriter(a.options.path).PlanCleanupDelete(cmd.Context(), input)
	if err != nil {
		result.Reason = err.Error()
		return result, input
	}
	plan.Evidence = entry.MergeEvidence
	if a.options.origin {
		input.ExpectedOriginSHA = &plan.OriginSHA
	}
	if result.Forced {
		plan.Reason = "branch-specific forced cleanup; confirmation required before writes"
	} else {
		plan.Reason = "fresh merged PR matches the exact inspected local tip"
	}
	return plan, input
}

func containsName(names []string, name string) bool {
	for _, selected := range names {
		if selected == name {
			return true
		}
	}
	return false
}

func (a *cleanupApplication) confirm(cmd *cobra.Command) error {
	reader := bufio.NewReader(cmd.InOrStdin())
	for _, result := range a.plan.Results {
		if !result.Forced || result.State != "planned" {
			continue
		}
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Force cleanup %q at local tip %s, origin tip %s: local=%s origin=%s worktree=%q. This may discard unpublished commits and dirty, untracked or ignored files.\nType the exact branch name %q to confirm: ", result.Name, result.SHA, result.OriginSHA, result.Local, result.Origin, result.WorktreePath, result.Name); err != nil {
			return err
		}
		answer, err := reader.ReadString('\n')
		if err != nil || strings.TrimSuffix(strings.TrimSuffix(answer, "\n"), "\r") != result.Name {
			for _, pending := range a.plan.Results {
				if pending.State == "planned" {
					pending.State = "retained"
					pending.Reason = "force confirmation was not supplied; no writes performed"
					if pending.Local == "planned" {
						pending.Local = "retained"
					}
					if pending.Origin == "planned" {
						pending.Origin = "retained"
					}
					if pending.Worktree == "planned" {
						pending.Worktree = "retained"
					}
				}
			}
			return fmt.Errorf("force confirmation for %q missing or mismatched; no cleanup performed", result.Name)
		}
	}
	return nil
}

func (a *cleanupApplication) execute(cmd *cobra.Command) error {
	var failures []error
	for index, previous := range a.plan.Results {
		if previous.State != "planned" {
			continue
		}
		inspection, err := a.inspect(cmd)
		if err != nil {
			failures = append(failures, a.retainAfterChange(previous, err))
			continue
		}
		entry := cleanupEntry(inspection, previous.Name)
		paths := map[string]bool{previous.WorktreePath: previous.WorktreePath != ""}
		current, input := a.planBranch(cmd, inspection, entry, paths)
		if current.State != "planned" || current.SHA != previous.SHA || current.OriginSHA != previous.OriginSHA || current.WorktreePath != previous.WorktreePath || input.DefaultBranch != a.inputs[previous.Name].DefaultBranch || input.PushURL != a.inputs[previous.Name].PushURL {
			failures = append(failures, a.retainAfterChange(previous, fmt.Errorf("cleanup state changed since preview: %s; inspect and retry", current.Reason)))
			continue
		}
		result, writeErr := git.NewBranchWriter(a.options.path).CleanupDelete(cmd.Context(), input)
		if result != nil {
			result.Evidence = current.Evidence
			a.plan.Results[index] = result
		}
		if writeErr != nil {
			if result == nil {
				writeErr = a.retainAfterChange(previous, writeErr)
			}
			failures = append(failures, writeErr)
		}
	}
	return errors.Join(failures...)
}

func (a *cleanupApplication) retainAfterChange(result *model.CleanupBranchResult, err error) error {
	result.State = "retained"
	result.Reason = err.Error()
	if result.Local == "planned" {
		result.Local = "retained"
	}
	if result.Origin == "planned" {
		result.Origin = "retained"
	}
	if result.Worktree == "planned" {
		result.Worktree = "retained"
	}
	return err
}

func (a *cleanupApplication) verifyOriginPolicy(cmd *cobra.Command, name string) error {
	if err := requireOriginWriteTarget(cmd, git.NewRepositoryResolver(""), a.options.path, a.repository.String()); err != nil {
		return err
	}
	safety := a.service.InspectSafety(cmd.Context(), a.repository, name)
	if safety.Permissions.State != "available" || safety.CanPush == nil || !*safety.CanPush {
		return fmt.Errorf("origin write permission unavailable or denied; resolve provider access")
	}
	if safety.DefaultBranch.State != "available" || safety.IsDefault == nil || *safety.IsDefault {
		return fmt.Errorf("origin default branch is protected or its identity is unavailable")
	}
	if safety.Protection.State != "available" || safety.Protected == nil || *safety.Protected {
		return fmt.Errorf("origin branch is protected or its protection status is unavailable")
	}
	return nil
}
