package commands

import (
	"context"
	"fmt"
	"strings"

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
		Long:  "Inspect, create, publish, rename, or delete one branch. Use --dry-run to inspect a mutation before execution.",
	}
	command.AddCommand(newBranchShowCmd(service, resolver), newBranchCreateCmd(), newBranchPublishCmd(service, resolver), newBranchRenameCmd(service, resolver), newBranchDeleteCmd(service, resolver))
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
	var publish, dryRun bool
	command := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a local branch, optionally publishing it to origin",
		Long:  "Create a local branch at --from (or HEAD). --publish also publishes it to origin.",
		Args:  exactArgsWithFormat(1, &format),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
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
	return command
}

func newBranchPublishCmd(service *branch.Service, resolver *git.RepositoryResolver) *cobra.Command {
	var format, path, repository string
	var dryRun bool
	command := &cobra.Command{
		Use:   "publish <name>",
		Short: "Publish an existing local branch through a guarded origin preflight",
		Long: `Inspect an existing local branch before publishing it to origin and setting its upstream.

The preflight reports the origin target, cached origin state, local upstream and
divergence, and provider push permission. It never fetches. This workflow is
for a committed local branch without an upstream; use git push -u for a
straightforward publish. An explicit permission denial blocks publication. If
permission data is unavailable, the authenticated Git push determines whether
publication succeeds. Add --dry-run to inspect the plan without pushing.`,
		Args: exactArgsWithFormat(1, &format),
		RunE: func(cmd *cobra.Command, args []string) error {
			publication, outputFormat, err := prepareBranchPublication(cmd, service, resolver, args[0], repository, path, dryRun)
			if err != nil {
				return renderCommandError(cmd, outputFormat, "branch_publication_failed", err)
			}
			if dryRun {
				return output.BranchPublication(cmd.OutOrStdout(), outputFormat, publication)
			}
			if err := validateBranchPublication(publication); err != nil {
				return renderCommandError(cmd, outputFormat, "branch_publication_failed", err)
			}
			if err := git.NewBranchWriter(path).Publish(cmd.Context(), publication.Name); err != nil {
				return renderCommandError(cmd, outputFormat, "branch_publication_failed", fmt.Errorf("publish branch to origin: %w", err))
			}
			publication.Publication = "completed"
			refreshPublishedBranch(cmd.Context(), publication, path)
			return output.BranchPublication(cmd.OutOrStdout(), outputFormat, publication)
		},
	}
	addMutationFlags(command, &format, &path, &dryRun)
	command.Flags().StringVar(&repository, "repo", "", "Repository for origin push-permission checks (owner/repo)")
	return command
}

func prepareBranchPublication(cmd *cobra.Command, service *branch.Service, resolver *git.RepositoryResolver, name, repository, path string, dryRun bool) (*model.BranchPublication, output.Format, error) {
	formatValue, err := output.ParseFormat(flagValue(cmd, "format"))
	if err != nil {
		return nil, formatValue, err
	}
	if service == nil {
		return nil, formatValue, fmt.Errorf("branch publication is not configured")
	}
	if resolver == nil {
		return nil, formatValue, fmt.Errorf("repository resolution is not configured")
	}
	target, err := resolver.ResolveAtPath(cmd.Context(), repository, path)
	if err != nil {
		return nil, formatValue, fmt.Errorf("resolve repository for origin publication: %w", err)
	}
	inspection, err := service.WithLister(git.NewBranchLister(path)).Show(cmd.Context(), name, target, nil)
	if err != nil {
		return nil, formatValue, fmt.Errorf("inspect branch for publication: %w", err)
	}
	if inspection.Local == nil {
		return nil, formatValue, fmt.Errorf("branch %q is not a local branch", name)
	}
	if inspection.OriginState != "cached" {
		return nil, formatValue, fmt.Errorf("origin is not configured for branch publication")
	}
	if strings.TrimSpace(inspection.Local.SHA) == "" {
		return nil, formatValue, fmt.Errorf("branch %q has no committed tip", name)
	}
	if inspection.Local.Upstream != "" {
		return nil, formatValue, fmt.Errorf("branch %q already tracks %s; use git push for a straightforward update", name, inspection.Local.Upstream)
	}
	return &model.BranchPublication{
		SchemaVersion: model.BranchPublicationSchemaVersion,
		Repository:    inspection.Repository,
		Name:          name,
		Origin:        inspection.Origin,
		OriginState:   inspection.OriginState,
		Target:        "origin/" + name,
		Local:         inspection.Local,
		OriginBranch:  inspection.OriginBranch,
		Permissions:   inspection.Safety.Permissions,
		CanPush:       inspection.Safety.CanPush,
		DryRun:        dryRun,
		Publication:   "planned",
	}, formatValue, nil
}

func validateBranchPublication(publication *model.BranchPublication) error {
	if publication.Permissions.State == "unavailable" {
		// GitHub's permission response is optional; the authenticated Git
		// transport will be authoritative when publication is attempted.
		return nil
	}
	if publication.Permissions.State != "available" || publication.CanPush == nil {
		return fmt.Errorf("origin push permission is unavailable; resolve provider access and retry")
	}
	if !*publication.CanPush {
		return fmt.Errorf("origin push permission is denied")
	}
	return nil
}

func refreshPublishedBranch(ctx context.Context, publication *model.BranchPublication, path string) {
	inspection, err := git.NewBranchLister(path).Inspect(ctx, publication.Name)
	if err != nil || inspection.Local == nil {
		publication.Local.Upstream = publication.Target
		publication.Local.DivergenceState = "unavailable"
		publication.Local.DivergenceMessage = "publication completed; local divergence could not be refreshed"
		return
	}
	publication.Local = inspection.Local
	publication.OriginBranch = inspection.OriginBranch
}

func flagValue(cmd *cobra.Command, name string) string {
	value, err := cmd.Flags().GetString(name)
	if err != nil {
		return ""
	}
	return value
}

func newBranchRenameCmd(service *branch.Service, resolver *git.RepositoryResolver) *cobra.Command {
	var format, path, repository string
	var origin, dryRun, force bool
	command := &cobra.Command{
		Use:   "rename <old> <new>",
		Short: "Rename a local branch, optionally renaming it on origin",
		Long:  "Rename a local branch. --origin also creates the new origin name and removes the old one after safety checks.",
		Args:  exactArgsWithFormat(2, &format),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
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
	command.Flags().BoolVar(&force, "force", false, "Override origin branch safety guardrails")
	command.Flags().StringVar(&repository, "repo", "", "Repository for origin safety signals (owner/repo)")
	return command
}

//nolint:gocyclo // This command validates and executes two independently selectable targets.
func newBranchDeleteCmd(service *branch.Service, resolver *git.RepositoryResolver) *cobra.Command {
	var format, path, repository string
	var local, origin, dryRun, force bool
	command := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an explicitly selected local branch, origin branch, or both",
		Long:  "Select --local, --origin, or both. If a selected local branch is current and not the default branch, GHA switches to the default branch before deleting it. Origin default, protected, or unverifiable branches require --force.",
		Args:  exactArgsWithFormat(1, &format),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			if !local && !origin {
				return renderCommandError(cmd, outputFormat, "invalid_argument", fmt.Errorf("select at least one target with --local and/or --origin"))
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
