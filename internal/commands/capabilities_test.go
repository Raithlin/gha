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
	require.Len(t, capabilities.Commands, 12)
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "agent install", Status: "available", ReadOnly: false, Notes: "Copies bundled gha guidance for Codex or Claude Code; supports --dry-run and requires --confirm to write."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "analyze", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.RepositoryAnalysisSchemaVersion, Notes: "Offline local Git worktree, history, storage, and largest-file analysis."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "branch show <name>", Status: "available", ReadOnly: true, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.BranchInspectionSchemaVersion, Notes: "Single-branch inspection with explicit provider safety signals."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "branch delete <name>", Status: "available", ReadOnly: false, Formats: []string{"text", "json", "yaml"}, SchemaVersion: model.BranchMutationSchemaVersion, Notes: "Requires explicit local/origin target; origin deletion requires confirmation."})
	assert.Contains(t, capabilities.Commands, model.Capability{Command: "dashboard", Status: "unavailable", ReadOnly: true, Notes: "The TUI dashboard is not implemented in this build."})
}

func TestDashboardFailsExplicitlyInsteadOfPretendingToLaunch(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"dashboard"})

	err := root.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "dashboard is unavailable")
}
