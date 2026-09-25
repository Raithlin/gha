package github

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/pkg/model"
)

type releaseTransport func(*http.Request) *http.Response

func (f releaseTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request), nil
}

func releaseClient(t *testing.T, transport releaseTransport) *GitHubClient {
	t.Helper()
	baseURL, err := url.Parse("https://api.example.test/")
	require.NoError(t, err)
	return &GitHubClient{BaseURL: baseURL, HTTPClient: &http.Client{Transport: transport}}
}

func response(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Status: http.StatusText(code), Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

func TestReleaseInspectDistinguishesAbsentAndExistingRemoteTag(t *testing.T) {
	for _, test := range []struct {
		name           string
		code           int
		body, expected string
	}{
		{"absent", 404, `{}`, ""}, {"existing", 200, `{"ref":"refs/tags/v1.2.3"}`, "refs/tags/v1.2.3"},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := releaseClient(t, func(request *http.Request) *http.Response {
				switch {
				case strings.HasSuffix(request.URL.Path, "/git/ref/tags/v1.2.3"):
					return response(test.code, test.body)
				case strings.Contains(request.URL.Path, "/check-runs"):
					return response(200, `{"check_runs":[{"name":"test","status":"completed","conclusion":"success"}]}`)
				default:
					return response(200, `{"default_branch":"main","permissions":{"push":true}}`)
				}
			})
			state, err := client.Inspect(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "abc123", "v1.2.3")
			require.NoError(t, err)
			assert.Equal(t, test.expected, state.RemoteTag)
			assert.Equal(t, "main", state.Repository.DefaultBranch)
			require.Len(t, state.Checks, 1)
		})
	}
}

func TestReleaseObservationKeepsPendingDistinctFromCompleted(t *testing.T) {
	for _, test := range []struct{ status, conclusion, expected string }{
		{"in_progress", "", "triggered"}, {"completed", "success", "completed"}, {"completed", "failure", "failed"},
	} {
		client := releaseClient(t, func(request *http.Request) *http.Response {
			assert.Equal(t, "abc123", request.URL.Query().Get("head_sha"))
			assert.Empty(t, request.URL.Query().Get("branch"))
			return response(200, `{"workflow_runs":[{"head_branch":"v1.2.3","event":"push","status":"`+test.status+`","conclusion":"`+test.conclusion+`"}]}`)
		})
		observation, err := client.Observe(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, ".github/workflows/release.yml", "v1.2.3", "abc123")
		require.NoError(t, err)
		assert.Equal(t, test.expected, observation.State)
	}
}

func TestReleaseInspectFailsWhenTagStatusCannotBeVerified(t *testing.T) {
	client := releaseClient(t, func(request *http.Request) *http.Response {
		switch {
		case strings.Contains(request.URL.Path, "/check-runs"):
			return response(200, `{"check_runs":[]}`)
		case strings.Contains(request.URL.Path, "/git/ref/tags/"):
			return response(403, `{"message":"forbidden"}`)
		default:
			return response(200, `{"default_branch":"main","permissions":{"push":true}}`)
		}
	})
	_, err := client.Inspect(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "abc123", "v1.2.3")
	require.ErrorContains(t, err, "inspect origin tag")
}

func TestReleaseObservationReportsUnavailableWhenRunNotVisible(t *testing.T) {
	client := releaseClient(t, func(_ *http.Request) *http.Response { return response(200, `{"workflow_runs":[]}`) })
	observation, err := client.Observe(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, ".github/workflows/release.yml", "v1.2.3", "abc123")
	require.NoError(t, err)
	assert.Equal(t, "unavailable", observation.State)
}

func TestReleaseProviderKeepsUnavailableChecksAndAPIErrorDistinct(t *testing.T) {
	client := releaseClient(t, func(request *http.Request) *http.Response {
		switch {
		case strings.Contains(request.URL.Path, "/check-runs"):
			return response(503, `{"message":"unavailable"}`)
		case strings.Contains(request.URL.Path, "/git/ref/tags/"):
			return response(404, `{}`)
		default:
			return response(200, `{"default_branch":"main","permissions":{"push":true}}`)
		}
	})
	state, err := client.Inspect(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "abc123", "v1.2.3")
	require.NoError(t, err)
	require.Error(t, state.ChecksError)
	assert.Empty(t, state.RemoteTag)

	client = releaseClient(t, func(_ *http.Request) *http.Response { return response(503, `{"message":"unavailable"}`) })
	_, err = client.Inspect(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "abc123", "v1.2.3")
	require.Error(t, err)
	_, err = client.Observe(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, ".github/workflows/release.yml", "v1.2.3", "abc123")
	require.Error(t, err)
}
