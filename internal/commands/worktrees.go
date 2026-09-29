package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/pkg/model"
)

// newWorktreesCmd constructs the read-only linked-worktree inventory command.
func newWorktreesCmd() *cobra.Command {
	var format, path string
	var limit int
	command := &cobra.Command{
		Use:   "worktrees",
		Short: "Inspect linked Git worktrees",
		Long: `List worktrees registered with the current Git repository, including
branch, detached, lock, prune, checkout-status, and current-path information.
Use --path to select any checkout in the repository's worktree set.`,
		Args: noArgsWithFormat(&format),
		RunE: func(cmd *cobra.Command, _ []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			if limit < 1 || limit > 100 {
				return renderCommandError(cmd, outputFormat, "invalid_argument", fmt.Errorf("limit must be between 1 and 100"))
			}
			inventory, err := git.NewWorktreeService(path).List(cmd.Context(), limit)
			if err != nil {
				return renderCommandError(cmd, outputFormat, "worktree_inventory_failed", err)
			}
			return output.WorktreeInventory(cmd.OutOrStdout(), outputFormat, inventory)
		},
	}
	command.SilenceUsage = true
	command.SilenceErrors = true
	command.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().StringVar(&path, "path", "", "Local Git checkout used to find registered worktrees")
	command.Flags().IntVarP(&limit, "limit", "l", 30, "Maximum worktrees to return (1-100)")
	return command
}

// newWorktreeCmd constructs the worktree lifecycle command group.
func newWorktreeCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "worktree",
		Short: "Add or remove a linked Git worktree",
	}
	command.AddCommand(newWorktreeAddCmd(), newWorktreeRemoveCmd())
	return command
}

func newWorktreeAddCmd() *cobra.Command {
	var format, path, branch, from string
	var newBranch, dryRun bool
	command := &cobra.Command{
		Use:   "add <path>",
		Short: "Add a linked worktree using an existing or new local branch",
		Long: `Add a linked worktree at <path>. Select an existing local branch with
--branch, or create a new branch with --branch and --new-branch. --from selects
the start point for a new branch and defaults to HEAD. Use --dry-run to validate
the plan without creating the worktree; otherwise the operation runs by default.`,
		Args: exactArgsWithFormat(1, &format),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			service := git.NewWorktreeService(path)
			options := git.WorktreeAddOptions{Path: args[0], Branch: branch, From: from, NewBranch: newBranch}
			var mutation *model.WorktreeMutation
			if dryRun {
				mutation, err = service.PlanAdd(cmd.Context(), options)
			} else {
				mutation, err = service.Add(cmd.Context(), options)
			}
			if err != nil {
				return renderCommandError(cmd, outputFormat, "worktree_mutation_failed", err)
			}
			return output.WorktreeMutation(cmd.OutOrStdout(), outputFormat, mutation)
		},
	}
	addWorktreeMutationFlags(command, &format, &path, &dryRun)
	command.Flags().StringVar(&branch, "branch", "", "Local branch to check out or create")
	command.Flags().BoolVar(&newBranch, "new-branch", false, "Create the selected branch in the new worktree")
	command.Flags().StringVar(&from, "from", "", "Start point for a new branch (default HEAD)")
	return command
}

func newWorktreeRemoveCmd() *cobra.Command {
	var format, path string
	var dryRun bool
	command := &cobra.Command{
		Use:   "remove <path>",
		Short: "Remove a clean, unlocked linked worktree",
		Long: `Remove one linked worktree after validating that its path is registered
with this repository and its checkout is clean and unlocked. The main worktree
cannot be removed. Use --dry-run to review without removing; otherwise the
operation runs by default. Git's own removal checks remain enabled.`,
		Args: exactArgsWithFormat(1, &format),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			service := git.NewWorktreeService(path)
			var mutation *model.WorktreeMutation
			if dryRun {
				mutation, err = service.PlanRemove(cmd.Context(), args[0])
			} else {
				mutation, err = service.Remove(cmd.Context(), args[0])
			}
			if err != nil {
				return renderCommandError(cmd, outputFormat, "worktree_mutation_failed", err)
			}
			return output.WorktreeMutation(cmd.OutOrStdout(), outputFormat, mutation)
		},
	}
	addWorktreeMutationFlags(command, &format, &path, &dryRun)
	return command
}

func addWorktreeMutationFlags(command *cobra.Command, format, path *string, dryRun *bool) {
	command.SilenceUsage = true
	command.SilenceErrors = true
	command.Flags().StringVarP(format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().StringVar(path, "path", "", "Local checkout used to resolve the repository")
	command.Flags().BoolVar(dryRun, "dry-run", false, "Validate and show the operation without writing")
}
