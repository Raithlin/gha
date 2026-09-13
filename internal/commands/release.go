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
			since, err := time.Parse(time.RFC3339, sinceText)
			if err != nil {
				return fmt.Errorf("invalid --since %q (use an RFC 3339 timestamp, for example 2026-09-01T00:00:00Z)", sinceText)
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
	command.Flags().StringVar(&sinceText, "since", "", "Inclusive RFC 3339 start of the release window (required)")
	command.Flags().IntVarP(&limit, "limit", "l", 100, "Maximum merged pull requests to include (1-100)")
	_ = command.MarkFlagRequired("since")
	return command
}
