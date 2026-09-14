package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/internal/review"
	"github.com/raithlin/gha/pkg/model"
)

// newPRCmd constructs the guarded pull-request preparation and creation flow.
func newPRCmd(service *review.Service, resolver *git.RepositoryResolver) *cobra.Command {
	command := &cobra.Command{
		Use:   "pr",
		Short: "Prepare or create one pull request",
		Long:  "Prepare a decision-ready pull request plan, then create it only with explicit confirmation.",
	}
	command.AddCommand(newPRPrepareCmd(service, resolver), newPRCreateCmd(service, resolver))
	return command
}

func newPRPrepareCmd(service *review.Service, resolver *git.RepositoryResolver) *cobra.Command {
	options := &prOptions{}
	command := &cobra.Command{
		Use:   "prepare",
		Short: "Preview one pull request with base, head, and safety signals",
		Long: `Resolve a base and head, compare them with provider data, and report an existing open pull request.

This command is read-only. Use gha pr create --confirm after reviewing the plan.`,
		Args: noArgsWithFormat(&options.format),
		RunE: func(cmd *cobra.Command, _ []string) error {
			preparation, format, err := preparePullRequest(cmd, service, resolver, options, false)
			if err != nil {
				return renderCommandError(cmd, format, "pull_request_preparation_failed", err)
			}
			return output.PullRequestPreparation(cmd.OutOrStdout(), format, preparation)
		},
	}
	addPRFlags(command, options, false)
	return command
}

func newPRCreateCmd(service *review.Service, resolver *git.RepositoryResolver) *cobra.Command {
	options := &prOptions{}
	command := &cobra.Command{
		Use:   "create",
		Short: "Create a reviewed pull request with explicit confirmation",
		Long: `Run the same guarded preflight as gha pr prepare, then create the pull request only with --confirm.

Use --dry-run to return the creation plan without writing to the provider.`,
		Args: noArgsWithFormat(&options.format),
		RunE: func(cmd *cobra.Command, _ []string) error {
			preparation, format, err := preparePullRequest(cmd, service, resolver, options, options.dryRun)
			if err != nil {
				return renderCommandError(cmd, format, "pull_request_preparation_failed", err)
			}
			if options.dryRun {
				return output.PullRequestPreparation(cmd.OutOrStdout(), format, preparation)
			}
			if !options.confirm {
				return renderCommandError(cmd, format, "pull_request_creation_failed", fmt.Errorf("pull request creation requires --confirm; use gha pr prepare or --dry-run to review the plan"))
			}
			created, err := service.CreatePreparedPullRequest(cmd.Context(), preparation)
			if err != nil {
				return renderCommandError(cmd, format, "pull_request_creation_failed", err)
			}
			preparation.Creation = "completed"
			preparation.CreatedPullRequest = created
			return output.PullRequestPreparation(cmd.OutOrStdout(), format, preparation)
		},
	}
	addPRFlags(command, options, true)
	return command
}

type prOptions struct {
	repository string
	path       string
	format     string
	title      string
	body       string
	head       string
	base       string
	confirm    bool
	dryRun     bool
}

func addPRFlags(command *cobra.Command, options *prOptions, create bool) {
	command.Flags().StringVarP(&options.repository, "repo", "r", "", "Repository for the pull request (owner/repo)")
	command.Flags().StringVar(&options.path, "path", "", "Local checkout used to resolve repository and default head")
	command.Flags().StringVarP(&options.format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().StringVar(&options.title, "title", "", "Pull request title (required)")
	command.Flags().StringVar(&options.body, "body", "", "Pull request description")
	command.Flags().StringVar(&options.head, "head", "", "Head branch; defaults to the current local branch")
	command.Flags().StringVar(&options.base, "base", "", "Base branch; defaults to the provider default branch")
	if create {
		command.Flags().BoolVar(&options.confirm, "confirm", false, "Confirm creating the reviewed pull request")
		command.Flags().BoolVar(&options.dryRun, "dry-run", false, "Show the creation plan without writing")
	}
	command.SilenceUsage = true
	command.SilenceErrors = true
}

func preparePullRequest(cmd *cobra.Command, service *review.Service, resolver *git.RepositoryResolver, options *prOptions, dryRun bool) (*model.PullRequestPreparation, output.Format, error) {
	format, err := output.ParseFormat(options.format)
	if err != nil {
		return nil, format, err
	}
	if service == nil {
		return nil, format, fmt.Errorf("pull request preparation is not configured")
	}
	if resolver == nil {
		return nil, format, fmt.Errorf("repository resolution is not configured")
	}
	if strings.TrimSpace(options.title) == "" {
		return nil, format, fmt.Errorf("--title is required")
	}
	head := strings.TrimSpace(options.head)
	if head == "" {
		head, err = git.CurrentBranch(cmd.Context(), options.path)
		if err != nil {
			return nil, format, fmt.Errorf("resolve pull request head: %w", err)
		}
	}
	target, err := resolver.ResolveAtPath(cmd.Context(), options.repository, options.path)
	if err != nil {
		return nil, format, err
	}
	preparation, err := service.PreparePullRequest(cmd.Context(), review.PreparePullRequestInput{Repository: target, Title: strings.TrimSpace(options.title), Body: options.body, Head: head, Base: strings.TrimSpace(options.base), DryRun: dryRun})
	return preparation, format, err
}
