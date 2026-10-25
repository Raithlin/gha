package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lifecycleOutputFixture() *model.PRWorktree {
	return &model.PRWorktree{SchemaVersion: "v1", Operation: "checkout", Repository: model.RepositoryRef{Owner: "acme", Name: "project"}, PullRequest: &model.PullRequest{Number: 42, Title: "Fix\x1b[31m", State: "open"}, Path: "/repo/pr42", Branch: "feature", Decision: "resume", State: "resumed", Fetch: "not_needed", BranchState: "retained", WorktreeState: "not_needed", DryRun: true, Worktree: &model.Worktree{StatusState: "dirty"}, Comparison: model.BranchComparison{State: "ahead"}, ReviewState: model.ProviderSignal{State: "unavailable", Message: "offline"}, Review: &model.ReviewSummary{Readiness: model.ReviewReadiness{CIStatus: "failure", MergeableState: "blocked", ReviewThreadsState: "unresolved"}}, Push: model.PRPushTarget{State: "available", Repository: model.RepositoryRef{Owner: "contributor", Name: "fork"}, URL: "https://github.com/contributor/fork.git", Refspec: "HEAD:refs/heads/feature"}, Reasons: []string{"local work"}, RecommendedActions: []model.RecommendedAction{{Action: "fix_ci", Reason: "checks failed"}}}
}

func TestPRWorktreeOutputSharesFacts(t *testing.T) {
	value := lifecycleOutputFixture()
	for _, format := range []Format{Text, JSON, YAML} {
		var out bytes.Buffer
		require.NoError(t, PRWorktree(&out, format, value))
		for _, expected := range []string{"feature", "/repo/pr42", "resumed", "failure", "offline", "HEAD:refs/heads/feature", "local work", "fix_ci"} {
			assert.Contains(t, out.String(), expected)
		}
		if format == JSON {
			var decoded model.PRWorktree
			require.NoError(t, json.Unmarshal(out.Bytes(), &decoded))
			assert.Equal(t, value, &decoded)
		}
		if format == Text {
			assert.NotContains(t, out.String(), "\x1b")
			assert.Contains(t, out.String(), "Dry run")
		}
	}
	value.DryRun = false
	value.Review = nil
	value.Worktree = nil
	value.ReviewState.Message = ""
	value.Push.State = "unavailable"
	require.NoError(t, PRWorktree(io.Discard, Text, value))
	require.ErrorContains(t, PRWorktree(io.Discard, Text, nil), "nil")
}

type lifecycleFailWriter struct{ remaining int }

func (w *lifecycleFailWriter) Write(data []byte) (int, error) {
	if w.remaining == 0 {
		return 0, fmt.Errorf("output failed")
	}
	w.remaining--
	return len(data), nil
}
func TestPRWorktreeOutputPreservesWriteErrors(t *testing.T) {
	counter := &lifecycleFailWriter{remaining: 1000}
	require.NoError(t, PRWorktree(counter, Text, lifecycleOutputFixture()))
	for index := 0; index < 1000-counter.remaining; index++ {
		err := PRWorktree(&lifecycleFailWriter{remaining: index}, Text, lifecycleOutputFixture())
		require.ErrorContains(t, err, "output failed")
	}
	require.Error(t, PRWorktree(&lifecycleFailWriter{}, JSON, lifecycleOutputFixture()))
}

func TestPRWorktreeTextExplainsLocalCommitsAndPendingReview(t *testing.T) {
	value := lifecycleOutputFixture()
	value.Comparison.AheadBy = 2
	value.Comparison.BehindBy = 3
	value.Review.Readiness.PendingReviewers = []model.User{{Login: "reviewer"}}
	var out bytes.Buffer
	require.NoError(t, PRWorktree(&out, Text, value))
	assert.Contains(t, out.String(), "Commits absent from PR: 2")
	assert.Contains(t, out.String(), "Commits behind PR: 3")
	assert.Contains(t, out.String(), "Pending reviewers: reviewer")
	value.Comparison.State = "unavailable"
	out.Reset()
	require.NoError(t, PRWorktree(&out, Text, value))
	assert.NotContains(t, out.String(), "Commits absent from PR:")
}
