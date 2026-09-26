package commands

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/pkg/model"
)

func TestCapabilitiesCommandProvidesCompleteVersionedInventory(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetArgs([]string{"capabilities", "--format", "json"})

	require.NoError(t, root.Execute())
	var capabilities model.Capabilities
	require.NoError(t, json.Unmarshal(output.Bytes(), &capabilities))
	assert.Equal(t, model.CapabilitiesSchemaVersion, capabilities.SchemaVersion)
	require.Len(t, capabilities.Commands, 23)
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "update", Status: "available", ReadOnly: false, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.GuidanceUpdateSchemaVersion, Notes: "Discovers the latest published GHA release (including prereleases) and refreshes skill and managed guidance for recorded agent installations; writes by default, --dry-run previews, and unavailable sources are reported; does not update the executable."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "agent install", Status: "available", ReadOnly: false, Notes: "Detects configured supported harnesses and installs bundled gha skills and guidance; --agent selects harnesses explicitly, --binary-only skips setup, and --dry-run previews without writing."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "agent list", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.AgentInstallationListSchemaVersion, Notes: "Lists harnesses recorded as configured by GHA, managed destinations, and whether guidance and skill files are present; does not detect unconfigured harnesses."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "agent uninstall", Status: "available", ReadOnly: false, Notes: "Removes GHA-managed guidance and skills for selected configured agents while retaining destinations shared by other configured agents; accepts comma-separated agents; --dry-run previews without writing."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "releases", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.ReleaseListSchemaVersion, Notes: "Bounded listing of published releases; drafts are excluded."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "release create-notes --since <timestamp>", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.ReleaseNotesSchemaVersion, Notes: "Generates bounded local release notes; does not publish a GitHub release."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "release publish <version>", Status: "available", ReadOnly: false, Formats: []string{"text", "json", "yaml"}, SchemaVersion: "v1", Notes: "Preflights the default-branch commit, tags, checks, release workflow, and opted-in reviewed notes; --dry-run previews without publishing."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "version", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.VersionInfoSchemaVersion, Notes: "Identifies the installed build version, commit, and build time."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "pr prepare", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.PullRequestPreparationSchemaVersion, Notes: "Drafts a title and description from bounded local commits and changed files; reports source signals and provider preflight."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "branches", Status: "available", ReadOnly: false, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.BranchInventorySchemaVersion, Notes: "Bounded branch inventory; origin refresh is explicit and --dry-run previews the fetch."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "branches cleanup", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.BranchCleanupSchemaVersion, Notes: "Read-only, bounded local cleanup candidates using an explicit reachability rule."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "pr create", Status: "available", ReadOnly: false, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.PullRequestPreparationSchemaVersion, Notes: "Runs the pull request preflight; --dry-run previews without creating."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "analyze", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.RepositoryAnalysisSchemaVersion, Notes: "Offline local Git worktree, history, storage, and largest-file analysis."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "branch show <name>", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.BranchInspectionSchemaVersion, Notes: "Single-branch inspection with explicit provider safety signals."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "branch publish <name>", Status: "available", ReadOnly: false, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.BranchPublicationSchemaVersion, Notes: "Preflights an existing local branch; explicit provider permission denial blocks publication, while unavailable permission is verified by the Git push; --dry-run previews without publishing."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "branch delete <name>", Status: "available", ReadOnly: false, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.BranchMutationSchemaVersion, Notes: "Requires explicit local/origin target; --force overrides documented safety guardrails; --dry-run previews without writing."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "tag publish <name>", Status: "available", ReadOnly: false, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.TagPublicationSchemaVersion, Notes: "Creates a tag at a selected commit and pushes that exact ref; --dry-run previews without writing."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "dashboard", Status: "unavailable", ReadOnly: true, Notes: "The TUI dashboard is not implemented in this build."})
}

func TestDashboardFailsExplicitlyInsteadOfPretendingToLaunch(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"dashboard"})

	err := root.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "dashboard is unavailable")
}
