package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/pkg/model"
)

func TestWorktreeInventoryRendersTextAndStructuredData(t *testing.T) {
	inventory := &model.WorktreeInventory{
		SchemaVersion: "v1",
		Path:          "/repo",
		Limit:         10,
		Total:         1,
		Worktrees: []model.Worktree{{
			Path:        "/repo-feature",
			Head:        "abcdef0123456789",
			Branch:      "feature/test",
			Current:     true,
			StatusState: "clean",
		}},
	}
	var text bytes.Buffer
	require.NoError(t, WorktreeInventory(&text, Text, inventory))
	assert.Contains(t, text.String(), "feature/test")
	assert.Contains(t, text.String(), "clean")
	assert.Contains(t, text.String(), "current")

	var structured bytes.Buffer
	require.NoError(t, WorktreeInventory(&structured, JSON, inventory))
	assert.JSONEq(t, `{"schema_version":"v1","path":"/repo","limit":10,"total":1,"truncated":false,"worktrees":[{"path":"/repo-feature","head":"abcdef0123456789","branch":"feature/test","detached":false,"bare":false,"current":true,"main":false,"locked":false,"prunable":false,"status_state":"clean"}]}`, structured.String())
}

func TestWorktreeInventoryTextExplainsEmptyAndUnavailableStates(t *testing.T) {
	var text bytes.Buffer
	require.NoError(t, WorktreeInventory(&text, Text, &model.WorktreeInventory{
		Path:      "/repo",
		Limit:     1,
		Total:     1,
		Worktrees: []model.Worktree{{Path: "/gone", Head: "abc", Detached: true, StatusState: "unavailable", StatusMessage: "missing checkout"}},
	}))
	assert.Contains(t, text.String(), "Detached")
	assert.Contains(t, text.String(), "unavailable")
	assert.Contains(t, text.String(), "missing checkout")
}

func TestWorktreeInventoryTextCoversLabelsFlagsDetailsAndTruncation(t *testing.T) {
	inventory := &model.WorktreeInventory{
		Path:      "/repo",
		Limit:     1,
		Total:     2,
		Truncated: true,
		Worktrees: []model.Worktree{{
			Path:           "/repo/linked",
			Head:           "abc",
			Branch:         "feature/test",
			Main:           true,
			Current:        true,
			Locked:         true,
			LockReason:     "in use",
			Prunable:       true,
			PrunableReason: "missing checkout",
			StatusState:    "unavailable",
			StatusMessage:  "status failed",
		}},
	}
	var text bytes.Buffer
	require.NoError(t, WorktreeInventory(&text, Text, inventory))
	for _, expected := range []string{"Additional worktrees omitted", "main, current, locked, prunable", "in use", "missing checkout", "status failed"} {
		assert.Contains(t, text.String(), expected)
	}
}

func TestWorktreeInventoryTextIdentifiesBareAndUnbranchedWorktrees(t *testing.T) {
	for _, worktree := range []model.Worktree{
		{Path: "/bare", Bare: true, Head: "abc", StatusState: "not_applicable"},
		{Path: "/detached", Detached: true, Head: "def", StatusState: "clean"},
		{Path: "/empty-branch", Head: "ghi", StatusState: "clean"},
	} {
		var text bytes.Buffer
		require.NoError(t, WorktreeInventory(&text, Text, &model.WorktreeInventory{Worktrees: []model.Worktree{worktree}}))
	}
}

func TestWorktreeMutationRendersPreviewAndCompletion(t *testing.T) {
	mutation := &model.WorktreeMutation{Operation: "add", Path: "/repo-feature", Branch: "feature/test", From: "main", DryRun: true, State: "planned"}
	var text bytes.Buffer
	require.NoError(t, WorktreeMutation(&text, Text, mutation))
	assert.Contains(t, text.String(), "Dry run")
	assert.Contains(t, text.String(), "feature/test")
	assert.Contains(t, text.String(), "planned")

	var structured bytes.Buffer
	require.NoError(t, WorktreeMutation(&structured, JSON, mutation))
	assert.Contains(t, structured.String(), `"operation": "add"`)
	assert.Contains(t, structured.String(), `"state": "planned"`)
}

func TestWorktreeMutationRendersCompletionWithoutOptionalFields(t *testing.T) {
	var text bytes.Buffer
	require.NoError(t, WorktreeMutation(&text, Text, &model.WorktreeMutation{Operation: "remove", Path: "/repo/linked", State: "completed"}))
	assert.Contains(t, text.String(), "completed")
	assert.NotContains(t, text.String(), "Branch:")
	assert.NotContains(t, text.String(), "From:")
	assert.NotContains(t, text.String(), "Dry run")
}

func TestWorktreeRenderersRejectNilAndWriterErrors(t *testing.T) {
	assert.Error(t, WorktreeInventory(&bytes.Buffer{}, Text, nil))
	assert.Error(t, WorktreeMutation(&bytes.Buffer{}, Text, nil))
	assert.Error(t, WorktreeInventory(failingWriter{}, Text, &model.WorktreeInventory{}))
	assert.Error(t, WorktreeMutation(failingWriter{}, Text, &model.WorktreeMutation{}))
}

func TestWorktreeTextRenderersPropagateEveryWriteError(t *testing.T) {
	inventory := &model.WorktreeInventory{
		Path:      "/repo",
		Total:     1,
		Truncated: true,
		Worktrees: []model.Worktree{{
			Path: "/linked", Head: "abc", Branch: "feature/test", Main: true, Current: true,
			Locked: true, LockReason: "busy", Prunable: true, PrunableReason: "stale",
			StatusState: "unavailable", StatusMessage: "no status",
		}},
	}
	for write := 1; write <= 9; write++ {
		assert.Error(t, WorktreeInventory(&failingWriterAt{failAt: write}, Text, inventory), "inventory write %d", write)
	}
	mutation := &model.WorktreeMutation{Operation: "add", Path: "/linked", Branch: "feature/test", From: "main", State: "planned", DryRun: true}
	for write := 1; write <= 5; write++ {
		assert.Error(t, WorktreeMutation(&failingWriterAt{failAt: write}, Text, mutation), "mutation write %d", write)
	}
}

type failingWriterAt struct {
	failAt int
	writes int
}

func (writer *failingWriterAt) Write(content []byte) (int, error) {
	writer.writes++
	if writer.writes == writer.failAt {
		return 0, assert.AnError
	}
	return len(content), nil
}
