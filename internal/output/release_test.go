package output

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/internal/release"
	"github.com/raithlin/gha/pkg/model"
)

func TestReleasePublicationRendersAllDecisionSignals(t *testing.T) {
	plan := &release.Plan{SchemaVersion: "v1", Repository: model.RepositoryRef{Owner: "acme", Name: "tool"}, Version: "1.2.3", Tag: "v1.2.3", Commit: "abc123", Branch: "main", OriginRef: "refs/tags/v1.2.3", Ready: false, Blockers: []string{"CI check failed"}, Checks: []*model.CheckRun{{Name: "test", Status: "completed", Conclusion: "failure"}}, Workflow: release.WorkflowState{State: "available", Path: ".github/workflows/release.yml"}, ReleaseNotes: release.NotesState{State: "available", Path: "docs/releases/v1.2.3.md"}, LocalTag: "planned", OriginTag: "planned", ReleaseWorkflow: release.WorkflowObservation{State: "unavailable", Message: "pending visibility"}}
	for _, format := range []Format{Text, JSON, YAML} {
		var rendered bytes.Buffer
		require.NoError(t, ReleasePublication(&rendered, format, plan))
		assert.Contains(t, rendered.String(), "v1.2.3")
		assert.Contains(t, rendered.String(), "CI check failed")
		assert.Contains(t, rendered.String(), "pending visibility")
		assert.Contains(t, rendered.String(), "docs/releases/v1.2.3.md")
	}
}

type failedReleaseWriter struct{}

func (failedReleaseWriter) Write([]byte) (int, error) { return 0, errors.New("closed") }

func TestReleasePublicationPropagatesOutputError(t *testing.T) {
	err := ReleasePublication(failedReleaseWriter{}, Text, &release.Plan{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "closed")
}

type nthFailedReleaseWriter struct{ writes, failAt int }

func (w *nthFailedReleaseWriter) Write(data []byte) (int, error) {
	w.writes++
	if w.writes == w.failAt {
		return 0, errors.New("closed")
	}
	return len(data), nil
}

func TestReleasePublicationPropagatesErrorsFromLaterSections(t *testing.T) {
	plan := &release.Plan{Checks: []*model.CheckRun{{Name: "test", Status: "completed"}}, Blockers: []string{"blocked"}, ReleaseWorkflow: release.WorkflowObservation{Message: "pending"}}
	for failAt := 2; failAt <= 5; failAt++ {
		writer := &nthFailedReleaseWriter{failAt: failAt}
		err := ReleasePublication(writer, Text, plan)
		require.ErrorContains(t, err, "closed")
	}
}
