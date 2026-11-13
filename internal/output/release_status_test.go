package output

import (
	"bytes"
	"testing"

	"github.com/raithlin/gha/internal/release"
	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestReleaseStatusRendersTerminalAndJSON(t *testing.T) {
	status := &release.Status{SchemaVersion: "v1", Repository: model.RepositoryRef{Owner: "acme", Name: "tool"}, Tag: "v1.2.3", TagState: "available", Commit: "abc123", State: "published",
		Workflow:      release.StatusWorkflow{State: "succeeded", Path: ".github/workflows/release.yml", Candidates: []string{".github/workflows/other.yml"}, RunID: 42, Attempt: 2, Status: "completed", Conclusion: "success", URL: "https://example.test/run", Message: "workflow note"},
		HostedRelease: release.HostedRelease{State: "available", URL: "https://example.test/release", Message: "release note"}, Wait: release.WaitResult{State: "completed", Observations: 3}}
	for _, format := range []Format{Text, JSON, YAML} {
		var rendered bytes.Buffer
		require.NoError(t, ReleaseStatus(&rendered, format, status))
		require.Contains(t, rendered.String(), "published")
		require.Contains(t, rendered.String(), "abc123")
		require.Contains(t, rendered.String(), "42")
		require.Contains(t, rendered.String(), ".github/workflows/other.yml")
	}
}

func TestReleaseStatusPropagatesWriterFailure(t *testing.T) {
	require.Error(t, ReleaseStatus(failedReleaseWriter{}, Text, &release.Status{}))
	require.Error(t, ReleaseStatus(failedReleaseWriter{}, JSON, &release.Status{}))
}
