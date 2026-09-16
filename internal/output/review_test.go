package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/pkg/model"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("writer failed") }

type failAfterWriter struct {
	failAt int
	writes int
}

func (writer *failAfterWriter) Write(value []byte) (int, error) {
	writer.writes++
	if writer.writes == writer.failAt {
		return 0, errors.New("writer failed")
	}
	return len(value), nil
}

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

func TestBranchCleanupTextExplainsCandidatesAndExclusions(t *testing.T) {
	var writer bytes.Buffer
	cleanup := &model.BranchCleanup{
		Rule:       "tip_reachable_from_base",
		Base:       "main",
		Candidates: []*model.BranchCleanupCandidate{{Name: "feature/merged", Reason: "tip_reachable_from_base"}},
		Excluded:   []*model.BranchCleanupCandidate{{Name: "feature/active", Reason: "not_reachable_from_base"}},
	}

	require.NoError(t, BranchCleanup(&writer, Text, cleanup))

	assert.Contains(t, writer.String(), "Cleanup candidates")
	assert.Contains(t, writer.String(), "Base: main")
	assert.Contains(t, writer.String(), "feature/merged (tip reachable from base)")
	assert.Contains(t, writer.String(), "feature/active (not reachable from base)")
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

func TestReleaseListTextRendersPublishedReleasesAndTruncation(t *testing.T) {
	var writer bytes.Buffer
	releases := &model.ReleaseList{
		Repository: model.RepositoryRef{Owner: "Raithlin", Name: "gha"},
		Truncated:  true,
		Releases: []*model.Release{{
			TagName:     "v1.2.0",
			Name:        "Release 1.2.0",
			Prerelease:  true,
			PublishedAt: "2026-09-03T00:00:00Z",
		}},
	}

	require.NoError(t, ReleaseList(&writer, Text, releases))

	assert.Contains(t, writer.String(), "Published releases")
	assert.Contains(t, writer.String(), "v1.2.0 — Release 1.2.0")
	assert.Contains(t, writer.String(), "pre-release")
	assert.Contains(t, writer.String(), "Additional releases omitted; increase --limit.")
}

func TestReleaseListJSONUsesVersionedSchema(t *testing.T) {
	var writer bytes.Buffer
	releases := &model.ReleaseList{SchemaVersion: model.ReleaseListSchemaVersion}

	require.NoError(t, ReleaseList(&writer, JSON, releases))

	var value map[string]any
	require.NoError(t, json.Unmarshal(writer.Bytes(), &value))
	assert.Equal(t, model.ReleaseListSchemaVersion, value["schema_version"])
	assert.Contains(t, value, "releases")
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
		OriginRefresh:  model.OriginRefresh{State: "not_requested"},
		LocalTruncated: true,
		Local: []*model.Branch{{
			Name: "feature", SHA: "0123456789abcdef", Current: true, Upstream: "origin/feature", DivergenceState: "available", Ahead: &ahead, Behind: &behind,
		}},
		OriginBranches: []*model.Branch{{Name: "main", SHA: "abcdef0123456789"}},
	}

	require.NoError(t, BranchInventory(&writer, Text, inventory))
	assert.Contains(t, writer.String(), "Origin: git@example.com:acme/project.git")
	assert.Contains(t, writer.String(), "Origin refresh: not_requested")
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
	assert.Equal(t, "", value["origin_refresh"].(map[string]any)["state"])
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

func TestParseFormatAndStructuredRenderers(t *testing.T) {
	for _, test := range []struct {
		value string
		want  Format
	}{
		{"", Text}, {"text", Text}, {"human", Text}, {"json", JSON}, {"yaml", YAML},
	} {
		got, err := ParseFormat(test.value)
		require.NoError(t, err)
		assert.Equal(t, test.want, got)
	}
	_, err := ParseFormat("xml")
	assert.ErrorContains(t, err, "unsupported format")

	var writer bytes.Buffer
	require.NoError(t, CommandError(&writer, YAML, &model.CommandError{Code: "invalid", Message: "bad input"}))
	assert.Contains(t, writer.String(), "code: invalid")
	assert.Error(t, structured(&writer, Text, struct{}{}))
}

func TestTextRenderersCoverDecisionReadyDetails(t *testing.T) {
	yes, no := true, false
	branch := &model.Branch{Name: "feature", SHA: "0123456789abcdef", Current: true, Upstream: "origin/feature", DivergenceState: "available", Ahead: &yesInt, Behind: &noInt}
	var writer bytes.Buffer

	require.NoError(t, RepositoryAnalysis(&writer, Text, &model.RepositoryAnalysis{
		Path: "/work/project", AnalyzedAt: "2026-09-15T10:00:00Z", Head: model.AnalysisHead{State: "available", Branch: "main", SHA: "abcdef0123456789", Commits: 3},
		Worktree: model.WorktreeSummary{State: "open", Staged: 1, Unstaged: 2, Untracked: 3, Conflicted: 1, ChangesTruncated: true, Changes: []model.WorktreeChange{{IndexStatus: "M", WorktreeStatus: " ", OriginalPath: "old.txt", Path: "new.txt"}}},
		Storage:  model.RepositoryStorage{LooseObjects: 1, LooseKiB: 2, PackedKiB: 3}, RecentCommitsTruncated: true, RecentCommits: []model.LocalCommit{{SHA: "123456789abcdef", Subject: "Improve output", AuthoredAt: "today"}}, LargestFilesSignal: model.AnalysisSignal{State: "available"}, LargestFilesTruncated: true, LargestFiles: []model.LargestFile{{Bytes: 42, Path: "large.bin"}},
	}))
	assert.Contains(t, writer.String(), "old.txt → new.txt")
	assert.Contains(t, writer.String(), "large.bin")

	writer.Reset()
	require.NoError(t, BranchInventory(&writer, Text, &model.BranchInventory{Origin: "git@example/project", OriginState: "cached", OriginRefresh: model.OriginRefresh{State: "planned"}, Local: []*model.Branch{branch}, LocalTruncated: true, OriginBranches: []*model.Branch{}, OriginTruncated: true}))
	assert.Contains(t, writer.String(), "1 ahead, 0 behind")
	assert.Contains(t, writer.String(), "additional branches omitted")

	writer.Reset()
	require.NoError(t, BranchInspection(&writer, Text, &model.BranchInspection{Name: "feature", Repository: &model.RepositoryRef{Owner: "acme", Name: "project"}, Origin: "origin", OriginState: "cached", Local: branch, Safety: model.BranchSafety{Provider: "gitlab", CheckedAt: "now", Requests: model.ProviderSignal{State: "available"}, OpenPullRequests: []*model.PullRequest{{Number: 3, Title: "Open"}}, Protection: model.ProviderSignal{State: "available"}, Protected: &no, Permissions: model.ProviderSignal{State: "available"}, CanPush: &yes, DefaultBranch: model.ProviderSignal{State: "available"}, DefaultBranchName: "main", IsDefault: &no, Merge: model.ProviderSignal{State: "available"}, Mergeable: &yes}}))
	assert.Contains(t, writer.String(), "Provider: gitlab")
	assert.Contains(t, writer.String(), "#3 Open")

	writer.Reset()
	require.NoError(t, BranchMutation(&writer, Text, &model.BranchMutation{Operation: "rename", Name: "old", NewName: "new", From: "main", CheckedOut: "main", DryRun: true, Local: "completed", Origin: "planned"}))
	assert.Contains(t, writer.String(), "Dry run")

	writer.Reset()
	require.NoError(t, BranchPublication(&writer, Text, &model.BranchPublication{Repository: &model.RepositoryRef{Owner: "acme", Name: "project"}, Name: "feature", Target: "origin/feature", Origin: "origin", OriginState: "cached", Local: branch, Permissions: model.ProviderSignal{State: "available"}, CanPush: &yes, DryRun: true, Publication: "planned"}))
	assert.Contains(t, writer.String(), "Divergence: 1 ahead, 0 behind")

	writer.Reset()
	require.NoError(t, PullRequestPreparation(&writer, Text, &model.PullRequestPreparation{Repository: model.RepositoryRef{Owner: "acme", Name: "project"}, Title: "Title", Head: "feature", Base: "main", DryRun: true, Creation: "planned", Comparison: model.BranchComparison{State: "ahead", Message: "two commits", AheadBy: 2}, ExistingRequests: model.ProviderSignal{State: "available"}, Permissions: model.ProviderSignal{State: "available"}, CanPush: &yes, RiskSignals: []model.RiskSignal{{Severity: "high", Kind: "ci", Detail: "failed"}}, RecommendedActions: []model.RecommendedAction{{Action: "fix", Reason: "CI"}}, CreatedPullRequest: &model.PullRequest{Number: 4, Title: "Title"}}))
	assert.Contains(t, writer.String(), "Created: #4 Title")
}

func TestTextRenderersCoverEmptyAndUnavailableStates(t *testing.T) {
	var writer bytes.Buffer
	require.NoError(t, RepositoryAnalysis(&writer, Text, &model.RepositoryAnalysis{Head: model.AnalysisHead{State: "unborn"}, Worktree: model.WorktreeSummary{State: "clean"}, LargestFilesSignal: model.AnalysisSignal{State: "unavailable"}}))
	assert.Contains(t, writer.String(), "HEAD: unborn")
	assert.Contains(t, writer.String(), "Largest tracked files")

	writer.Reset()
	require.NoError(t, ReleaseNotes(&writer, Text, &model.ReleaseNotes{Repository: model.RepositoryRef{Owner: "acme", Name: "project"}}))
	assert.Contains(t, writer.String(), "No pull requests")
	writer.Reset()
	require.NoError(t, ReleaseList(&writer, Text, &model.ReleaseList{Repository: model.RepositoryRef{Owner: "acme", Name: "project"}}))
	assert.Contains(t, writer.String(), "No published releases")

	writer.Reset()
	require.NoError(t, PullRequestList(&writer, Text, &model.PullRequestList{Truncated: true, PullRequests: []*model.PullRequest{{Number: 1, Title: "A very long title that needs a useful short terminal representation", State: "closed", User: model.User{Login: "alice"}}}}, "Pull requests"))
	assert.Contains(t, writer.String(), "additional pull requests")
	assert.Contains(t, writer.String(), "...")

	writer.Reset()
	require.NoError(t, Capabilities(&writer, Text, &model.Capabilities{Commands: []model.Capability{{Command: "review", Status: "available", Notes: "safe"}}}))
	assert.Contains(t, writer.String(), "review [available]: safe")
	writer.Reset()
	require.NoError(t, VersionInfo(&writer, Text, &model.VersionInfo{Version: "v1", Commit: "0123456789abcdef", Date: "today"}))
	assert.Contains(t, writer.String(), "0123456789ab")
}

func TestRendererHelpersCoverAllStates(t *testing.T) {
	assert.Equal(t, "not applicable", signalText(model.ProviderSignal{State: "not_applicable"}))
	assert.Equal(t, "unavailable", signalText(model.ProviderSignal{}))
	assert.Equal(t, "custom", signalText(model.ProviderSignal{State: "custom"}))
	assert.Equal(t, "GitHub", providerName("GITHUB"))
	assert.Equal(t, "unknown", booleanText(nil))
	assert.Equal(t, "unknown", yesNoText(nil))
	assert.Equal(t, "unknown", mergeableText(nil))
	yes, no := true, false
	assert.Equal(t, "true", booleanText(&yes))
	assert.Equal(t, "no", yesNoText(&no))
	assert.Equal(t, "false", mergeableText(&no))
	availableSafety := model.BranchSafety{Requests: model.ProviderSignal{State: "available"}, Protection: model.ProviderSignal{State: "available"}, Permissions: model.ProviderSignal{State: "available"}, DefaultBranch: model.ProviderSignal{State: "available"}, Merge: model.ProviderSignal{State: "available"}}
	assert.Equal(t, "available", providerStatus(availableSafety))
	availableSafety.Requests.State = "unavailable"
	assert.Contains(t, providerStatus(availableSafety), "partial")
	assert.Contains(t, providerStatus(model.BranchSafety{Requests: model.ProviderSignal{State: "unavailable"}, Protection: model.ProviderSignal{State: "unavailable"}, Permissions: model.ProviderSignal{State: "unavailable"}, DefaultBranch: model.ProviderSignal{State: "unavailable"}, Merge: model.ProviderSignal{State: "unavailable"}}), "unavailable")
	assert.Equal(t, "plain\ntext", sanitizeTerminal("plain\x1b[31m\ntext\x1b[0m"))
}

func TestReviewAndPullRequestTextCoverOptionalDetails(t *testing.T) {
	mergeable := true
	var writer bytes.Buffer
	pr := &model.PullRequest{Number: 9, Title: "Ready", State: "merged", User: model.User{Login: "alice"}, CreatedAt: "yesterday", UpdatedAt: "today", ClosedAt: "today", MergedAt: "today", Head: model.BranchRef{Ref: "feature", SHA: "0123456789abcdef"}, Base: model.BranchRef{Ref: "main", SHA: "abcdef0123456789"}, Body: "fallback body", Comments: 2, Commits: 3, Additions: 4, Deletions: 5}
	require.NoError(t, PullRequest(&writer, Text, pr))
	assert.Contains(t, writer.String(), "Closed: today")
	assert.Contains(t, writer.String(), "Merged: today")
	assert.Contains(t, writer.String(), "fallback body")

	writer.Reset()
	require.NoError(t, ReviewSummary(&writer, Text, &model.ReviewSummary{PullRequest: pr, Readiness: model.ReviewReadiness{Mergeable: &mergeable, CIStatus: "success", ReviewThreadsState: "resolved", ApprovedBy: []model.User{{Login: "bob"}}, ChangesRequestedBy: []model.User{{Login: "carol"}}, PendingReviewers: []model.User{{Login: "dave"}}}, RiskSignals: []model.RiskSignal{{Severity: "low", Kind: "docs", Detail: "review"}}, RecommendedActions: []model.RecommendedAction{{Action: "merge", Reason: "ready"}}}))
	assert.Contains(t, writer.String(), "Approved by: bob")
	assert.Contains(t, writer.String(), "[low] docs: review")
}

func TestAdditionalTextRendererVariants(t *testing.T) {
	var writer bytes.Buffer
	require.NoError(t, BranchCleanup(&writer, Text, &model.BranchCleanup{Base: "main", Truncated: true}))
	assert.Contains(t, writer.String(), "Additional local branches omitted")
	writer.Reset()
	require.NoError(t, BranchPublication(&writer, Text, &model.BranchPublication{Name: "feature", Target: "origin/feature", OriginState: "absent", Permissions: model.ProviderSignal{State: "unavailable"}, Publication: "blocked"}))
	assert.Contains(t, writer.String(), "Local branch: none")
	writer.Reset()
	require.NoError(t, BranchInventory(&writer, Text, &model.BranchInventory{OriginState: "absent", OriginRefresh: model.OriginRefresh{State: "not_requested"}}))
	assert.Contains(t, writer.String(), "Local branches (0 total)")
}

func TestBranchInspectionIgnoresNullOpenPullRequestEntries(t *testing.T) {
	var writer bytes.Buffer
	inspection := &model.BranchInspection{Name: "feature", Safety: model.BranchSafety{Requests: model.ProviderSignal{State: "available"}, OpenPullRequests: []*model.PullRequest{nil}, Protection: model.ProviderSignal{State: "unavailable"}, Permissions: model.ProviderSignal{State: "unavailable"}, DefaultBranch: model.ProviderSignal{State: "unavailable"}, Merge: model.ProviderSignal{State: "unavailable"}}}
	require.NoError(t, BranchInspection(&writer, Text, inspection))
	assert.Contains(t, writer.String(), "Open pull requests: 0")
}

func TestTextRenderersPropagateWriterFailures(t *testing.T) {
	writer := failingWriter{}
	pr := &model.PullRequest{}
	assert.Error(t, PullRequest(writer, Text, pr))
	assert.Error(t, ReviewSummary(writer, Text, &model.ReviewSummary{PullRequest: pr}))
	assert.Error(t, ReleaseNotes(writer, Text, &model.ReleaseNotes{}))
	assert.Error(t, ReleaseList(writer, Text, &model.ReleaseList{}))
	assert.Error(t, PullRequestPreparation(writer, Text, &model.PullRequestPreparation{}))
	assert.Error(t, RepositoryAnalysis(writer, Text, &model.RepositoryAnalysis{}))
	assert.Error(t, BranchInventory(writer, Text, &model.BranchInventory{}))
	assert.Error(t, BranchCleanup(writer, Text, &model.BranchCleanup{}))
	assert.Error(t, BranchInspection(writer, Text, &model.BranchInspection{}))
	assert.Error(t, BranchMutation(writer, Text, &model.BranchMutation{}))
	assert.Error(t, BranchPublication(writer, Text, &model.BranchPublication{}))
	assert.Error(t, PullRequestList(writer, Text, &model.PullRequestList{}, "Pull requests"))
	assert.Error(t, Capabilities(writer, Text, &model.Capabilities{}))
	assert.Error(t, VersionInfo(writer, Text, &model.VersionInfo{}))
}

func TestTextRenderersPropagateFailuresAtEveryWrite(t *testing.T) {
	yes, no := true, false
	ahead, behind := 2, 1
	branch := &model.Branch{Name: "feature", SHA: "0123456789abcdef", Current: true, Upstream: "origin/feature", DivergenceState: "available", Ahead: &ahead, Behind: &behind}
	pr := &model.PullRequest{Number: 7, Title: "Ready", State: "merged", User: model.User{Login: "alice"}, CreatedAt: "yesterday", UpdatedAt: "today", ClosedAt: "today", MergedAt: "today", Head: model.BranchRef{Ref: "feature", SHA: "0123456789abcdef"}, Base: model.BranchRef{Ref: "main", SHA: "abcdef0123456789"}, BodyText: "Description", Comments: 1, Commits: 2, Additions: 3, Deletions: 4}
	renderers := map[string]func(io.Writer) error{
		"pull request": func(writer io.Writer) error { return PullRequest(writer, Text, pr) },
		"review summary": func(writer io.Writer) error {
			return ReviewSummary(writer, Text, &model.ReviewSummary{PullRequest: pr, Reviews: []*model.Review{{User: model.User{Login: "bob"}, State: "APPROVED", SubmittedAt: "now"}}, Readiness: model.ReviewReadiness{Mergeable: &yes, CIStatus: "success", ReviewThreadsState: "resolved", ApprovedBy: []model.User{{Login: "bob"}}, ChangesRequestedBy: []model.User{{Login: "carol"}}, PendingReviewers: []model.User{{Login: "dave"}}}, RiskSignals: []model.RiskSignal{{Severity: "high", Kind: "risk", Detail: "detail"}}, RecommendedActions: []model.RecommendedAction{{Action: "act", Reason: "reason"}}})
		},
		"release notes": func(writer io.Writer) error {
			return ReleaseNotes(writer, Text, &model.ReleaseNotes{Repository: model.RepositoryRef{Owner: "acme", Name: "project"}, Since: "today", Truncated: true, PullRequests: []*model.PullRequest{pr}, Contributors: []model.User{{Login: "alice"}}})
		},
		"release list": func(writer io.Writer) error {
			return ReleaseList(writer, Text, &model.ReleaseList{Repository: model.RepositoryRef{Owner: "acme", Name: "project"}, Truncated: true, Releases: []*model.Release{{TagName: "v1", Name: "Release", Prerelease: true, PublishedAt: "today"}}})
		},
		"pull request preparation": func(writer io.Writer) error {
			return PullRequestPreparation(writer, Text, &model.PullRequestPreparation{Repository: model.RepositoryRef{Owner: "acme", Name: "project"}, Title: "Title", Head: "feature", Base: "main", DryRun: true, Creation: "planned", Comparison: model.BranchComparison{State: "ahead", Message: "message", AheadBy: 2, BehindBy: 1}, ExistingRequests: model.ProviderSignal{State: "available"}, ExistingPullRequests: []*model.PullRequest{pr}, Permissions: model.ProviderSignal{State: "available"}, CanPush: &yes, RiskSignals: []model.RiskSignal{{Severity: "high", Kind: "risk", Detail: "detail"}}, RecommendedActions: []model.RecommendedAction{{Action: "act", Reason: "reason"}}, CreatedPullRequest: pr})
		},
		"repository analysis": func(writer io.Writer) error {
			return RepositoryAnalysis(writer, Text, &model.RepositoryAnalysis{Path: "/work/project", AnalyzedAt: "now", Head: model.AnalysisHead{State: "available", Branch: "main", SHA: "abcdef0123456789", Commits: 2}, Worktree: model.WorktreeSummary{State: "dirty", Staged: 1, Unstaged: 1, Untracked: 1, Conflicted: 1, ChangesTruncated: true, Changes: []model.WorktreeChange{{IndexStatus: "M", WorktreeStatus: "M", OriginalPath: "old", Path: "new"}}}, Storage: model.RepositoryStorage{LooseObjects: 1, LooseKiB: 1, PackedKiB: 1}, RecentCommitsTruncated: true, RecentCommits: []model.LocalCommit{{SHA: "abcdef0123456789", Subject: "subject", AuthoredAt: "now"}}, LargestFilesSignal: model.AnalysisSignal{State: "available"}, LargestFilesTruncated: true, LargestFiles: []model.LargestFile{{Path: "file", Bytes: 1}}})
		},
		"branch inventory": func(writer io.Writer) error {
			return BranchInventory(writer, Text, &model.BranchInventory{Origin: "origin", OriginState: "cached", OriginRefresh: model.OriginRefresh{State: "completed"}, Local: []*model.Branch{branch}, LocalTruncated: true, OriginBranches: []*model.Branch{branch}, OriginTruncated: true})
		},
		"branch cleanup": func(writer io.Writer) error {
			return BranchCleanup(writer, Text, &model.BranchCleanup{Base: "main", Truncated: true, Candidates: []*model.BranchCleanupCandidate{{Name: "merged", Reason: "tip_reachable_from_base"}}, Excluded: []*model.BranchCleanupCandidate{{Name: "active", Reason: "not_reachable_from_base"}}})
		},
		"branch inspection": func(writer io.Writer) error {
			return BranchInspection(writer, Text, &model.BranchInspection{Name: "feature", Repository: &model.RepositoryRef{Owner: "acme", Name: "project"}, Origin: "origin", OriginState: "cached", Local: branch, OriginBranch: branch, Safety: model.BranchSafety{Provider: "github", CheckedAt: "now", Requests: model.ProviderSignal{State: "available"}, OpenPullRequests: []*model.PullRequest{pr}, Protection: model.ProviderSignal{State: "available"}, Protected: &no, Permissions: model.ProviderSignal{State: "available"}, CanPush: &yes, DefaultBranch: model.ProviderSignal{State: "available"}, DefaultBranchName: "main", IsDefault: &no, Merge: model.ProviderSignal{State: "available"}, Mergeable: &yes}})
		},
		"branch mutation": func(writer io.Writer) error {
			return BranchMutation(writer, Text, &model.BranchMutation{Operation: "rename", Name: "old", NewName: "new", From: "main", CheckedOut: "main", DryRun: true, Local: "completed", Origin: "completed"})
		},
		"branch publication": func(writer io.Writer) error {
			return BranchPublication(writer, Text, &model.BranchPublication{Repository: &model.RepositoryRef{Owner: "acme", Name: "project"}, Name: "feature", Target: "origin/feature", Origin: "origin", OriginState: "cached", Local: branch, OriginBranch: branch, Permissions: model.ProviderSignal{State: "available"}, CanPush: &yes, DryRun: true, Publication: "planned"})
		},
		"pull request list": func(writer io.Writer) error {
			return PullRequestList(writer, Text, &model.PullRequestList{Truncated: true, PullRequests: []*model.PullRequest{pr}}, "Pull requests")
		},
		"capabilities": func(writer io.Writer) error {
			return Capabilities(writer, Text, &model.Capabilities{Commands: []model.Capability{{Command: "review", Status: "available", Notes: "safe"}}})
		},
		"version": func(writer io.Writer) error {
			return VersionInfo(writer, Text, &model.VersionInfo{Version: "v1", Commit: "abcdef0123456789", Date: "now"})
		},
	}

	for name, render := range renderers {
		t.Run(name, func(t *testing.T) {
			for failAt := 1; failAt < 100; failAt++ {
				writer := &failAfterWriter{failAt: failAt}
				if err := render(writer); err == nil {
					return
				}
			}
			t.Fatal("renderer wrote more than 99 times")
		})
	}
}

var (
	yesInt = 1
	noInt  = 0
)
