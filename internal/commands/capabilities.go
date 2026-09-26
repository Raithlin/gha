package commands

import (
	"github.com/spf13/cobra"

	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/internal/release"
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
		RunE: func(cmd *cobra.Command, _ []string) error {
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
			{Command: "agent list", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.AgentInstallationListSchemaVersion, Notes: "Lists harnesses recorded as configured by GHA, managed destinations, and whether guidance and skill files are present; does not detect unconfigured harnesses."},
			{Command: "agent install", Status: "available", ReadOnly: false, Notes: "Installs bundled gha skills and supported guidance for Codex, Claude Code, Pi, OpenCode, GitHub Copilot, or Gemini CLI; accepts comma-separated agents and records configured destinations; supports --dry-run and requires --confirm to write."},
			{Command: "agent uninstall", Status: "available", ReadOnly: false, Notes: "Removes GHA-managed guidance and skills for selected configured agents while retaining destinations shared by other configured agents; accepts comma-separated agents, supports --dry-run, and requires --confirm to write."},
			{Command: "version", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.VersionInfoSchemaVersion, Notes: "Identifies the installed build version, commit, and build time."},
			{Command: "pr prepare", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.PullRequestPreparationSchemaVersion, Notes: "Resolves pull request base and head, compares branches, and detects existing open pull requests."},
			{Command: "pr create", Status: "available", ReadOnly: false, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.PullRequestPreparationSchemaVersion, Notes: "Runs the pull request preflight; --dry-run does not write and creation requires --confirm."},
			{Command: "analyze", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.RepositoryAnalysisSchemaVersion, Notes: "Offline local Git worktree, history, storage, and largest-file analysis."},
			{Command: "branches", Status: "available", ReadOnly: false, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.BranchInventorySchemaVersion, Notes: "Bounded branch inventory; origin refresh is explicit and confirmed."},
			{Command: "branches cleanup", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.BranchCleanupSchemaVersion, Notes: "Read-only, bounded local cleanup candidates using an explicit reachability rule."},
			{Command: "branch show <name>", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.BranchInspectionSchemaVersion, Notes: "Single-branch inspection with explicit provider safety signals."},
			{Command: "branch create <name>", Status: "available", ReadOnly: false, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.BranchMutationSchemaVersion, Notes: "Creates locally; --publish changes origin only with --confirm-origin."},
			{Command: "branch publish <name>", Status: "available", ReadOnly: false, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.BranchPublicationSchemaVersion, Notes: "Preflights an existing local branch; explicit provider permission denial blocks publication, while unavailable permission is verified by the Git push; --confirm-origin is required."},
			{Command: "branch rename <old> <new>", Status: "available", ReadOnly: false, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.BranchMutationSchemaVersion, Notes: "Renames locally; --origin requires --confirm-origin and safety checks."},
			{Command: "branch delete <name>", Status: "available", ReadOnly: false, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.BranchMutationSchemaVersion, Notes: "Requires explicit local/origin target; origin deletion requires confirmation."},
			{Command: "tag publish <name>", Status: "available", ReadOnly: false, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.TagPublicationSchemaVersion, Notes: "Creates a tag at a selected commit and pushes that exact ref; origin write requires --confirm-origin."},
			{Command: "prs", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.PullRequestListSchemaVersion, Notes: "Bounded pull request listing; supports filters and --since."},
			{Command: "review <number>", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.ReviewSummarySchemaVersion, Notes: "Decision-ready inspection of one pull request."},
			{Command: "releases", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.ReleaseListSchemaVersion, Notes: "Bounded listing of published releases; drafts are excluded."},
			{Command: "release create-notes --since <timestamp>", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.ReleaseNotesSchemaVersion, Notes: "Generates bounded local release notes; does not publish a GitHub release."},
			{Command: "release publish <version>", Status: "available", ReadOnly: false, Formats: []string{"text", "json", "yaml"}, SchemaVersion: release.SchemaVersion, Notes: "Preflights the default-branch commit, tags, checks, release workflow, and opted-in reviewed notes; an annotated tag push requires --confirm-origin."},
			{Command: "dashboard", Status: "unavailable", ReadOnly: true, Notes: "The TUI dashboard is not implemented in this build."},
			{Command: "capabilities", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.CapabilitiesSchemaVersion, Notes: "Versioned inventory of this command surface."},
		},
	}
}
