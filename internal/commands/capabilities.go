package commands

import (
	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/pkg/model"
)

// newCapabilitiesCmd constructs the complete, versioned command inventory.
func newCapabilitiesCmd() *cobra.Command {
	var format string
	command := &cobra.Command{
		Use:   "capabilities",
		Short: "List command capabilities for people and agents",
		Long: `List every command in this build and whether it is safe and available.

Use --format json for the stable automation contract.`,
		Args: noArgsWithFormat(&format),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, err := output.ParseFormat(format)
			if err != nil {
				return err
			}
			return output.Capabilities(cmd.OutOrStdout(), outputFormat, ghaCapabilities())
		},
	}
	command.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")
	command.SilenceUsage = true
	command.SilenceErrors = true
	return command
}

func ghaCapabilities() *model.Capabilities {
	return &model.Capabilities{
		SchemaVersion: model.CapabilitiesSchemaVersion,
		Commands: []model.Capability{
			{Command: "branches", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.BranchInventorySchemaVersion, Notes: "Bounded local and cached origin branch inventory."},
			{Command: "prs", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.PullRequestListSchemaVersion, Notes: "Bounded pull request listing; supports filters and --since."},
			{Command: "review <number>", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.ReviewSummarySchemaVersion, Notes: "Decision-ready inspection of one pull request."},
			{Command: "release --since <timestamp>", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.ReleaseNotesSchemaVersion, Notes: "Generates bounded local release notes; does not publish a GitHub release."},
			{Command: "dashboard", Status: "unavailable", ReadOnly: true, Notes: "The TUI dashboard is not implemented in this build."},
			{Command: "capabilities", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.CapabilitiesSchemaVersion, Notes: "Versioned inventory of this command surface."},
		},
	}
}
