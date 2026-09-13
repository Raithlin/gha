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
	var repository, format, sinceText string
	var limit int

	command := &cobra.Command{
		Use:   "release",
		Short: "Generate release notes from merged pull requests",
		Long: `Generate read-only release notes and a contributor summary from merged pull requests.

The release window starts at --since (inclusive). The repository is taken from
--repo, GHA_REPOSITORY, or the current directory's origin remote (in that order).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit < 1 || limit > 100 {
				return fmt.Errorf("limit must be between 1 and 100")
			}
			since, err := parseReleaseSince(sinceText, time.Local)
			if err != nil {
				return err
			}
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			target, err := resolver.Resolve(cmd.Context(), repository)
			if err != nil {
				return err
			}
			notes, err := service.ReleaseNotes(cmd.Context(), target, since, limit)
			if err != nil {
				return err
			}
			return output.ReleaseNotes(cmd.OutOrStdout(), outputFormat, notes)
		},
	}

	command.Flags().StringVarP(&repository, "repo", "r", "", "Repository to inspect (owner/repo)")
	command.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")
	command.Flags().StringVar(&sinceText, "since", "", "Inclusive release-window start: RFC 3339, local datetime, or date (required)")
	command.Flags().IntVarP(&limit, "limit", "l", 100, "Maximum merged pull requests to include (1-100)")
	_ = command.MarkFlagRequired("since")
	return command
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
