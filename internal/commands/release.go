package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/internal/review"
)

func newReleaseCmd(service *review.Service, resolver *git.RepositoryResolver) *cobra.Command {
	command := &cobra.Command{
		Use:   "release",
		Short: "Release workflows",
		Long: `Release workflows.

Use "gha release create-notes --since ..." to generate read-only release notes
and a contributor summary from merged pull requests.

The release window starts at --since (inclusive). The repository is taken from
--repo, the origin remote in --path, GHA_REPOSITORY, or the current directory's
origin remote (in that order).`,
	}
	command.AddCommand(newReleaseCreateNotesCmd(service, resolver))
	command.SilenceUsage = true
	command.SilenceErrors = true
	return command
}

func newReleaseCreateNotesCmd(service *review.Service, resolver *git.RepositoryResolver) *cobra.Command {
	options := &releaseNotesOptions{}
	command := &cobra.Command{
		Use:   "create-notes",
		Short: "Generate release notes from merged pull requests",
		Long: `Generate read-only release notes and a contributor summary from merged pull requests.

The release window starts at --since (inclusive). The repository is taken from
--repo, the origin remote in --path, GHA_REPOSITORY, or the current directory's
origin remote (in that order).`,
		Args: noArgsWithFormat(&options.format),
		RunE: newReleaseNotesRunE(service, resolver, options),
	}
	addReleaseNotesFlags(command, options)
	command.SilenceUsage = true
	command.SilenceErrors = true
	return command
}

type releaseNotesOptions struct {
	repository string
	path       string
	format     string
	sinceText  string
	limit      int
}

func addReleaseNotesFlags(command *cobra.Command, options *releaseNotesOptions) {
	command.Flags().StringVarP(&options.repository, "repo", "r", "", "Repository to inspect (owner/repo)")
	command.Flags().StringVar(&options.path, "path", "", "Local Git checkout whose origin selects the repository")
	command.Flags().StringVarP(&options.format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().StringVar(&options.sinceText, "since", "", "Inclusive release-window start: RFC 3339, local datetime, or date (required)")
	command.Flags().IntVarP(&options.limit, "limit", "l", 100, "Maximum merged pull requests to include (1-100)")
}

func newReleaseNotesRunE(service *review.Service, resolver *git.RepositoryResolver, options *releaseNotesOptions) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		outputFormat, err := output.ParseFormat(options.format)
		if err != nil {
			return err
		}
		if options.limit < 1 || options.limit > 100 {
			return renderCommandError(cmd, outputFormat, "invalid_argument", fmt.Errorf("limit must be between 1 and 100"))
		}
		if options.sinceText == "" {
			return renderCommandError(cmd, outputFormat, "invalid_argument", fmt.Errorf("required flag(s) \"since\" not set"))
		}
		since, err := parseReleaseSince(options.sinceText, time.Local)
		if err != nil {
			return renderCommandError(cmd, outputFormat, "invalid_argument", err)
		}
		target, err := resolver.ResolveAtPath(cmd.Context(), options.repository, options.path)
		if err != nil {
			return renderCommandError(cmd, outputFormat, "repository_resolution_failed", err)
		}
		notes, err := service.ReleaseNotes(cmd.Context(), target, since, options.limit)
		if err != nil {
			return renderCommandError(cmd, outputFormat, "release_notes_failed", err)
		}
		return output.ReleaseNotes(cmd.OutOrStdout(), outputFormat, notes)
	}
}

// parseReleaseSince accepts explicit RFC 3339 timestamps unchanged. A
// timezone-less ISO timestamp, including a date alone, is interpreted in the
// current machine timezone so its calendar date has the expected local meaning.
func parseReleaseSince(value string, location *time.Location) (time.Time, error) {
	if timestamp, err := time.Parse(time.RFC3339, value); err == nil {
		return timestamp, nil
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02"} {
		if timestamp, err := time.ParseInLocation(layout, value, location); err == nil {
			return timestamp, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid --since %q (use RFC 3339, a local ISO datetime such as 2025-09-01T09:30, or a date such as 2025-09-01)", value)
}
