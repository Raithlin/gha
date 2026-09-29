package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorktreeInventoryJSONUsesVersionedFields(t *testing.T) {
	inventory := WorktreeInventory{
		SchemaVersion: WorktreeInventorySchemaVersion,
		Path:          "/repo",
		Limit:         10,
		Total:         2,
		Truncated:     false,
		Worktrees: []Worktree{{
			Path:        "/repo-feature",
			Head:        "abc123",
			Branch:      "feature/example",
			Current:     true,
			Main:        false,
			StatusState: "clean",
		}},
	}

	encoded, err := json.Marshal(inventory)
	require.NoError(t, err)
	var decoded WorktreeInventory
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.Equal(t, inventory, decoded)
	assert.Contains(t, string(encoded), `"schema_version":"v1"`)
}

func TestWorktreeMutationRepresentsPreviewAndCompletion(t *testing.T) {
	mutation := WorktreeMutation{
		SchemaVersion: WorktreeMutationSchemaVersion,
		Operation:     "add",
		Path:          "/repo-feature",
		Branch:        "feature/example",
		From:          "main",
		DryRun:        true,
		State:         "planned",
	}
	assert.Equal(t, "v1", mutation.SchemaVersion)
	assert.True(t, mutation.DryRun)
	assert.Equal(t, "planned", mutation.State)
}
