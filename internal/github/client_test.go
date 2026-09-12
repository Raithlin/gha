package github

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/internal/interfaces"
	"github.com/raithlin/gha/pkg/model"
)

func newTestClient(t *testing.T, handler http.Handler) (*GitHubClient, func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	return &GitHubClient{HTTPClient: server.Client(), BaseURL: baseURL, Token: "test-token"}, server.Close
}

func TestGetPullRequestDecodesGitHubResponse(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/Raithlin/gha/pulls/12", r.URL.Path)
		assert.Equal(t, "token test-token", r.Header.Get("Authorization"))
		_, _ = io.WriteString(w, `{
          "id": 1,
          "number": 12,
          "title": "Improve reviews",
          "user": {"login": "octo", "id": 2, "type": "User"},
          "created_at": "2026-09-01T00:00:00Z",
          "mergeable_state": "clean",
          "head": {"ref": "feature", "sha": "abc"},
          "base": {"ref": "main", "sha": "def"},
          "requested_reviewers": [{"login": "stephen", "id": 3, "type": "User"}]
        }`)
	}))
	defer closeServer()

	pr, err := client.GetPullRequest(context.Background(), "Raithlin", "gha", 12)
	require.NoError(t, err)
	assert.Equal(t, "octo", pr.User.Login)
	assert.Equal(t, "2026-09-01T00:00:00Z", pr.CreatedAt)
	assert.Equal(t, "clean", pr.MergeableState)
	assert.Equal(t, "feature", pr.Head.Ref)
	assert.Equal(t, "stephen", pr.RequestedReviewers[0].Login)
}

func TestListIssuesUsesAssigneeAndDecodesPullRequestReference(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "stephen", r.URL.Query().Get("assignee"))
		assert.Equal(t, "100", r.URL.Query().Get("per_page"))
		_, _ = io.WriteString(w, `[{"number": 5, "user": {"login":"octo"}, "labels":[{"name":"bug"}], "pull_request":{"url":"https://api.github.com/pr/5"}}]`)
	}))
	defer closeServer()

	issues, err := client.ListIssues(context.Background(), "Raithlin", "gha", interfaces.ListIssuesOptions{Assignee: "stephen", PerPage: 100})
	require.NoError(t, err)
	require.Len(t, issues, 1)
	assert.Equal(t, "octo", issues[0].User.Login)
	assert.Equal(t, "bug", issues[0].Labels[0].Name)
	require.NotNil(t, issues[0].PullRequest)
}

func TestCreatePullRequestUsesInputSchema(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/repos/Raithlin/gha/pulls", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"title":"Improve reviews","body":"Description","head":"feature","base":"main"}`, string(body))
		_, _ = io.WriteString(w, `{"number": 13, "user": {"login":"octo"}}`)
	}))
	defer closeServer()

	pr, err := client.CreatePullRequest(context.Background(), "Raithlin", "gha", &model.PullRequestInput{
		Title: "Improve reviews", Body: "Description", Head: "feature", Base: "main",
	})
	require.NoError(t, err)
	assert.Equal(t, 13, pr.Number)
}
