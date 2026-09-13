package commands

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/internal/git"
	gh "github.com/raithlin/gha/internal/github"
	"github.com/raithlin/gha/internal/review"
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

func TestReleaseCommandRequiresReleaseWindow(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"release"})

	err := root.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "required flag(s) \"since\" not set")
}

func TestReleaseCommandGeneratesNotes(t *testing.T) {
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
	command := newReleaseCmd(review.NewService(client), git.NewRepositoryResolver(""))
	command.SetArgs([]string{"--repo", "Raithlin/gha", "--since", "2026-09-01T00:00:00Z"})
	command.SetContext(context.Background())
	var output bytes.Buffer
	command.SetOut(&output)

	require.NoError(t, command.Execute())
	assert.Contains(t, output.String(), "#42 Ship it (alice)")
}

func TestReleaseCommandTreatsDateOnlySinceAsLocalMidnight(t *testing.T) {
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
	command := newReleaseCmd(review.NewService(client), git.NewRepositoryResolver(""))
	command.SetArgs([]string{"--repo", "Raithlin/gha", "--since", "2025-09-01"})
	command.SetContext(context.Background())

	require.NoError(t, command.Execute())
}

func TestParseReleaseSincePreservesExplicitTimezone(t *testing.T) {
	timestamp, err := parseReleaseSince("2025-09-01T00:00:00-04:00", time.FixedZone("UTC+2", 2*60*60))

	require.NoError(t, err)
	assert.Equal(t, "2025-09-01T00:00:00-04:00", timestamp.Format(time.RFC3339))
}
