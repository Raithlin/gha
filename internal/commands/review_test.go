package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/internal/git"
	gh "github.com/raithlin/gha/internal/github"
	"github.com/raithlin/gha/internal/review"
	"github.com/raithlin/gha/pkg/model"
)

func TestReviewCommandRequiresPullRequestNumber(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"review"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestReviewCommandRejectsListingModes(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"review", "123", "--mine"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown flag")
}

func TestRepositoryCommandsExposePathSelection(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	for _, commandName := range [][]string{{"prs"}, {"review"}, {"releases"}, {"release", "create-notes"}} {
		t.Run(strings.Join(commandName, " "), func(t *testing.T) {
			var output bytes.Buffer
			root.SetOut(&output)
			root.SetArgs(append(commandName, "--help"))

			require.NoError(t, root.Execute())
			assert.Contains(t, output.String(), "--path")
		})
	}
}

func TestReleasesListsBoundedPublishedReleases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/Raithlin/gha/releases", r.URL.Path)
		assert.Equal(t, "3", r.URL.Query().Get("per_page"))
		assert.Equal(t, "1", r.URL.Query().Get("page"))
		_, _ = io.WriteString(w, `[
			{"id": 3, "tag_name": "v1.2.0", "name": "1.2.0", "published_at": "2026-09-03T00:00:00Z"},
			{"id": 2, "tag_name": "v1.1.0", "name": "1.1.0", "published_at": "2026-09-02T00:00:00Z"},
			{"id": 1, "tag_name": "v1.0.0", "name": "1.0.0", "published_at": "2026-09-01T00:00:00Z"}
		]`)
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client := &gh.GitHubClient{HTTPClient: server.Client(), BaseURL: baseURL}
	command := newReleasesCmd(review.NewService(client), git.NewRepositoryResolver(""))
	command.SetArgs([]string{"--repo", "Raithlin/gha", "--limit", "2", "--format", "json"})
	command.SetContext(context.Background())
	var rendered bytes.Buffer
	command.SetOut(&rendered)

	require.NoError(t, command.Execute())
	var releases model.ReleaseList
	require.NoError(t, json.Unmarshal(rendered.Bytes(), &releases))
	assert.Equal(t, model.ReleaseListSchemaVersion, releases.SchemaVersion)
	assert.Equal(t, model.RepositoryRef{Owner: "Raithlin", Name: "gha"}, releases.Repository)
	assert.Equal(t, 2, releases.Limit)
	assert.True(t, releases.Truncated)
	require.Len(t, releases.Releases, 2)
	assert.Equal(t, "v1.2.0", releases.Releases[0].TagName)
}

func TestReleaseCreateNotesRequiresReleaseWindow(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"release", "create-notes"})

	err := root.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "required flag(s) \"since\" not set")
}

func TestReleaseCreateNotesRendersStructuredMissingWindowError(t *testing.T) {
	command := newReleaseCreateNotesCmd(nil, nil)
	var diagnostics bytes.Buffer
	command.SetErr(&diagnostics)
	command.SetArgs([]string{"--format", "json"})

	err := command.Execute()

	require.Error(t, err)
	assert.True(t, IsReportedError(err))
	var commandError model.CommandError
	require.NoError(t, json.Unmarshal(diagnostics.Bytes(), &commandError))
	assert.Equal(t, "invalid_argument", commandError.Code)
}

func TestReviewCommandRendersStructuredArgumentErrors(t *testing.T) {
	command := newReviewCmd(nil, nil)
	var diagnostics bytes.Buffer
	command.SetErr(&diagnostics)
	command.SetArgs([]string{"--format", "json"})

	err := command.Execute()

	require.Error(t, err)
	assert.True(t, IsReportedError(err))
	var commandError model.CommandError
	require.NoError(t, json.Unmarshal(diagnostics.Bytes(), &commandError))
	assert.Equal(t, "invalid_argument", commandError.Code)
}

func TestReleaseCreateNotesGeneratesNotes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/Raithlin/gha/pulls", r.URL.Path)
		assert.Equal(t, "closed", r.URL.Query().Get("state"))
		assert.Equal(t, "2026-09-01T00:00:00Z", r.URL.Query().Get("since"))
		assert.Equal(t, "updated", r.URL.Query().Get("sort"))
		assert.Equal(t, "asc", r.URL.Query().Get("direction"))
		_, _ = io.WriteString(w, `[{"number":42,"title":"Ship it","merged_at":"2026-09-02T00:00:00Z","user":{"login":"alice"}}]`)
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client := &gh.GitHubClient{HTTPClient: server.Client(), BaseURL: baseURL}
	command := newReleaseCreateNotesCmd(review.NewService(client), git.NewRepositoryResolver(""))
	command.SetArgs([]string{"--repo", "Raithlin/gha", "--since", "2026-09-01T00:00:00Z"})
	command.SetContext(context.Background())
	var output bytes.Buffer
	command.SetOut(&output)

	require.NoError(t, command.Execute())
	assert.Contains(t, output.String(), "#42 Ship it (alice)")
}

func TestReleaseCreateNotesTreatsDateOnlySinceAsLocalMidnight(t *testing.T) {
	originalLocal := time.Local
	time.Local = time.FixedZone("UTC+2", 2*60*60)
	t.Cleanup(func() { time.Local = originalLocal })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "2025-08-31T22:00:00Z", r.URL.Query().Get("since"))
		_, _ = io.WriteString(w, `[]`)
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client := &gh.GitHubClient{HTTPClient: server.Client(), BaseURL: baseURL}
	command := newReleaseCreateNotesCmd(review.NewService(client), git.NewRepositoryResolver(""))
	command.SetArgs([]string{"--repo", "Raithlin/gha", "--since", "2025-09-01"})
	command.SetContext(context.Background())

	require.NoError(t, command.Execute())
}

func TestReleaseRejectsRemovedNotesSpelling(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"release", "--since", "2026-09-01T00:00:00Z"})

	err := root.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown flag: --since")
}

func TestParseReleaseSincePreservesExplicitTimezone(t *testing.T) {
	timestamp, err := parseReleaseSince("2025-09-01T00:00:00-04:00", time.FixedZone("UTC+2", 2*60*60))

	require.NoError(t, err)
	assert.Equal(t, "2025-09-01T00:00:00-04:00", timestamp.Format(time.RFC3339))
}

func TestListHelpersValidateAndFilterPullRequests(t *testing.T) {
	assert.NoError(t, validateListOptions("open", "updated", "desc", "2026-09-01T00:00:00Z"))
	for _, values := range [][4]string{{"draft", "", "", ""}, {"open", "random", "", ""}, {"open", "", "sideways", ""}, {"open", "", "", "yesterday"}} {
		assert.Error(t, validateListOptions(values[0], values[1], values[2], values[3]))
	}
	assert.True(t, oneOf("a", "a", "b"))
	assert.False(t, oneOf("c", "a", "b"))
	assert.Equal(t, 2, boolCount(true, false, true))

	pr := &model.PullRequest{User: model.User{Login: "alice"}, RequestedReviewers: []model.User{{Login: "bob"}}}
	assert.True(t, matchesPullRequest(pr, "alice", "bob"))
	assert.False(t, matchesPullRequest(pr, "other", "bob"))
	assert.False(t, matchesPullRequest(pr, "alice", "other"))
	assert.True(t, hasRequestedReviewer(pr, "bob"))
	assert.False(t, hasRequestedReviewer(pr, "carol"))

	prs := []*model.PullRequest{{Number: 1}, {Number: 2}}
	assert.Len(t, limitPullRequests(prs, 1), 1)
	list := pullRequestList(model.RepositoryRef{Owner: "acme", Name: "project"}, 1, prs)
	assert.True(t, list.Truncated)
	assert.Equal(t, 1, list.PullRequests[0].Number)
}

func TestParseReleaseSinceAcceptsLocalDateTimesAndRejectsInvalidInput(t *testing.T) {
	location := time.FixedZone("UTC+2", 2*60*60)
	for _, value := range []string{"2025-09-01", "2025-09-01T09:30", "2025-09-01T09:30:15"} {
		timestamp, err := parseReleaseSince(value, location)
		require.NoError(t, err)
		assert.Equal(t, location, timestamp.Location())
	}
	_, err := parseReleaseSince("not-a-date", time.UTC)
	assert.ErrorContains(t, err, "invalid --since")
}

func TestPRListAndReleaseCommandsRenderStructuredValidationErrors(t *testing.T) {
	for _, args := range [][]string{{"--assigned", "--queue"}, {"--mine", "--state", "closed"}, {"--limit", "0"}, {"--state", "draft"}} {
		command := newPRsCmd(nil, nil)
		var diagnostics bytes.Buffer
		command.SetErr(&diagnostics)
		command.SetArgs(append(args, "--format", "json"))
		err := command.Execute()
		require.Error(t, err)
		assert.True(t, IsReportedError(err))
		assert.Contains(t, diagnostics.String(), `"code": "invalid_argument"`)
	}
	command := newReleasesCmd(nil, nil)
	command.SetArgs([]string{"--limit", "101"})
	err := command.Execute()
	assert.ErrorContains(t, err, "limit must be between 1 and 100")
}

func TestResolveListUsersExpandsAuthenticatedAliases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/user", r.URL.Path)
		_, _ = io.WriteString(w, `{"login":"octo"}`)
	}))
	defer server.Close()
	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	service := review.NewService(&gh.GitHubClient{HTTPClient: server.Client(), BaseURL: baseURL})
	command := newPRsCmd(service, git.NewRepositoryResolver(""))
	command.SetContext(context.Background())

	author, reviewer, err := resolveListUsers(command, service, " @me ", "@me")
	require.NoError(t, err)
	assert.Equal(t, "octo", author)
	assert.Equal(t, "octo", reviewer)
}

func TestReviewCommandRendersDecisionReadyJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/acme/project/pulls/12":
			_, _ = io.WriteString(w, `{"number":12,"title":"Improve coverage","head":{"sha":"abc"}}`)
		case "/repos/acme/project/pulls/12/reviews":
			_, _ = io.WriteString(w, `[]`)
		case "/repos/acme/project/commits/abc/check-runs":
			_, _ = io.WriteString(w, `{"check_runs":[]}`)
		case "/graphql":
			_, _ = io.WriteString(w, `{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":null}}}}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	command := newReviewCmd(review.NewService(&gh.GitHubClient{HTTPClient: server.Client(), BaseURL: baseURL}), git.NewRepositoryResolver(""))
	var rendered bytes.Buffer
	command.SetOut(&rendered)
	command.SetArgs([]string{"12", "--repo", "acme/project", "--format", "json"})
	require.NoError(t, command.Execute())
	var summary model.ReviewSummary
	require.NoError(t, json.Unmarshal(rendered.Bytes(), &summary))
	assert.Equal(t, 12, summary.PullRequest.Number)
	assert.Equal(t, "none", summary.Readiness.CIStatus)
	assert.Equal(t, "none", summary.Readiness.ReviewThreadsState)
}
