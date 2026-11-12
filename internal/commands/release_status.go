package commands

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/internal/release"
	"github.com/spf13/cobra"
)

var releaseStatusTag = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*$`)

func newReleaseStatusCmd(resolver *git.RepositoryResolver, provider release.StatusProvider) *cobra.Command {
	options := &releaseStatusOptions{timeout: 5 * time.Minute}
	command := &cobra.Command{
		Use:   "status <tag>",
		Short: "Inspect release publication without changing the tag or workflow",
		Long:  `Read the remote tag, its exact commit and matching tag-triggered workflow run, and the hosted GitHub Release. Use --workflow when more than one workflow matches. --wait polls until publication, failure, cancellation, or timeout and returns the latest observation. --repo selects the provider target; --path selects a local checkout only for repository resolution.`,
		Args:  exactArgsWithFormat(1, &options.format),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReleaseStatus(cmd, args[0], options, resolver, provider)
		},
	}
	command.Flags().StringVarP(&options.repository, "repo", "r", "", "Provider repository (owner/repo)")
	command.Flags().StringVar(&options.path, "path", "", "Local checkout whose origin selects the repository")
	command.Flags().StringVar(&options.workflow, "workflow", "", "Exact release workflow path when multiple workflows match")
	command.Flags().StringVarP(&options.format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().BoolVar(&options.wait, "wait", false, "Wait for a terminal publication outcome")
	command.Flags().DurationVar(&options.timeout, "timeout", 5*time.Minute, "Maximum time to wait (up to 30m)")
	command.SilenceUsage, command.SilenceErrors = true, true
	return command
}

type releaseStatusOptions struct {
	repository, path, workflow, format string
	wait                               bool
	timeout                            time.Duration
}

func runReleaseStatus(cmd *cobra.Command, tag string, options *releaseStatusOptions, resolver *git.RepositoryResolver, provider release.StatusProvider) error {
	format, err := output.ParseFormat(options.format)
	if err != nil {
		return err
	}
	if !validReleaseStatusTag(tag) {
		return renderCommandError(cmd, format, "invalid_argument", fmt.Errorf("invalid tag name %q", tag))
	}
	if options.timeout <= 0 || options.timeout > 30*time.Minute || !options.wait && cmd.Flags().Changed("timeout") {
		return renderCommandError(cmd, format, "invalid_argument", fmt.Errorf("--timeout requires --wait and a duration between 1ns and 30m"))
	}
	if resolver == nil || provider == nil {
		return renderCommandError(cmd, format, "release_status_failed", fmt.Errorf("release status is not configured"))
	}
	target, err := resolver.ResolveAtPath(cmd.Context(), options.repository, options.path)
	if err != nil {
		return renderCommandError(cmd, format, "repository_resolution_failed", err)
	}
	service := release.NewStatusService(provider)
	var result *release.Status
	if options.wait {
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer stop()
		result, err = service.Wait(ctx, target, tag, options.workflow, options.timeout, 3*time.Second)
	} else {
		result, err = service.Check(cmd.Context(), target, tag, options.workflow)
	}
	if err != nil {
		return renderCommandError(cmd, format, "release_status_failed", err)
	}
	return output.ReleaseStatus(cmd.OutOrStdout(), format, result)
}

func validReleaseStatusTag(tag string) bool {
	return releaseStatusTag.MatchString(tag) && !strings.Contains(tag, "..") && !strings.Contains(tag, "//") && !strings.HasSuffix(tag, ".lock") && !strings.HasSuffix(tag, "/") && !strings.HasSuffix(tag, ".")
}
