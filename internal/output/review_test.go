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

func TestBranchInspectionTextShowsExplicitUnavailableSafetySignals(t *testing.T) {
	var writer bytes.Buffer
	inspection := &model.BranchInspection{
		Name: "feature/api",
		Safety: model.BranchSafety{
			Provider:      "github",
			CheckedAt:     "2026-09-13T10:00:00Z",
			Requests:      model.ProviderSignal{State: "unavailable", Message: "token rejected"},
			Protection:    model.ProviderSignal{State: "unavailable", Message: "token rejected"},
			Permissions:   model.ProviderSignal{State: "unavailable", Message: "token rejected"},
			DefaultBranch: model.ProviderSignal{State: "unavailable", Message: "token rejected"},
			Merge:         model.ProviderSignal{State: "unavailable", Message: "token rejected"},
		},
	}

	require.NoError(t, BranchInspection(&writer, Text, inspection))

	assert.Contains(t, writer.String(), "Branch: feature/api")
	assert.Contains(t, writer.String(), "Provider: GitHub")
	assert.Contains(t, writer.String(), "Safety checked: 2026-09-13T10:00:00Z")
	assert.Contains(t, writer.String(), "Provider status: unavailable; use --format json for details")
	assert.Contains(t, writer.String(), "Open pull requests: unavailable")
	assert.Contains(t, writer.String(), "Mergeable: unavailable")
	assert.NotContains(t, writer.String(), "token rejected")
}

func TestBranchInspectionTextUsesClearDefaultBranchContext(t *testing.T) {
	var writer bytes.Buffer
	no := false
	inspection := &model.BranchInspection{
		Name:       "feature/api",
		Repository: &model.RepositoryRef{Owner: "Raithlin", Name: "gha"},
		Safety: model.BranchSafety{
			Provider:          "github",
			CheckedAt:         "2026-09-13T10:00:00Z",
			Requests:          model.ProviderSignal{State: "available"},
			Protection:        model.ProviderSignal{State: "available"},
			Permissions:       model.ProviderSignal{State: "available"},
			DefaultBranch:     model.ProviderSignal{State: "available"},
			DefaultBranchName: "main",
			IsDefault:         &no,
			Merge:             model.ProviderSignal{State: "not_applicable"},
		},
	}

	require.NoError(t, BranchInspection(&writer, Text, inspection))

	assert.Contains(t, writer.String(), "Repository: Raithlin/gha")
	assert.Contains(t, writer.String(), "Provider status: available")
	assert.Contains(t, writer.String(), "Default branch: main")
	assert.Contains(t, writer.String(), "Is default branch: no")
	assert.Contains(t, writer.String(), "Mergeable: not applicable")
}

func TestReleaseNotesTextRendersChangesAndContributors(t *testing.T) {
	var writer bytes.Buffer
	notes := &model.ReleaseNotes{
		Repository: model.RepositoryRef{Owner: "Raithlin", Name: "gha"},
		Since:      "2026-09-01T00:00:00Z",
		PullRequests: []*model.PullRequest{
			{Number: 42, Title: "Improve release notes", User: model.User{Login: "alice"}},
		},
		Contributors: []model.User{{Login: "alice"}},
	}

	require.NoError(t, ReleaseNotes(&writer, Text, notes))

	assert.Contains(t, writer.String(), "Repository: Raithlin/gha")
	assert.Contains(t, writer.String(), "- #42 Improve release notes (alice)")
	assert.Contains(t, writer.String(), "Contributors:")
}

func TestReleaseNotesJSONUsesVersionedSchema(t *testing.T) {
	var writer bytes.Buffer
	notes := &model.ReleaseNotes{SchemaVersion: model.ReleaseNotesSchemaVersion}

	require.NoError(t, ReleaseNotes(&writer, JSON, notes))

	var value map[string]any
	require.NoError(t, json.Unmarshal(writer.Bytes(), &value))
	assert.Equal(t, model.ReleaseNotesSchemaVersion, value["schema_version"])
	assert.Contains(t, value, "pull_requests")
	assert.Contains(t, value, "contributors")
	assert.Contains(t, value, "limit")
	assert.Contains(t, value, "truncated")
}

func TestPullRequestListJSONUsesVersionedSchemaAndTruncation(t *testing.T) {
	var writer bytes.Buffer
	list := &model.PullRequestList{
		SchemaVersion: model.PullRequestListSchemaVersion,
		Limit:         1,
		Truncated:     true,
		PullRequests:  []*model.PullRequest{{Number: 123}},
	}

	require.NoError(t, PullRequestList(&writer, JSON, list, "Pull Requests"))
	var value map[string]any
	require.NoError(t, json.Unmarshal(writer.Bytes(), &value))
	assert.Equal(t, model.PullRequestListSchemaVersion, value["schema_version"])
	assert.Equal(t, true, value["truncated"])
	assert.Contains(t, value, "pull_requests")
}

func TestCapabilitiesJSONUsesVersionedSchema(t *testing.T) {
	var writer bytes.Buffer
	capabilities := &model.Capabilities{SchemaVersion: model.CapabilitiesSchemaVersion, Commands: []model.Capability{}}

	require.NoError(t, Capabilities(&writer, JSON, capabilities))
	var value map[string]any
	require.NoError(t, json.Unmarshal(writer.Bytes(), &value))
	assert.Equal(t, model.CapabilitiesSchemaVersion, value["schema_version"])
	assert.Contains(t, value, "commands")
}

func TestPullRequestPreparationRendersDecisionReadyText(t *testing.T) {
	preparation := &model.PullRequestPreparation{
		SchemaVersion:        model.PullRequestPreparationSchemaVersion,
		Repository:           model.RepositoryRef{Owner: "acme", Name: "project"},
		Title:                "Improve reviews",
		Head:                 "feature",
		Base:                 "main",
		Creation:             "planned",
		Comparison:           model.BranchComparison{State: "ahead", AheadBy: 2},
		Permissions:          model.ProviderSignal{State: "available"},
		ExistingRequests:     model.ProviderSignal{State: "available"},
		ExistingPullRequests: []*model.PullRequest{},
		RecommendedActions:   []model.RecommendedAction{{Action: "create", Reason: "rerun with confirmation"}},
	}
	var writer bytes.Buffer

	require.NoError(t, PullRequestPreparation(&writer, Text, preparation))

	assert.Contains(t, writer.String(), "Pull request preparation")
	assert.Contains(t, writer.String(), "feature → main")
	assert.Contains(t, writer.String(), "2 ahead")
	assert.Contains(t, writer.String(), "Recommended next actions")
}

func TestBranchInventoryTextRendersSourcesAndDivergence(t *testing.T) {
	var writer bytes.Buffer
	ahead, behind := 2, 1
	inventory := &model.BranchInventory{
		SchemaVersion:  model.BranchInventorySchemaVersion,
		Origin:         "git@example.com:acme/project.git",
		OriginState:    "cached",
		LocalTruncated: true,
		Local: []*model.Branch{{
			Name: "feature", SHA: "0123456789abcdef", Current: true, Upstream: "origin/feature", DivergenceState: "available", Ahead: &ahead, Behind: &behind,
		}},
		OriginBranches: []*model.Branch{{Name: "main", SHA: "abcdef0123456789"}},
	}

	require.NoError(t, BranchInventory(&writer, Text, inventory))
	assert.Contains(t, writer.String(), "Origin: git@example.com:acme/project.git")
	assert.Contains(t, writer.String(), "* feature")
	assert.Contains(t, writer.String(), "origin/feature (2 ahead, 1 behind)")
	assert.Contains(t, writer.String(), "additional branches omitted; increase --limit")
	assert.Contains(t, writer.String(), "Origin branches")
}

func TestBranchMutationTextReportsCheckoutTransition(t *testing.T) {
	var writer bytes.Buffer
	mutation := &model.BranchMutation{Operation: "delete", Name: "feature", CheckedOut: "main", Local: "completed", Origin: "not_requested"}

	require.NoError(t, BranchMutation(&writer, Text, mutation))

	assert.Contains(t, writer.String(), "Checked out: main")
}

func TestBranchPublicationTextRendersTheGuardedPlan(t *testing.T) {
	var writer bytes.Buffer
	canPush := true
	publication := &model.BranchPublication{
		Name:        "feature/api",
		Target:      "origin/feature/api",
		Origin:      "git@example.com:acme/project.git",
		OriginState: "cached",
		Local:       &model.Branch{SHA: "0123456789abcdef", DivergenceState: "not_tracked"},
		Permissions: model.ProviderSignal{State: "available"},
		CanPush:     &canPush,
		DryRun:      true,
		Publication: "planned",
	}

	require.NoError(t, BranchPublication(&writer, Text, publication))

	assert.Contains(t, writer.String(), "Branch publication: feature/api → origin/feature/api")
	assert.Contains(t, writer.String(), "Origin: git@example.com:acme/project.git (cached)")
	assert.Contains(t, writer.String(), "Upstream: none")
	assert.Contains(t, writer.String(), "Divergence: not tracked")
	assert.Contains(t, writer.String(), "Can push: true")
	assert.Contains(t, writer.String(), "Dry run: no changes were made.")
}

func TestBranchInventoryJSONUsesVersionedSchema(t *testing.T) {
	var writer bytes.Buffer

	require.NoError(t, BranchInventory(&writer, JSON, &model.BranchInventory{SchemaVersion: model.BranchInventorySchemaVersion, OriginState: "absent"}))

	var value map[string]any
	require.NoError(t, json.Unmarshal(writer.Bytes(), &value))
	assert.Equal(t, model.BranchInventorySchemaVersion, value["schema_version"])
	assert.Contains(t, value, "local")
	assert.Contains(t, value, "origin_branches")
	assert.Equal(t, "absent", value["origin_state"])
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
