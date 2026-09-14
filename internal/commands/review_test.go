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
