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

func TestListReleasesUsesBoundedPaginationAndDecodesGitHubResponse(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/repos/Raithlin/gha/releases", r.URL.Path)
		assert.Equal(t, "25", r.URL.Query().Get("per_page"))
		assert.Equal(t, "2", r.URL.Query().Get("page"))
		_, _ = io.WriteString(w, `[{"id": 7, "tag_name": "v1.0.0", "name": "First release", "target_commitish": "main", "prerelease": true, "published_at": "2026-09-01T00:00:00Z", "author": {"login": "octo"}}]`)
	}))
	defer closeServer()

	releases, err := client.ListReleases(context.Background(), "Raithlin", "gha", interfaces.ListReleasesOptions{PerPage: 25, Page: 2})

	require.NoError(t, err)
	require.Len(t, releases, 1)
	assert.Equal(t, "v1.0.0", releases[0].TagName)
	assert.True(t, releases[0].Prerelease)
	assert.Equal(t, "octo", releases[0].Author.Login)
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
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
	}))
	defer closeServer()

	_, err := client.GetPullRequest(context.Background(), "Raithlin", "gha", 404)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestClientMethodsReturnProviderFailures(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"provider unavailable"}`, http.StatusServiceUnavailable)
	}))
	defer closeServer()

	for name, call := range map[string]func() error{
		"authenticated user": func() error { _, err := client.GetAuthenticatedUser(context.Background()); return err },
		"repositories":       func() error { _, err := client.ListRepositories(context.Background()); return err },
		"repository":         func() error { _, err := client.GetRepository(context.Background(), "acme", "project"); return err },
		"releases": func() error {
			_, err := client.ListReleases(context.Background(), "acme", "project", interfaces.ListReleasesOptions{})
			return err
		},
		"pull requests": func() error {
			_, err := client.ListPullRequests(context.Background(), "acme", "project", interfaces.ListPRsOptions{})
			return err
		},
		"pull request": func() error { _, err := client.GetPullRequest(context.Background(), "acme", "project", 1); return err },
		"create pull request": func() error {
			_, err := client.CreatePullRequest(context.Background(), "acme", "project", &model.PullRequestInput{})
			return err
		},
		"comparison": func() error {
			_, err := client.CompareBranches(context.Background(), "acme", "project", "main", "feature")
			return err
		},
		"update pull request": func() error {
			_, err := client.UpdatePullRequest(context.Background(), "acme", "project", 1, &model.PullRequestInput{})
			return err
		},
		"issues": func() error {
			_, err := client.ListIssues(context.Background(), "acme", "project", interfaces.ListIssuesOptions{})
			return err
		},
		"issue": func() error { _, err := client.GetIssue(context.Background(), "acme", "project", 1); return err },
		"comment": func() error {
			_, err := client.AddComment(context.Background(), "acme", "project", 1, "body")
			return err
		},
		"reviews": func() error { _, err := client.ListReviews(context.Background(), "acme", "project", 1); return err },
		"check runs": func() error {
			_, err := client.ListCheckRuns(context.Background(), "acme", "project", "abc")
			return err
		},
		"review threads": func() error {
			_, err := client.ListReviewThreads(context.Background(), "acme", "project", 1)
			return err
		},
		"submit review": func() error {
			_, err := client.SubmitReview(context.Background(), "acme", "project", 1, &model.ReviewInput{})
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.ErrorContains(t, call(), "503")
		})
	}

	safety, err := client.InspectBranchSafety(context.Background(), model.RepositoryRef{Owner: "acme", Name: "project"}, "feature")
	require.NoError(t, err)
	assert.Equal(t, "unavailable", safety.DefaultBranch.State)
	assert.Equal(t, "unavailable", safety.Permissions.State)
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
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

func TestCompareBranchesDecodesAheadAndBehindCounts(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/repos/Raithlin/gha/compare/main...feature", r.URL.Path)
		_, _ = io.WriteString(w, `{"status":"ahead","ahead_by":3,"behind_by":1}`)
	}))
	defer closeServer()

	comparison, err := client.CompareBranches(context.Background(), "Raithlin", "gha", "main", "feature")

	require.NoError(t, err)
	assert.Equal(t, "ahead", comparison.State)
	assert.Equal(t, 3, comparison.AheadBy)
	assert.Equal(t, 1, comparison.BehindBy)
}

func TestClientCoversAccountRepositoryAndMutationEndpoints(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			assert.Equal(t, http.MethodGet, r.Method)
			_, _ = io.WriteString(w, `{"login":"octo"}`)
		case "/user/repos":
			_, _ = io.WriteString(w, `[{"full_name":"acme/project"}]`)
		case "/repos/acme/project":
			_, _ = io.WriteString(w, `{"full_name":"acme/project"}`)
		case "/repos/acme/project/pulls":
			if r.Method == http.MethodGet {
				assert.Equal(t, "open", r.URL.Query().Get("state"))
				assert.Equal(t, "acme:feature", r.URL.Query().Get("head"))
				assert.Equal(t, "main", r.URL.Query().Get("base"))
				assert.Equal(t, "created", r.URL.Query().Get("sort"))
				assert.Equal(t, "desc", r.URL.Query().Get("direction"))
				assert.Equal(t, "2026-01-01", r.URL.Query().Get("since"))
				_, _ = io.WriteString(w, `[{"number":1}]`)
				return
			}
			assert.Equal(t, http.MethodPost, r.Method)
			_, _ = io.WriteString(w, `{"number":2}`)
		case "/repos/acme/project/pulls/2":
			assert.Equal(t, http.MethodPatch, r.Method)
			_, _ = io.WriteString(w, `{"number":2,"title":"updated"}`)
		case "/repos/acme/project/issues/3":
			_, _ = io.WriteString(w, `{"number":3,"title":"issue"}`)
		case "/repos/acme/project/issues/3/comments":
			assert.Equal(t, http.MethodPost, r.Method)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.JSONEq(t, `{"body":"hello"}`, string(body))
			_, _ = io.WriteString(w, `{"id":4,"body":"hello"}`)
		case "/repos/acme/project/pulls/2/reviews":
			assert.Equal(t, http.MethodPost, r.Method)
			_, _ = io.WriteString(w, `{"id":5,"state":"APPROVED"}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer closeServer()

	user, err := client.GetAuthenticatedUser(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "octo", user.Login)
	repositories, err := client.ListRepositories(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "acme/project", repositories[0].FullName)
	repository, err := client.GetRepository(context.Background(), "acme", "project")
	require.NoError(t, err)
	assert.Equal(t, "acme/project", repository.FullName)
	pullRequests, err := client.ListPullRequests(context.Background(), "acme", "project", interfaces.ListPRsOptions{State: "open", Head: "acme:feature", Base: "main", Sort: "created", Direction: "desc", Since: "2026-01-01", PerPage: 10, Page: 2})
	require.NoError(t, err)
	assert.Equal(t, 1, pullRequests[0].Number)
	updated, err := client.UpdatePullRequest(context.Background(), "acme", "project", 2, &model.PullRequestInput{Title: "updated"})
	require.NoError(t, err)
	assert.Equal(t, "updated", updated.Title)
	issue, err := client.GetIssue(context.Background(), "acme", "project", 3)
	require.NoError(t, err)
	assert.Equal(t, "issue", issue.Title)
	comment, err := client.AddComment(context.Background(), "acme", "project", 3, "hello")
	require.NoError(t, err)
	assert.Equal(t, int64(4), comment.ID)
	review, err := client.SubmitReview(context.Background(), "acme", "project", 2, &model.ReviewInput{Event: "APPROVE"})
	require.NoError(t, err)
	assert.Equal(t, int64(5), review.ID)
}

func TestListIssuesIncludesAllFiltersAndUtilityCases(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "open", r.URL.Query().Get("state"))
		assert.Equal(t, "bug,urgent", r.URL.Query().Get("labels"))
		assert.Equal(t, "updated", r.URL.Query().Get("sort"))
		assert.Equal(t, "asc", r.URL.Query().Get("direction"))
		assert.Equal(t, "2026-01-01", r.URL.Query().Get("since"))
		assert.Equal(t, "10", r.URL.Query().Get("per_page"))
		assert.Equal(t, "2", r.URL.Query().Get("page"))
		_, _ = io.WriteString(w, `[]`)
	}))
	defer closeServer()
	issues, err := client.ListIssues(context.Background(), "acme", "project", interfaces.ListIssuesOptions{State: "open", Labels: []string{"bug", "urgent"}, Sort: "updated", Direction: "asc", Since: "2026-01-01", PerPage: 10, Page: 2})
	require.NoError(t, err)
	assert.Empty(t, issues)
	assert.Equal(t, "", joinStrings(nil, ","))
	assert.Equal(t, "one", joinStrings([]string{"one"}, ","))
	assert.Equal(t, "one,two", joinStrings([]string{"one", "two"}, ","))

	created, err := NewGitHubClient("token")
	require.NoError(t, err)
	assert.Equal(t, "https://api.github.com/", created.BaseURL.String())
}

func TestInspectBranchSafetyPreservesUnavailableIndependentSignals(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/acme/project":
			_, _ = io.WriteString(w, `{}`)
		case "/repos/acme/project/branches/feature":
			http.Error(w, "no access", http.StatusForbidden)
		case "/repos/acme/project/pulls":
			_, _ = io.WriteString(w, `[]`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer closeServer()

	safety, err := client.InspectBranchSafety(context.Background(), model.RepositoryRef{Owner: "acme", Name: "project"}, "feature")
	require.NoError(t, err)
	assert.Equal(t, "unavailable", safety.DefaultBranch.State)
	assert.Equal(t, "unavailable", safety.Permissions.State)
	assert.Equal(t, "unavailable", safety.Protection.State)
	assert.Equal(t, "available", safety.Requests.State)
	assert.Equal(t, "not_applicable", safety.Merge.State)
}

func TestGitHubClientReportsInvalidResponsesAndIncompleteGraphQLPages(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			_, _ = io.WriteString(w, "not json")
		case "/graphql":
			_, _ = io.WriteString(w, `{"data":{"repository":null}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer closeServer()
	_, err := client.GetAuthenticatedUser(context.Background())
	assert.ErrorContains(t, err, "failed to decode response")
	_, err = client.ListReviewThreads(context.Background(), "acme", "project", 1)
	assert.ErrorContains(t, err, "did not include pull request")
}

func TestGitHubClientReportsMissingGraphQLCursorAndBranchSafetyEarlyReturn(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/graphql":
			_, _ = io.WriteString(w, `{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[],"pageInfo":{"hasNextPage":true,"endCursor":null}}}}}}`)
		case "/repos/acme/project":
			http.Error(w, "offline", http.StatusServiceUnavailable)
		case "/repos/acme/project/branches/feature":
			http.Error(w, "offline", http.StatusServiceUnavailable)
		case "/repos/acme/project/pulls":
			http.Error(w, "offline", http.StatusServiceUnavailable)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer closeServer()
	_, err := client.ListReviewThreads(context.Background(), "acme", "project", 1)
	assert.ErrorContains(t, err, "without a cursor")
	safety, err := client.InspectBranchSafety(context.Background(), model.RepositoryRef{Owner: "acme", Name: "project"}, "feature")
	require.NoError(t, err)
	assert.Equal(t, "unavailable", safety.Requests.State)
	assert.Equal(t, "unavailable", safety.Merge.State)
}

func TestInspectBranchSafetyTreatsNullRepositoryAsUnavailable(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/acme/project":
			_, _ = io.WriteString(w, "null")
		case "/repos/acme/project/branches/feature":
			_, _ = io.WriteString(w, `{"protected":false}`)
		case "/repos/acme/project/pulls":
			_, _ = io.WriteString(w, `[]`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer closeServer()

	safety, err := client.InspectBranchSafety(context.Background(), model.RepositoryRef{Owner: "acme", Name: "project"}, "feature")
	require.NoError(t, err)
	assert.Equal(t, "unavailable", safety.DefaultBranch.State)
	assert.Equal(t, "unavailable", safety.Permissions.State)
	assert.Equal(t, "available", safety.Protection.State)
}

func TestInspectBranchSafetyTreatsNullOpenPullRequestAsUnavailableMergeSignal(t *testing.T) {
	client, closeServer := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/acme/project":
			_, _ = io.WriteString(w, `{"default_branch":"main","permissions":{"push":true}}`)
		case "/repos/acme/project/branches/feature":
			_, _ = io.WriteString(w, `{"protected":false}`)
		case "/repos/acme/project/pulls":
			_, _ = io.WriteString(w, `[null]`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer closeServer()

	safety, err := client.InspectBranchSafety(context.Background(), model.RepositoryRef{Owner: "acme", Name: "project"}, "feature")
	require.NoError(t, err)
	assert.Equal(t, "available", safety.Requests.State)
	assert.Equal(t, "unavailable", safety.Merge.State)
}
