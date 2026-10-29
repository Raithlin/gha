package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestCleanupEvidenceAndWorktreeReasonsRenderTogether(t *testing.T) {
	result := &model.BranchCleanup{Rule: "fresh_merged_pr_head_matches", Base: "main", Candidates: []*model.BranchCleanupCandidate{}, Excluded: []*model.BranchCleanupCandidate{{Name: "topic", Reason: "checked_out_in_worktree", WorktreePath: "/tmp/linked", MergeEvidence: &model.CleanupMergeEvidence{State: "fresh", Reason: "merged_pr_head_matches", Repository: model.RepositoryRef{Owner: "acme", Name: "project"}, PullRequest: &model.PullRequest{Number: 7}, Message: "exact head matches"}, WorktreeRetirement: &model.PRWorktree{Decision: "keep", DryRun: true, Reasons: []string{"ignored files must be kept"}}}}}
	var text, jsonOutput bytes.Buffer
	require.NoError(t, BranchCleanup(&text, Text, result))
	require.Contains(t, text.String(), "fresh merged PR")
	require.Contains(t, text.String(), "acme/project#7, fresh")
	require.Contains(t, text.String(), "Worktree retirement preview: keep")
	require.Contains(t, text.String(), "ignored files must be kept")
	require.NoError(t, BranchCleanup(&jsonOutput, JSON, result))
	var decoded model.BranchCleanup
	require.NoError(t, json.Unmarshal(jsonOutput.Bytes(), &decoded))
	require.True(t, decoded.Excluded[0].WorktreeRetirement.DryRun)
	require.Equal(t, 7, decoded.Excluded[0].MergeEvidence.PullRequest.Number)
	probe := &failAfterWriter{failAt: 1000}
	require.NoError(t, BranchCleanup(probe, Text, result))
	for failAt := 1; failAt <= probe.writes; failAt++ {
		writer := &failAfterWriter{failAt: failAt}
		require.Error(t, BranchCleanup(writer, Text, result))
	}
	result.Excluded[0].MergeEvidence.PullRequest = nil
	var unavailable bytes.Buffer
	require.NoError(t, BranchCleanup(&unavailable, Text, result))
	require.Contains(t, unavailable.String(), "acme/project, fresh")
}

func TestCleanupExecutionPreservesEffectsAndRecovery(t *testing.T) {
	result := &model.CleanupExecution{SchemaVersion: model.CleanupExecutionSchemaVersion, Results: []*model.CleanupBranchResult{{Name: "topic", State: "failed", Local: "retained", Origin: "failed", Worktree: "completed", WorktreePath: "/tmp/linked", Forced: true, Reason: "origin changed", Recovery: "recreate checkout from retained local branch"}}}
	var text, jsonOutput bytes.Buffer
	require.NoError(t, CleanupExecution(&text, Text, result))
	require.Contains(t, text.String(), "origin: failed")
	require.Contains(t, text.String(), "worktree: completed")
	require.Contains(t, text.String(), "recreate checkout")
	require.NoError(t, CleanupExecution(&jsonOutput, JSON, result))
	var decoded model.CleanupExecution
	require.NoError(t, json.Unmarshal(jsonOutput.Bytes(), &decoded))
	require.Equal(t, "retained", decoded.Results[0].Local)
	probe := &failAfterWriter{failAt: 100}
	require.NoError(t, CleanupExecution(probe, Text, result))
	for failAt := 1; failAt <= probe.writes; failAt++ {
		require.Error(t, CleanupExecution(&failAfterWriter{failAt: failAt}, Text, result))
	}
}
