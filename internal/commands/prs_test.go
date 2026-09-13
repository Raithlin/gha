package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/internal/git"
	gh "github.com/raithlin/gha/internal/github"
	"github.com/raithlin/gha/internal/review"
	"github.com/raithlin/gha/pkg/model"
)

func TestPRsCommandShowsHelpWithoutConfiguration(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetArgs([]string{"prs", "--help"})

	require.NoError(t, root.Execute())
	assert.Contains(t, output.String(), "List pull requests in a GitHub repository")
	assert.Contains(t, output.String(), "--queue")
	assert.Contains(t, output.String(), "--path")
}

func TestPRsCommandRejectsConflictingModes(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"prs", "--mine", "--queue"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "use only one")
}

func TestPRsCommandRejectsListFiltersWithSpecialMode(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"prs", "--mine", "--state", "all"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be combined with list filters")
}

func TestPRsCommandAllowsLimitWithSpecialMode(t *testing.T) {
	command := newPRsCmd(nil, nil)
	require.NoError(t, command.Flags().Set("limit", "10"))

	assert.False(t, hasListFilters(command))
}

func TestPRsCommandValidatesListFiltersBeforeResolvingRepository(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"prs", "--state", "merged"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported state")
}

func TestPRsCommandResolvesBareHeadToAuthenticatedUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			_, _ = io.WriteString(w, `{"login":"alice"}`)
		case "/repos/Raithlin/gha/pulls":
			assert.Equal(t, "alice:feature", r.URL.Query().Get("head"))
			assert.Equal(t, "1", r.URL.Query().Get("page"))
			_, _ = io.WriteString(w, `[]`)
		default:
			t.Fatalf("unexpected request path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client := &gh.GitHubClient{HTTPClient: server.Client(), BaseURL: baseURL}
	command := newPRsCmd(review.NewService(client), git.NewRepositoryResolver(""))
	command.SetArgs([]string{"--repo", "Raithlin/gha", "--head", "feature", "--format", "json"})
	command.SetContext(context.Background())
	var output bytes.Buffer
	command.SetOut(&output)

	require.NoError(t, command.Execute())
	assert.JSONEq(t, `{"schema_version":"v1","repository":{"owner":"Raithlin","name":"gha"},"limit":30,"truncated":false,"pull_requests":[]}`, output.String())
}

func TestPRsCommandPaginatesFilteredResultsAndEmitsValidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/Raithlin/gha/pulls", r.URL.Path)
		switch r.URL.Query().Get("page") {
		case "1":
			prs := make([]*model.PullRequest, 100)
			for i := range prs {
				prs[i] = &model.PullRequest{Number: i + 1, User: model.User{Login: "other"}}
			}
			require.NoError(t, json.NewEncoder(w).Encode(prs))
		case "2":
			require.NoError(t, json.NewEncoder(w).Encode([]*model.PullRequest{{Number: 101, Title: "later match", User: model.User{Login: "alice"}}}))
		default:
			t.Fatalf("unexpected page %q", r.URL.Query().Get("page"))
		}
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client := &gh.GitHubClient{HTTPClient: server.Client(), BaseURL: baseURL}
	command := newPRsCmd(review.NewService(client), git.NewRepositoryResolver(""))
	command.SetArgs([]string{"--repo", "Raithlin/gha", "--author", "alice", "--limit", "1", "--format", "json"})
	command.SetContext(context.Background())
	var output bytes.Buffer
	command.SetOut(&output)

	require.NoError(t, command.Execute())
	var list model.PullRequestList
	require.NoError(t, json.Unmarshal(output.Bytes(), &list))
	assert.Equal(t, model.PullRequestListSchemaVersion, list.SchemaVersion)
	require.Len(t, list.PullRequests, 1)
	assert.Equal(t, 101, list.PullRequests[0].Number)
}

func TestPRsCommandRendersStructuredValidationErrors(t *testing.T) {
	command := newPRsCmd(nil, nil)
	var diagnostics bytes.Buffer
	command.SetErr(&diagnostics)
	command.SetArgs([]string{"--format", "json", "--limit", "0"})

	err := command.Execute()

	require.Error(t, err)
	assert.True(t, IsReportedError(err))
	var commandError model.CommandError
	require.NoError(t, json.Unmarshal(diagnostics.Bytes(), &commandError))
	assert.Equal(t, "invalid_argument", commandError.Code)
}

func TestPRsCommandValidatesSinceBeforeResolvingRepository(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	root.SetArgs([]string{"prs", "--since", "not-a-timestamp"})

	err := root.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --since")
}

func TestPRsCommandRendersStructuredArgumentErrors(t *testing.T) {
	command := newPRsCmd(nil, nil)
	var diagnostics bytes.Buffer
	command.SetErr(&diagnostics)
	command.SetArgs([]string{"unexpected", "--format", "json"})

	err := command.Execute()

	require.Error(t, err)
	assert.True(t, IsReportedError(err))
	var commandError model.CommandError
	require.NoError(t, json.Unmarshal(diagnostics.Bytes(), &commandError))
	assert.Equal(t, "invalid_argument", commandError.Code)
}
