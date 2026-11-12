package github

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestReleaseStatusCorrelatesAnnotatedTagExactCommitWorkflowAndAttempt(t *testing.T) {
	client := releaseClient(t, func(request *http.Request) *http.Response {
		switch {
		case strings.Contains(request.URL.Path, "/git/ref/tags/"):
			return response(200, `{"object":{"type":"tag","sha":"tag-object"}}`)
		case strings.Contains(request.URL.Path, "/git/tags/tag-object"):
			return response(200, `{"object":{"type":"commit","sha":"abc123"}}`)
		case strings.Contains(request.URL.Path, "/actions/runs"):
			require.Equal(t, "abc123", request.URL.Query().Get("head_sha"))
			return response(200, `{"total_count":4,"workflow_runs":[
				{"id":5,"workflow_id":3,"run_attempt":2,"path":".github/workflows/release.yml","head_branch":"v1.2.3","head_sha":"wrong","event":"push","status":"completed","conclusion":"success"},
				{"id":6,"workflow_id":3,"run_attempt":2,"path":".github/workflows/other.yml","head_branch":"v1.2.3","head_sha":"abc123","event":"push","status":"completed","conclusion":"success"},
				{"id":7,"workflow_id":3,"run_attempt":1,"path":".github/workflows/release.yml","head_branch":"v1.2.3","head_sha":"abc123","event":"push","status":"completed","conclusion":"failure"},
				{"id":7,"workflow_id":3,"run_attempt":2,"path":".github/workflows/release.yml","head_branch":"v1.2.3","head_sha":"abc123","event":"push","status":"completed","conclusion":"success","html_url":"https://github.test/run/7"}]}`)
		case strings.Contains(request.URL.Path, "/releases/tags/"):
			return response(200, `{"tag_name":"v1.2.3","html_url":"https://github.test/release","draft":false,"prerelease":true}`)
		default:
			return response(404, `{}`)
		}
	})
	result, err := client.ReadStatus(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "v1.2.3", ".github/workflows/release.yml")
	require.NoError(t, err)
	require.Equal(t, "published", result.State)
	require.Equal(t, "abc123", result.Commit)
	require.Equal(t, int64(7), result.Workflow.RunID)
	require.Equal(t, 2, result.Workflow.Attempt)
	require.Equal(t, "available", result.HostedRelease.State)
}

func TestReleaseStatusDistinguishesPendingFailureCancellationAndUndiscovered(t *testing.T) {
	for _, test := range []struct{ status, conclusion, expected string }{
		{"queued", "", "pending"}, {"completed", "failure", "failed"}, {"completed", "cancelled", "cancelled"},
	} {
		t.Run(test.expected, func(t *testing.T) {
			client := releaseClient(t, func(request *http.Request) *http.Response {
				switch {
				case strings.Contains(request.URL.Path, "/git/ref/tags/"):
					return response(200, `{"object":{"type":"commit","sha":"abc123"}}`)
				case strings.Contains(request.URL.Path, "/actions/runs"):
					return response(200, `{"total_count":1,"workflow_runs":[{"id":9,"workflow_id":2,"run_attempt":1,"path":".github/workflows/release.yml","head_branch":"v1.2.3","head_sha":"abc123","event":"push","status":"`+test.status+`","conclusion":"`+test.conclusion+`"}]}`)
				default:
					return response(404, `{}`)
				}
			})
			result, err := client.ReadStatus(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "v1.2.3", "")
			require.NoError(t, err)
			require.Equal(t, test.expected, result.State)
		})
	}
	client := releaseClient(t, func(request *http.Request) *http.Response {
		if strings.Contains(request.URL.Path, "/git/ref/tags/") {
			return response(200, `{"object":{"type":"commit","sha":"abc123"}}`)
		}
		if strings.Contains(request.URL.Path, "/actions/runs") {
			return response(200, `{"total_count":0,"workflow_runs":[]}`)
		}
		return response(404, `{}`)
	})
	result, err := client.ReadStatus(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "v1.2.3", "")
	require.NoError(t, err)
	require.Equal(t, "undiscovered", result.State)
}

func TestReleaseStatusKeepsProviderFailureUnavailable(t *testing.T) {
	client := releaseClient(t, func(request *http.Request) *http.Response {
		if strings.Contains(request.URL.Path, "/git/ref/tags/") {
			return response(200, `{"object":{"type":"commit","sha":"abc123"}}`)
		}
		return response(503, `{}`)
	})
	result, err := client.ReadStatus(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "v1.2.3", "")
	require.NoError(t, err)
	require.Equal(t, "unavailable", result.State)
	require.Equal(t, "unavailable", result.Workflow.State)
	require.Equal(t, "unavailable", result.HostedRelease.State)
}

func TestReleaseStatusDoesNotTreatMissingActionsEndpointAsNoRun(t *testing.T) {
	client := releaseClient(t, func(request *http.Request) *http.Response {
		if strings.Contains(request.URL.Path, "/git/ref/tags/") {
			return response(200, `{"object":{"type":"commit","sha":"abc123"}}`)
		}
		return response(404, `{}`)
	})
	result, err := client.ReadStatus(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "v1.2.3", "")
	require.NoError(t, err)
	require.Equal(t, "unavailable", result.Workflow.State)
	// A missing hosted release is still an independently observed absence.
	require.Equal(t, "absent", result.HostedRelease.State)
}

func TestReleaseStatusKeepsAbsentTagDistinctFromUnavailableTag(t *testing.T) {
	for _, test := range []struct {
		code            int
		tagState, state string
	}{
		{404, "absent", "undiscovered"}, {503, "unavailable", "unavailable"},
	} {
		client := releaseClient(t, func(request *http.Request) *http.Response {
			require.Contains(t, request.URL.Path, "/git/ref/tags/")
			return response(test.code, `{}`)
		})
		result, err := client.ReadStatus(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "v1.2.3", "")
		require.NoError(t, err)
		require.Equal(t, test.tagState, result.TagState)
		require.Equal(t, test.state, result.State)
	}
}

func TestReleaseStatusMarksIncompleteRunInventoryUnavailable(t *testing.T) {
	client := releaseClient(t, func(request *http.Request) *http.Response {
		switch {
		case strings.Contains(request.URL.Path, "/git/ref/tags/"):
			return response(200, `{"object":{"type":"commit","sha":"abc123"}}`)
		case strings.Contains(request.URL.Path, "/actions/runs"):
			return response(200, `{"total_count":101,"workflow_runs":[{"id":9,"head_branch":"other","head_sha":"abc123","event":"push"}]}`)
		default:
			return response(404, `{}`)
		}
	})
	result, err := client.ReadStatus(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "v1.2.3", "")
	require.NoError(t, err)
	require.True(t, result.Workflow.Truncated)
	require.Equal(t, "unavailable", result.Workflow.State)
}

func TestReleaseStatusRequiresWorkflowSelectionWhenSeveralMatch(t *testing.T) {
	client := releaseClient(t, func(request *http.Request) *http.Response {
		switch {
		case strings.Contains(request.URL.Path, "/git/ref/tags/"):
			return response(200, `{"object":{"type":"commit","sha":"abc123"}}`)
		case strings.Contains(request.URL.Path, "/actions/runs"):
			return response(200, `{"total_count":2,"workflow_runs":[
				{"id":9,"workflow_id":1,"path":".github/workflows/release.yml","head_branch":"v1.2.3","head_sha":"abc123","event":"push","status":"completed","conclusion":"success"},
				{"id":8,"workflow_id":2,"path":".github/workflows/other.yml","head_branch":"v1.2.3","head_sha":"abc123","event":"push","status":"completed","conclusion":"success"}]}`)
		default:
			return response(200, `{"tag_name":"v1.2.3","draft":true}`)
		}
	})
	result, err := client.ReadStatus(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "v1.2.3", "")
	require.NoError(t, err)
	require.Equal(t, "ambiguous", result.Workflow.State)
	require.Equal(t, "draft", result.HostedRelease.State)
	require.Equal(t, "unavailable", result.State)
}
