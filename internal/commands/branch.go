package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/branch"
	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/pkg/model"
)

// newBranchCmd constructs explicit branch lifecycle operations.
func newBranchCmd(service *branch.Service, resolver *git.RepositoryResolver) *cobra.Command {
	command := &cobra.Command{
		Use:   "branch",
		Short: "Inspect and manage one branch",
		Long:  "Inspect, create, rename, or delete one branch. Remote changes require explicit confirmation.",
	}
	command.AddCommand(newBranchShowCmd(service, resolver), newBranchCreateCmd(), newBranchRenameCmd(service, resolver), newBranchDeleteCmd(service, resolver))
	return command
}

func newBranchShowCmd(service *branch.Service, resolver *git.RepositoryResolver) *cobra.Command {
	var format string
	var path string
	var repository string
	command := &cobra.Command{
		Use:   "show <name>",
		Short: "Inspect one branch and its safety signals",
		Long: `Inspect one local or cached-origin branch without changing Git or provider state.

Provider signals include open pull requests, protection, permission,
default-branch, and mergeability facts when the configured provider supports
them. Unavailable facts are reported explicitly.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			if service == nil {
				return renderCommandError(cmd, outputFormat, "branch_inspection_failed", fmt.Errorf("branch inspection is not configured"))
			}
			inspectionService := service
			if path != "" {
				inspectionService = service.WithLister(git.NewBranchLister(path))
			}

			var selected model.RepositoryRef
			var repositoryErr error
			if resolver == nil {
				repositoryErr = fmt.Errorf("repository resolution is not configured")
			} else {
				selected, repositoryErr = resolver.ResolveAtPath(cmd.Context(), repository, path)
			}
			inspection, err := inspectionService.Show(cmd.Context(), args[0], selected, repositoryErr)
			if err != nil {
				return renderCommandError(cmd, outputFormat, "branch_inspection_failed", err)
			}
			return output.BranchInspection(cmd.OutOrStdout(), outputFormat, inspection)
		},
	}
	command.SilenceUsage = true
	command.SilenceErrors = true
	command.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().StringVar(&path, "path", "", "Local Git checkout to inspect")
	command.Flags().StringVar(&repository, "repo", "", "Repository for provider safety signals (owner/repo)")
	return command
}

func newBranchCreateCmd() *cobra.Command {
	var format, path, from string
	var publish, confirmOrigin, dryRun bool
	command := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a local branch, optionally publishing it to origin",
		Long:  "Create a local branch at --from (or HEAD). --publish changes both local and origin state and requires --confirm-origin.",
		Args:  exactArgsWithFormat(1, &format),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			if publish && !confirmOrigin && !dryRun {
				return renderCommandError(cmd, outputFormat, "branch_mutation_failed", fmt.Errorf("remote changes require --confirm-origin"))
			}
			result := newBranchMutation("create", args[0], "", from, dryRun, "planned", targetState(publish, "planned"))
			if dryRun {
				return output.BranchMutation(cmd.OutOrStdout(), outputFormat, result)
			}
			writer := git.NewBranchWriter(path)
			if err := writer.CreateLocal(cmd.Context(), args[0], from); err != nil {
				return renderCommandError(cmd, outputFormat, "branch_mutation_failed", fmt.Errorf("create local branch: %w", err))
			}
			result.Local = "completed"
			if publish {
				if err := writer.Publish(cmd.Context(), args[0]); err != nil {
					return renderCommandError(cmd, outputFormat, "branch_mutation_failed", fmt.Errorf("publish branch to origin: %w", err))
				}
				result.Origin = "completed"
			}
			return output.BranchMutation(cmd.OutOrStdout(), outputFormat, result)
		},
	}
	addMutationFlags(command, &format, &path, &dryRun)
	command.Flags().StringVar(&from, "from", "", "Start point for the new local branch (default HEAD)")
	command.Flags().BoolVar(&publish, "publish", false, "Publish the new branch to origin")
	command.Flags().BoolVar(&confirmOrigin, "confirm-origin", false, "Confirm the requested origin change")
	return command
}

func newBranchRenameCmd(service *branch.Service, resolver *git.RepositoryResolver) *cobra.Command {
	var format, path, repository string
	var origin, confirmOrigin, dryRun, force bool
	command := &cobra.Command{
		Use:   "rename <old> <new>",
		Short: "Rename a local branch, optionally renaming it on origin",
		Long:  "Rename a local branch. --origin also creates the new origin name and removes the old one; it requires --confirm-origin and safety checks.",
		Args:  exactArgsWithFormat(2, &format),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			if origin && !confirmOrigin && !dryRun {
				return renderCommandError(cmd, outputFormat, "branch_mutation_failed", fmt.Errorf("remote changes require --confirm-origin"))
			}
			result := newBranchMutation("rename", args[0], args[1], "", dryRun, "planned", targetState(origin, "planned"))
			if dryRun {
				return output.BranchMutation(cmd.OutOrStdout(), outputFormat, result)
			}
			if origin && !force {
				if _, err := requireRemoteDestructionSafety(cmd, service, resolver, path, repository, args[0]); err != nil {
					return renderCommandError(cmd, outputFormat, "branch_mutation_failed", err)
				}
			}
			writer := git.NewBranchWriter(path)
			if err := writer.RenameLocal(cmd.Context(), args[0], args[1]); err != nil {
				return renderCommandError(cmd, outputFormat, "branch_mutation_failed", fmt.Errorf("rename local branch: %w", err))
			}
			result.Local = "completed"
			if origin {
				if err := writer.RenameOrigin(cmd.Context(), args[0], args[1]); err != nil {
					return renderCommandError(cmd, outputFormat, "branch_mutation_failed", fmt.Errorf("rename branch on origin: %w", err))
				}
				result.Origin = "completed"
			}
			return output.BranchMutation(cmd.OutOrStdout(), outputFormat, result)
		},
	}
	addMutationFlags(command, &format, &path, &dryRun)
	command.Flags().BoolVar(&origin, "origin", false, "Rename the branch on origin too")
	command.Flags().BoolVar(&confirmOrigin, "confirm-origin", false, "Confirm the requested origin change")
	command.Flags().BoolVar(&force, "force", false, "Override origin branch safety guardrails")
	command.Flags().StringVar(&repository, "repo", "", "Repository for origin safety signals (owner/repo)")
	return command
}

//nolint:gocyclo // This command validates and executes two independently selectable targets.
func newBranchDeleteCmd(service *branch.Service, resolver *git.RepositoryResolver) *cobra.Command {
	var format, path, repository string
	var local, origin, confirmOrigin, dryRun, force bool
	command := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an explicitly selected local branch, origin branch, or both",
		Long:  "Select --local, --origin, or both. If a selected local branch is current and not the default branch, GHA switches to the default branch before deleting it. Origin deletion requires --confirm-origin. Origin default, protected, or unverifiable branches require --force.",
		Args:  exactArgsWithFormat(1, &format),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			if !local && !origin {
				return renderCommandError(cmd, outputFormat, "invalid_argument", fmt.Errorf("select at least one target with --local and/or --origin"))
			}
			if origin && !confirmOrigin && !dryRun {
				return renderCommandError(cmd, outputFormat, "branch_mutation_failed", fmt.Errorf("remote changes require --confirm-origin"))
			}
			result := newBranchMutation("delete", args[0], "", "", dryRun, targetState(local, "planned"), targetState(origin, "planned"))
			writer := git.NewBranchWriter(path)
			if dryRun {
				checkedOut, err := currentBranchDeleteSwitch(cmd.Context(), writer, args[0], local, "")
				if err != nil {
					return renderCommandError(cmd, outputFormat, "branch_mutation_failed", err)
				}
				result.CheckedOut = checkedOut
				return output.BranchMutation(cmd.OutOrStdout(), outputFormat, result)
			}
			defaultBranch := ""
			if origin && !force {
				var err error
				defaultBranch, err = requireRemoteDestructionSafety(cmd, service, resolver, path, repository, args[0])
				if err != nil {
					return renderCommandError(cmd, outputFormat, "branch_mutation_failed", err)
				}
			}
			if local {
				checkedOut, err := currentBranchDeleteSwitch(cmd.Context(), writer, args[0], true, defaultBranch)
				if err != nil {
					return renderCommandError(cmd, outputFormat, "branch_mutation_failed", err)
				}
				if checkedOut != "" {
					if err := writer.Switch(cmd.Context(), checkedOut); err != nil {
						return renderCommandError(cmd, outputFormat, "branch_mutation_failed", fmt.Errorf("switch to default branch %q before delete: %w", checkedOut, err))
					}
					result.CheckedOut = checkedOut
				}
				if err := writer.DeleteLocal(cmd.Context(), args[0], force); err != nil {
					return renderCommandError(cmd, outputFormat, "branch_mutation_failed", fmt.Errorf("delete local branch: %w", err))
				}
				result.Local = "completed"
			}
			if origin {
				if err := writer.DeleteOrigin(cmd.Context(), args[0]); err != nil {
					return renderCommandError(cmd, outputFormat, "branch_mutation_failed", fmt.Errorf("delete branch from origin: %w", err))
				}
				result.Origin = "completed"
			}
			return output.BranchMutation(cmd.OutOrStdout(), outputFormat, result)
		},
	}
	addMutationFlags(command, &format, &path, &dryRun)
	command.Flags().BoolVar(&local, "local", false, "Delete the local branch")
	command.Flags().BoolVar(&origin, "origin", false, "Delete the branch from origin")
	command.Flags().BoolVar(&confirmOrigin, "confirm-origin", false, "Confirm the requested origin deletion")
	command.Flags().BoolVar(&force, "force", false, "Override Git and origin branch safety guardrails")
	command.Flags().StringVar(&repository, "repo", "", "Repository for origin safety signals (owner/repo)")
	return command
}

func addMutationFlags(command *cobra.Command, format, path *string, dryRun *bool) {
	command.SilenceUsage = true
	command.SilenceErrors = true
	command.Flags().StringVarP(format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().StringVar(path, "path", "", "Local Git checkout to change")
	command.Flags().BoolVar(dryRun, "dry-run", false, "Show the requested changes without writing")
}

func newBranchMutation(operation, name, newName, from string, dryRun bool, local, origin string) *model.BranchMutation {
	return &model.BranchMutation{SchemaVersion: model.BranchMutationSchemaVersion, Operation: operation, Name: name, NewName: newName, From: from, DryRun: dryRun, Local: local, Origin: origin}
}

func targetState(selected bool, state string) string {
	if selected {
		return state
	}
	return "not_requested"
}

func currentBranchDeleteSwitch(ctx context.Context, writer *git.BranchWriter, name string, local bool, defaultBranch string) (string, error) {
	if !local {
		return "", nil
	}
	current, err := writer.CurrentBranch(ctx)
	if err != nil {
		return "", fmt.Errorf("read current branch before delete: %w", err)
	}
	if current != name {
		return "", nil
	}
	if defaultBranch == "" {
		defaultBranch, err = writer.DefaultBranch(ctx)
		if err != nil {
			return "", fmt.Errorf("cannot delete the current branch without a default branch to switch to: %w", err)
		}
	}
	if defaultBranch == name {
		return "", fmt.Errorf("refusing to delete the current default branch")
	}
	return defaultBranch, nil
}

func requireRemoteDestructionSafety(cmd *cobra.Command, service *branch.Service, resolver *git.RepositoryResolver, path, repository, name string) (string, error) {
	if service == nil || resolver == nil {
		return "", fmt.Errorf("cannot verify origin branch safety; retry with --force only after independently verifying the branch")
	}
	selected, err := resolver.ResolveAtPath(cmd.Context(), repository, path)
	if err != nil {
		return "", fmt.Errorf("cannot resolve repository for origin branch safety: %w; retry with --force only after independently verifying the branch", err)
	}
	inspection, err := service.WithLister(git.NewBranchLister(path)).Show(cmd.Context(), name, selected, nil)
	if err != nil {
		return "", fmt.Errorf("cannot inspect origin branch safety: %w; retry with --force only after independently verifying the branch", err)
	}
	safety := inspection.Safety
	if safety.Permissions.State != "available" || safety.CanPush == nil || !*safety.CanPush {
		return "", fmt.Errorf("origin write permission is unavailable or denied; retry with --force only after independently verifying access")
	}
	if safety.DefaultBranch.State != "available" || safety.IsDefault == nil {
		return "", fmt.Errorf("origin default-branch status is unavailable; retry with --force only after independently verifying the branch")
	}
	if *safety.IsDefault {
		return "", fmt.Errorf("refusing to remove the origin default branch; use --force only if this is intentional")
	}
	if safety.Protection.State != "available" || safety.Protected == nil {
		return "", fmt.Errorf("origin protection status is unavailable; retry with --force only after independently verifying the branch")
	}
	if *safety.Protected {
		return "", fmt.Errorf("refusing to remove a protected origin branch; use --force only if this is intentional")
	}
	return safety.DefaultBranchName, nil
}
