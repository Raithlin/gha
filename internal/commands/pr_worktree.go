package commands

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/internal/review"
	"github.com/raithlin/gha/pkg/model"
	"github.com/spf13/cobra"
)

type prWorktreeOptions struct {
	repository, path, worktree, branch, format string
	dryRun                                     bool
}

func newPRWorktreeCmd(operation string, service *review.Service, resolver *git.RepositoryResolver) *cobra.Command {
	options := &prWorktreeOptions{}
	command := &cobra.Command{Use: operation + " <number>", Short: "Create or resume an isolated PR checkout with review and safety signals", Args: exactArgsWithFormat(1, &options.format), SilenceUsage: true, SilenceErrors: true}
	if operation == "retire" {
		command.Short = "Assess and safely retire a closed PR checkout, retaining its branch"
	}
	command.Long = `Combine fresh PR review signals with local branch and worktree safety.

Select an explicit PR number and --worktree path. --path selects the source
checkout; --repo, when supplied, must match that checkout's fetch origin.
The current checkout is never switched. The default call executes; --dry-run
previews without fetching or writing. Existing checkouts resume without reset.
Checkout fetches only when creating a new local branch. Retirement keeps open
PRs and dirty, ignored, unpublished, locked, or ambiguous work, and never
removes a branch. No provider write or push is performed.

For new development, use gha worktree add, publish the branch, then gha pr create.
For an existing PR, use gha pr checkout, work at the returned path and push to
the explicit PR push destination. Repeat checkout to resume; do not create a
second PR. Fork destinations may require separate push permission.`
	command.RunE = func(cmd *cobra.Command, args []string) error {
		format, err := output.ParseFormat(options.format)
		if err != nil {
			return err
		}
		result, err := runPRWorktree(cmd, operation, args[0], options, service, resolver)
		if result != nil {
			if writeErr := output.PRWorktree(cmd.OutOrStdout(), format, result); writeErr != nil {
				return errors.Join(err, writeErr)
			}
		}
		if err != nil {
			return renderCommandError(cmd, format, "pr_worktree_failed", err)
		}
		return nil
	}
	command.Flags().StringVarP(&options.repository, "repo", "r", "", "PR repository; must match the local fetch origin")
	command.Flags().StringVar(&options.path, "path", "", "Source local checkout (defaults to the current directory)")
	command.Flags().StringVar(&options.worktree, "worktree", "", "Explicit checkout path to create, resume or retire (relative to source root)")
	command.Flags().StringVar(&options.branch, "branch", "", "Local PR branch; defaults to head branch for same-repo PRs, pr/<number> for forks")
	command.Flags().StringVarP(&options.format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().BoolVar(&options.dryRun, "dry-run", false, "Preview without fetching or writing")
	return command
}

func runPRWorktree(cmd *cobra.Command, operation, numberText string, options *prWorktreeOptions, service *review.Service, resolver *git.RepositoryResolver) (*model.PRWorktree, error) {
	number, err := strconv.Atoi(numberText)
	if err != nil || number < 1 {
		return nil, fmt.Errorf("invalid pull request number %q", numberText)
	}
	if service == nil || resolver == nil {
		return nil, fmt.Errorf("PR worktree workflow is not configured")
	}
	if options.worktree == "" {
		return nil, fmt.Errorf("select an explicit worktree path with --worktree")
	}
	target, err := git.PRFetchRepository(cmd.Context(), options.path, options.repository)
	if err != nil {
		return nil, err
	}
	summary, pr, reviewState, err := inspectPRWorktree(cmd, service, target, number)
	if err != nil {
		return nil, err
	}
	if pr == nil || pr.Number != number {
		return nil, fmt.Errorf("provider did not return the selected PR identity")
	}
	input := git.PRWorktreeOptions{Operation: operation, Path: options.worktree, Branch: options.branch, Repository: target, PullRequest: pr, DryRun: true}
	local := git.NewWorktreeService(options.path)
	plan, err := local.PRWorktree(cmd.Context(), input)
	if err != nil {
		return nil, err
	}
	if !options.dryRun {
		fresh, getErr := service.Get(cmd.Context(), target, number)
		if getErr != nil {
			return nil, getErr
		}
		if !samePRTarget(pr, fresh) {
			return nil, fmt.Errorf("PR head or state changed during preflight; inspect and retry")
		}
		if _, targetErr := git.PRFetchRepository(cmd.Context(), options.path, target.String()); targetErr != nil {
			return nil, targetErr
		}
		input.PullRequest = fresh
		input.DryRun = false
		plan, err = local.PRWorktree(cmd.Context(), input)
	}
	decoratePRWorktree(plan, summary, reviewState)
	return plan, err
}

func samePRTarget(previous, current *model.PullRequest) bool {
	return current != nil && previous.Number == current.Number && previous.State == current.State && previous.MergedAt == current.MergedAt && reflect.DeepEqual(previous.Head, current.Head)
}

func inspectPRWorktree(cmd *cobra.Command, service *review.Service, target model.RepositoryRef, number int) (*model.ReviewSummary, *model.PullRequest, model.ProviderSignal, error) {
	summary, err := service.Inspect(cmd.Context(), target, number)
	if err == nil {
		return summary, summary.PullRequest, model.ProviderSignal{State: "available"}, nil
	}
	state := model.ProviderSignal{State: "unavailable", Message: err.Error()}
	pr, getErr := service.Get(cmd.Context(), target, number)
	return nil, pr, state, getErr
}

func decoratePRWorktree(plan *model.PRWorktree, summary *model.ReviewSummary, state model.ProviderSignal) {
	if plan == nil {
		return
	}
	plan.Review = summary
	plan.ReviewState = state
	if summary != nil {
		plan.RecommendedActions = append(plan.RecommendedActions, summary.RecommendedActions...)
	}
	if plan.Push.State == "unavailable" {
		plan.RecommendedActions = append(plan.RecommendedActions, model.RecommendedAction{Action: "inspect_push_target", Reason: "head repository is unavailable; do not guess a fork or publish another PR"})
	}
}
