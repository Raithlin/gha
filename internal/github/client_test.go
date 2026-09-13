package github

import (
	"context"
	"encoding/json"
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
		assert.Contains(t, r.Header.Get("Accept"), "application/vnd.github.text+json")
		assert.Contains(t, r.Header.Get("Accept"), "application/vnd.github+json")
		assert.Equal(t, GitHubAPIVersion, r.Header.Get("X-GitHub-Api-Version"))
		_, _ = io.WriteString(w, `{
          "id": 1,
          "number": 12,
          "title": "Improve reviews",
		  "body": "<p>Improve terminal output</p>",
		  "body_text": "Improve terminal output",
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
	assert.Equal(t, "Improve terminal output", pr.BodyText)
	assert.Equal(t, "feature", pr.Head.Ref)
	assert.Equal(t, "stephen", pr.RequestedReviewers[0].Login)
}

func TestInspectBranchSafetyKeepsIndependentGitHubSignals(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/Raithlin/gha":
			_, _ = io.WriteString(w, `{"default_branch":"main","permissions":{"push":true}}`)
		case r.URL.EscapedPath() == "/repos/Raithlin/gha/branches/feature%2Fapi":
			_, _ = io.WriteString(w, `{"protected":true}`)
		case r.URL.Path == "/repos/Raithlin/gha/pulls":
			assert.Equal(t, "open", r.URL.Query().Get("state"))
			assert.Equal(t, "Raithlin:feature/api", r.URL.Query().Get("head"))
			assert.Equal(t, "100", r.URL.Query().Get("per_page"))
			assert.Equal(t, "1", r.URL.Query().Get("page"))
			_, _ = io.WriteString(w, `[{"number":12,"title":"Feature API"}]`)
		case r.URL.Path == "/repos/Raithlin/gha/pulls/12":
			_, _ = io.WriteString(w, `{"number":12,"mergeable":true}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.EscapedPath())
		}
	}))
	defer closeServer()

	safety, err := client.InspectBranchSafety(context.Background(), model.RepositoryRef{Owner: "Raithlin", Name: "gha"}, "feature/api")

	require.NoError(t, err)
	assert.Equal(t, "github", safety.Provider)
	assert.NotEmpty(t, safety.CheckedAt)
	assert.Equal(t, "available", safety.Requests.State)
	require.Len(t, safety.OpenPullRequests, 1)
	assert.Equal(t, "available", safety.Protection.State)
	require.NotNil(t, safety.Protected)
	assert.True(t, *safety.Protected)
	assert.Equal(t, "available", safety.Permissions.State)
	require.NotNil(t, safety.CanPush)
	assert.True(t, *safety.CanPush)
	assert.Equal(t, "available", safety.DefaultBranch.State)
	assert.Equal(t, "main", safety.DefaultBranchName)
	require.NotNil(t, safety.IsDefault)
	assert.False(t, *safety.IsDefault)
	assert.Equal(t, "available", safety.Merge.State)
	require.NotNil(t, safety.Mergeable)
	assert.True(t, *safety.Mergeable)
}

func TestListReviewsPaginatesAllPages(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "100", r.URL.Query().Get("per_page"))
		switch r.URL.Query().Get("page") {
		case "1":
			reviews := make([]*model.Review, 100)
			for i := range reviews {
				reviews[i] = &model.Review{ID: int64(i + 1), User: model.User{Login: "alice"}, State: "APPROVED", SubmittedAt: "2026-09-01T00:00:00Z"}
			}
			require.NoError(t, json.NewEncoder(w).Encode(reviews))
		case "2":
			require.NoError(t, json.NewEncoder(w).Encode([]*model.Review{
				{ID: 101, User: model.User{Login: "alice"}, State: "CHANGES_REQUESTED", SubmittedAt: "2026-09-02T00:00:00Z"},
			}))
		default:
			t.Fatalf("unexpected review page %q", r.URL.Query().Get("page"))
		}
	}))
	defer closeServer()

	reviews, err := client.ListReviews(context.Background(), "Raithlin", "gha", 12)
	require.NoError(t, err)
	require.Len(t, reviews, 101)
	assert.Equal(t, "CHANGES_REQUESTED", reviews[100].State)
}

func TestGetPullRequestReturnsGitHubError(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
	}))
	defer closeServer()

	_, err := client.GetPullRequest(context.Background(), "Raithlin", "gha", 404)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
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

func TestListCheckRunsUsesHeadSHAAndDecodesResponse(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/Raithlin/gha/commits/abc123/check-runs", r.URL.Path)
		assert.Equal(t, "latest", r.URL.Query().Get("filter"))
		assert.Equal(t, "100", r.URL.Query().Get("per_page"))
		_, _ = io.WriteString(w, `{"check_runs":[{"name":"test","status":"completed","conclusion":"success"}]}`)
	}))
	defer closeServer()

	checks, err := client.ListCheckRuns(context.Background(), "Raithlin", "gha", "abc123")
	require.NoError(t, err)
	require.Len(t, checks, 1)
	assert.Equal(t, "test", checks[0].Name)
	assert.Equal(t, "success", checks[0].Conclusion)
}

func TestListReviewThreadsPaginatesAndDecodesResolution(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/graphql", r.URL.Path)
		assert.Equal(t, "token test-token", r.Header.Get("Authorization"))

		var request struct {
			Query     string `json:"query"`
			Variables struct {
				Owner  string  `json:"owner"`
				Repo   string  `json:"repo"`
				Number int     `json:"number"`
				Cursor *string `json:"cursor"`
			} `json:"variables"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		assert.Contains(t, request.Query, "reviewThreads")
		assert.Equal(t, "Raithlin", request.Variables.Owner)
		assert.Equal(t, "gha", request.Variables.Repo)
		assert.Equal(t, 12, request.Variables.Number)

		if request.Variables.Cursor == nil {
			_, _ = io.WriteString(w, `{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[{"isResolved":false}],"pageInfo":{"hasNextPage":true,"endCursor":"cursor-1"}}}}}}`)
			return
		}
		assert.Equal(t, "cursor-1", *request.Variables.Cursor)
		_, _ = io.WriteString(w, `{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[{"isResolved":true}],"pageInfo":{"hasNextPage":false,"endCursor":null}}}}}}`)
	}))
	defer closeServer()

	threads, err := client.ListReviewThreads(context.Background(), "Raithlin", "gha", 12)
	require.NoError(t, err)
	require.Len(t, threads, 2)
	assert.False(t, threads[0].IsResolved)
	assert.True(t, threads[1].IsResolved)
}

func TestListReviewThreadsReturnsGraphQLErrors(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"errors":[{"message":"reviewThreads requires additional permissions"}]}`)
	}))
	defer closeServer()

	_, err := client.ListReviewThreads(context.Background(), "Raithlin", "gha", 12)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "additional permissions")
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
