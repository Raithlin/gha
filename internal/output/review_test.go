package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/pkg/model"
)

func TestReviewSummaryJSONUsesVersionedSchema(t *testing.T) {
	var writer bytes.Buffer
	summary := &model.ReviewSummary{
		SchemaVersion: model.ReviewSummarySchemaVersion,
		PullRequest:   &model.PullRequest{Number: 123},
		Readiness:     model.ReviewReadiness{CIStatus: "unavailable", ReviewThreadsState: "unavailable"},
	}

	require.NoError(t, ReviewSummary(&writer, JSON, summary))

	var value map[string]any
	require.NoError(t, json.Unmarshal(writer.Bytes(), &value))
	assert.Equal(t, model.ReviewSummarySchemaVersion, value["schema_version"])
	assert.Contains(t, value, "pull_request")
	assert.Contains(t, value, "readiness")
	assert.Contains(t, value, "risk_signals")
	assert.Contains(t, value, "recommended_actions")
}

func TestReviewSummaryTextShowsUnavailableSignals(t *testing.T) {
	var writer bytes.Buffer
	summary := &model.ReviewSummary{
		PullRequest: &model.PullRequest{Number: 123, Title: "Improve automation"},
		Readiness:   model.ReviewReadiness{CIStatus: "unavailable", ReviewThreadsState: "unavailable"},
	}

	require.NoError(t, ReviewSummary(&writer, Text, summary))

	assert.Contains(t, writer.String(), "CI: unavailable")
	assert.Contains(t, writer.String(), "Review threads: unavailable")
}
