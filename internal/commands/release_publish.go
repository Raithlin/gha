package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/internal/release"
)

func newReleasePublishCmd(resolver *git.RepositoryResolver, services ...*release.Service) *cobra.Command {
	var format, path, repository, workflow string
	var dryRun, confirmOrigin bool
	command := &cobra.Command{
		Use:   "publish <version>",
		Short: "Plan and publish an annotated SemVer release tag",
		Long: `Inspect the checked-out default branch, origin tip, existing tags, CI checks, and tag-triggered release workflow.

Use --dry-run to review the exact tag and remote effect. --confirm-origin is required to create and push the annotated tag. A successful push triggers the release workflow; it does not itself prove that a GitHub Release was published.`,
		Args: exactArgsWithFormat(1, &format),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			if resolver == nil {
				return renderCommandError(cmd, outputFormat, "release_publication_failed", fmt.Errorf("repository resolution is not configured"))
			}
			if len(services) == 0 || services[0] == nil {
				return renderCommandError(cmd, outputFormat, "release_publication_failed", fmt.Errorf("release publication is not configured"))
			}
			if !dryRun && !confirmOrigin {
				return renderCommandError(cmd, outputFormat, "release_publication_failed", fmt.Errorf("release publication requires --confirm-origin; use --dry-run to inspect the plan"))
			}
			target, err := resolver.ResolveAtPath(cmd.Context(), repository, path)
			if err != nil {
				return renderCommandError(cmd, outputFormat, "repository_resolution_failed", err)
			}
			service := services[0]
			if path != "" || workflow != "" {
				service = release.NewService(release.GitCheckout{Path: path, Workflow: workflow}, service.Provider())
			}
			plan, err := service.Prepare(cmd.Context(), target, args[0], path, dryRun)
			if err != nil {
				return renderCommandError(cmd, outputFormat, "release_publication_failed", err)
			}
			if dryRun {
				return output.ReleasePublication(cmd.OutOrStdout(), outputFormat, plan)
			}
			if !plan.Ready {
				return renderCommandError(cmd, outputFormat, "release_publication_failed", fmt.Errorf("release preflight blocked: %v", plan.Blockers))
			}
			if err := service.Publish(cmd.Context(), plan); err != nil {
				if renderErr := output.ReleasePublication(cmd.OutOrStdout(), outputFormat, plan); renderErr != nil {
					return renderErr
				}
				return renderCommandError(cmd, outputFormat, "release_publication_failed", err)
			}
			return output.ReleasePublication(cmd.OutOrStdout(), outputFormat, plan)
		},
	}
	command.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().StringVar(&path, "path", "", "Local checkout to publish from")
	command.Flags().StringVarP(&repository, "repo", "r", "", "Provider repository (owner/repo)")
	command.Flags().StringVar(&workflow, "workflow", "", "Committed workflow path when multiple tag-triggered workflows match")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Show the publication plan without writing")
	command.Flags().BoolVar(&confirmOrigin, "confirm-origin", false, "Confirm creating and pushing the annotated tag")
	command.SilenceUsage = true
	command.SilenceErrors = true
	return command
}
