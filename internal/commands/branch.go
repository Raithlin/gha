package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/branch"
	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/pkg/model"
)

// newBranchCmd constructs branch lifecycle operations. Only inspection is
// available at this stage; future mutations will be explicit subcommands.
func newBranchCmd(service *branch.Service, resolver *git.RepositoryResolver) *cobra.Command {
	command := &cobra.Command{
		Use:   "branch",
		Short: "Inspect and manage one branch",
		Long:  "Inspect one branch and its available provider safety signals. Future write operations will be explicit subcommands.",
	}
	command.AddCommand(newBranchShowCmd(service, resolver))
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
