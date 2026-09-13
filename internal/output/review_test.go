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

func TestPullRequestTextPrefersPlainTextDescription(t *testing.T) {
	var writer bytes.Buffer
	pr := &model.PullRequest{
		Number:   123,
		Body:     "<p>Plain terminal text</p>",
		BodyText: "Plain terminal text",
	}

	require.NoError(t, PullRequest(&writer, Text, pr))

	assert.Contains(t, writer.String(), "Description:\nPlain terminal text")
	assert.NotContains(t, writer.String(), "<p>")
}

func TestPullRequestTextShowsBranchCommitIDs(t *testing.T) {
	var writer bytes.Buffer
	pr := &model.PullRequest{
		Number: 123,
		Head:   model.BranchRef{Ref: "feature", SHA: "0123456789abcdef"},
		Base:   model.BranchRef{Ref: "main", SHA: "abcdef0123456789"},
	}

	require.NoError(t, PullRequest(&writer, Text, pr))

	assert.Contains(t, writer.String(), "Branches: feature 0123456789ab  →  main abcdef012345")
}

func TestTextOutputSanitizesTerminalControlSequences(t *testing.T) {
	rawTitle := "unsafe\x1b]8;;https://example.invalid\x07title\x1b]8;;\x07"
	pr := &model.PullRequest{
		Number:   123,
		Title:    rawTitle,
		BodyText: "description\x1b[31mred\x1b[0m",
		User:     model.User{Login: "user\x1b[2J"},
		Head:     model.BranchRef{Ref: "branch\x1b[H", SHA: "abc\x1b[2J"},
		Base:     model.BranchRef{Ref: "main", SHA: "def"},
	}
	var writer bytes.Buffer

	require.NoError(t, PullRequest(&writer, Text, pr))
	assert.NotContains(t, writer.String(), "\x1b")
	assert.NotContains(t, writer.String(), "\x07")
	assert.Contains(t, writer.String(), "unsafetitle")
	assert.Contains(t, writer.String(), "descriptionred")

	writer.Reset()
	require.NoError(t, PullRequest(&writer, JSON, pr))
	var decoded model.PullRequest
	require.NoError(t, json.Unmarshal(writer.Bytes(), &decoded))
	assert.Equal(t, rawTitle, decoded.Title)
	assert.Equal(t, "description\x1b[31mred\x1b[0m", decoded.BodyText)
}

func TestTruncatePreservesUnicodeCodePoints(t *testing.T) {
	assert.Equal(t, "猫...", truncate("猫猫猫猫猫", 4))
	assert.Equal(t, "猫猫", truncate("猫猫猫", 2))
}
