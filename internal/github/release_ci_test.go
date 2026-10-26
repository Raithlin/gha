package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Models the native skip plus successful full package validation that blocked v0.1.0.
func releaseCIResponse(request *http.Request) *http.Response {
	switch {
	case request.URL.Path == "/repos/acme/tool/branches/main":
		return response(200, `{"protected":false}`)
	case strings.Contains(request.URL.Path, "/rules/branches/"):
		return response(200, `[]`)
	case strings.HasSuffix(request.URL.Path, "/status"):
		return response(200, `{"total_count":0,"statuses":[]}`)
	case strings.HasSuffix(request.URL.Path, "/runs/10/attempts/2/jobs"):
		return response(200, `{"total_count":1,"jobs":[{"head_sha":"abc123","check_run_url":"https://api.example.test/repos/acme/tool/check-runs/101"}]}`)
	case strings.HasSuffix(request.URL.Path, "/runs/20/attempts/1/jobs"):
		return response(200, `{"total_count":3,"jobs":[{"head_sha":"abc123","check_run_url":"https://api.example.test/repos/acme/tool/check-runs/201"},{"head_sha":"abc123","check_run_url":"https://api.example.test/repos/acme/tool/check-runs/202"},{"head_sha":"abc123","check_run_url":"https://api.example.test/repos/acme/tool/check-runs/203"}]}`)
	case strings.HasSuffix(request.URL.Path, "/actions/runs"):
		return response(200, `{"total_count":5,"workflow_runs":[
   {"id":10,"workflow_id":1,"check_suite_id":10,"run_attempt":2,"name":"ordinary CI","head_sha":"abc123","head_branch":"main","event":"push","status":"completed","conclusion":"success"},
   {"id":20,"workflow_id":2,"check_suite_id":20,"run_attempt":1,"name":"packages","head_sha":"abc123","head_branch":"main","event":"workflow_dispatch","status":"completed","conclusion":"success"},
   {"id":9,"workflow_id":1,"check_suite_id":9,"run_attempt":1,"head_sha":"abc123","head_branch":"main","event":"push","status":"completed","conclusion":"failure"},
   {"id":30,"workflow_id":3,"check_suite_id":30,"run_attempt":1,"head_sha":"other","head_branch":"main","event":"push","status":"completed","conclusion":"failure"},
   {"id":40,"workflow_id":4,"check_suite_id":40,"run_attempt":1,"head_sha":"abc123","head_branch":"topic","event":"pull_request","status":"completed","conclusion":"failure"}]}`)
	case strings.HasSuffix(request.URL.Path, "/check-runs"):
		return response(200, `{"total_count":6,"check_runs":[
   {"id":101,"name":"native","head_sha":"abc123","status":"completed","conclusion":"skipped","app":{"id":1,"slug":"github-actions"},"check_suite":{"id":10}},
   {"id":201,"name":"Arch","head_sha":"abc123","status":"completed","conclusion":"success","app":{"id":1,"slug":"github-actions"},"check_suite":{"id":20}},
   {"id":202,"name":"Debian","head_sha":"abc123","status":"completed","conclusion":"success","app":{"id":1,"slug":"github-actions"},"check_suite":{"id":20}},
   {"id":203,"name":"Fedora","head_sha":"abc123","status":"completed","conclusion":"success","app":{"id":1,"slug":"github-actions"},"check_suite":{"id":20}},
   {"name":"old native","head_sha":"abc123","status":"completed","conclusion":"failure","app":{"id":1,"slug":"github-actions"},"check_suite":{"id":9}},
   {"name":"wrong sha","head_sha":"other","status":"completed","conclusion":"failure","app":{"id":2,"slug":"external"}}]}`)
	default:
		panic("unexpected endpoint: " + request.URL.String())
	}
}

func TestReleaseCISelectsExactCommitAndLatestWorkflowRuns(t *testing.T) {
	client := releaseClient(t, func(req *http.Request) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/actions/runs") {
			assert.Equal(t, "abc123", req.URL.Query().Get("head_sha"))
		}
		if strings.HasSuffix(req.URL.Path, "/check-runs") {
			assert.Equal(t, "latest", req.URL.Query().Get("filter"))
		}
		return releaseCIResponse(req)
	})
	ci, checks, err := client.inspectReleaseCI(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "main", "abc123")
	require.NoError(t, err)
	assert.Equal(t, "available", ci.Policy.State)
	assert.Equal(t, "available", ci.Workflows.State)
	require.Len(t, ci.Runs, 2)
	assert.Equal(t, 2, ci.Runs[0].RunAttempt)
	require.Len(t, checks, 4)
	assert.Equal(t, "native", checks[0].Name)
}

func TestReleaseCIRequiredChecksCombineProtectionAndRules(t *testing.T) {
	client := releaseClient(t, func(req *http.Request) *http.Response {
		switch {
		case req.URL.Path == "/repos/acme/tool/branches/main":
			return response(200, `{"protected":true}`)
		case strings.HasSuffix(req.URL.Path, "/protection"):
			return response(200, `{"required_status_checks":{"contexts":["legacy"],"checks":[{"context":"Arch","app_id":1}]}}`)
		case strings.Contains(req.URL.Path, "/rules/branches/"):
			return response(200, `[{"type":"required_status_checks","parameters":{"required_status_checks":[{"context":"Debian","integration_id":1}]}}]`)
		case strings.HasSuffix(req.URL.Path, "/status"):
			return response(200, `{"total_count":1,"statuses":[{"context":"legacy","state":"success"}]}`)
		default:
			return releaseCIResponse(req)
		}
	})
	ci, checks, err := client.inspectReleaseCI(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "main", "abc123")
	require.NoError(t, err)
	assert.Equal(t, "available", ci.Policy.State)
	require.Len(t, ci.RequiredChecks, 3)
	require.Len(t, checks, 5)
	assert.Equal(t, "legacy", checks[4].Name)
}

func TestReleaseCIUnavailableInventoriesFailClosed(t *testing.T) {
	for _, endpoint := range []string{"/branches/main", "/rules/branches/main", "/actions/runs", "/check-runs", "/status"} {
		for _, code := range []int{403, 404, 503} {
			t.Run(fmt.Sprintf("%s/%d", endpoint, code), func(t *testing.T) {
				client := releaseClient(t, func(req *http.Request) *http.Response {
					if strings.HasSuffix(req.URL.Path, endpoint) {
						return response(code, `{"message":"unavailable"}`)
					}
					return releaseCIResponse(req)
				})
				ci, _, err := client.inspectReleaseCI(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "main", "abc123")
				switch endpoint {
				case "/branches/main", "/rules/branches/main":
					assert.Equal(t, "unavailable", ci.Policy.State)
				case "/actions/runs":
					assert.Equal(t, "unavailable", ci.Workflows.State)
				default:
					require.Error(t, err)
				}
			})
		}
	}
}

func TestReleaseCIIgnoresChecksFromPreviousAttempt(t *testing.T) {
	client := releaseClient(t, func(req *http.Request) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/check-runs") {
			return response(200, `{"total_count":2,"check_runs":[
    {"id":100,"name":"native","head_sha":"abc123","status":"completed","conclusion":"failure","app":{"id":1,"slug":"github-actions"},"check_suite":{"id":10}},
    {"id":101,"name":"native","head_sha":"abc123","status":"completed","conclusion":"success","app":{"id":1,"slug":"github-actions"},"check_suite":{"id":10}}]}`)
		}
		if strings.HasSuffix(req.URL.Path, "/runs/10/attempts/2/jobs") {
			return response(200, `{"total_count":1,"jobs":[{"check_run_url":"https://api.example.test/repos/acme/tool/check-runs/101","head_sha":"abc123"}]}`)
		}
		if strings.HasSuffix(req.URL.Path, "/runs/20/attempts/1/jobs") {
			return response(200, `{"total_count":0,"jobs":[]}`)
		}
		return releaseCIResponse(req)
	})
	_, checks, err := client.inspectReleaseCI(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "main", "abc123")
	require.NoError(t, err)
	require.Len(t, checks, 1)
	assert.Equal(t, "success", checks[0].Conclusion)
}

func TestReleaseCIRejectsIncompleteAndUnidentifiableEvidence(t *testing.T) {
	for _, tc := range []struct{ name, endpoint, body, signal string }{
		{"truncated checks", "/check-runs", `{"total_count":101,"check_runs":[]}`, "checks"},
		{"truncated runs", "/actions/runs", `{"total_count":101,"workflow_runs":[]}`, "workflows"},
		{"truncated statuses", "/status", `{"total_count":101,"statuses":[]}`, "checks"},
		{"truncated jobs", "/runs/10/attempts/2/jobs", `{"total_count":101,"jobs":[]}`, "checks"},
		{"missing check identity", "/check-runs", `{"total_count":1,"check_runs":[null]}`, "checks"},
		{"missing check suite", "/check-runs", `{"total_count":1,"check_runs":[{"id":101,"name":"native","head_sha":"abc123","app":{"id":1,"slug":"github-actions"}}]}`, "checks"},
		{"missing workflow identity", "/actions/runs", `{"total_count":1,"workflow_runs":[{"head_sha":"abc123","head_branch":"main"}]}`, "workflows"},
		{"missing job identity", "/runs/10/attempts/2/jobs", `{"total_count":1,"jobs":[{"head_sha":"abc123"}]}`, "checks"},
		{"invalid job URL", "/runs/10/attempts/2/jobs", `{"total_count":1,"jobs":[{"head_sha":"abc123","check_run_url":"%"}]}`, "checks"},
		{"wrong job commit", "/runs/10/attempts/2/jobs", `{"total_count":1,"jobs":[{"head_sha":"other","check_run_url":"https://api.example.test/check-runs/101"}]}`, "checks"},
		{"missing protection state", "/branches/main", `{}`, "policy"},
		{"unsupported workflow rule", "/rules/branches/main", `[{"type":"workflows"}]`, "policy"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := releaseClient(t, func(req *http.Request) *http.Response {
				if strings.HasSuffix(req.URL.Path, tc.endpoint) {
					return response(200, tc.body)
				}
				return releaseCIResponse(req)
			})
			ci, _, err := client.inspectReleaseCI(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "main", "abc123")
			switch tc.signal {
			case "checks":
				require.Error(t, err)
			case "workflows":
				assert.Equal(t, "unavailable", ci.Workflows.State)
				assert.NotEmpty(t, ci.Workflows.Message)
			case "policy":
				assert.Equal(t, "unavailable", ci.Policy.State)
				assert.NotEmpty(t, ci.Policy.Message)
			}
		})
	}
}

func TestReleaseCIProtectsPolicyAndAttemptReads(t *testing.T) {
	for _, endpoint := range []string{"/protection", "/runs/10/attempts/2/jobs"} {
		client := releaseClient(t, func(req *http.Request) *http.Response {
			if req.URL.Path == "/repos/acme/tool/branches/main" {
				return response(200, `{"protected":true}`)
			}
			if strings.HasSuffix(req.URL.Path, endpoint) {
				return response(403, `{}`)
			}
			if strings.HasSuffix(req.URL.Path, "/protection") {
				return response(200, `{"required_status_checks":{"contexts":["Arch"],"checks":[{"context":"Arch","app_id":1}]}}`)
			}
			return releaseCIResponse(req)
		})
		ci, _, err := client.inspectReleaseCI(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "main", "abc123")
		if endpoint == "/protection" {
			assert.Equal(t, "unavailable", ci.Policy.State)
		} else {
			require.Error(t, err)
		}
	}
}

func TestReleaseCIStatusStatesAndProtectionWithoutChecks(t *testing.T) {
	client := releaseClient(t, func(req *http.Request) *http.Response {
		if req.URL.Path == "/repos/acme/tool/branches/main" {
			return response(200, `{"protected":true}`)
		}
		if strings.HasSuffix(req.URL.Path, "/protection") {
			return response(200, `{"required_status_checks":null}`)
		}
		if strings.HasSuffix(req.URL.Path, "/status") {
			return response(200, `{"total_count":3,"statuses":[{"context":"test","state":"pending"},{"context":"lint","state":"failure"},{"context":"build","state":"error"}]}`)
		}
		return releaseCIResponse(req)
	})
	ci, checks, err := client.inspectReleaseCI(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "main", "abc123")
	require.NoError(t, err)
	assert.Equal(t, "available", ci.Policy.State)
	require.Len(t, checks, 7)
	assert.Equal(t, "pending", checks[4].Status)
	assert.Empty(t, checks[4].Conclusion)
	assert.Equal(t, "failure", checks[5].Conclusion)
	assert.Equal(t, "error", checks[6].Conclusion)
}

func TestReleaseCIPaginationIsUnavailable(t *testing.T) {
	client := releaseClient(t, func(req *http.Request) *http.Response {
		resp := releaseCIResponse(req)
		resp.Header.Set("Link", `<https://api.example.test/next>; rel="next"`)
		return resp
	})
	ci, _, err := client.inspectReleaseCI(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "main", "abc123")
	require.ErrorContains(t, err, "bound")
	assert.Equal(t, "unavailable", ci.Policy.State)
	assert.Equal(t, "unavailable", ci.Workflows.State)
}

func TestReleaseCIMissingInventoryIsUnavailable(t *testing.T) {
	for _, endpoint := range []string{"/actions/runs", "/check-runs", "/status", "/runs/10/attempts/2/jobs"} {
		t.Run(endpoint, func(t *testing.T) {
			client := releaseClient(t, func(req *http.Request) *http.Response {
				if strings.HasSuffix(req.URL.Path, endpoint) {
					return response(200, `{}`)
				}
				return releaseCIResponse(req)
			})
			ci, _, err := client.inspectReleaseCI(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "main", "abc123")
			if endpoint == "/actions/runs" {
				assert.Equal(t, "unavailable", ci.Workflows.State)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestReleaseCIRequiresCheckEvidenceForSelectedJobs(t *testing.T) {
	client := releaseClient(t, func(req *http.Request) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/check-runs") {
			return response(200, `{"total_count":0,"check_runs":[]}`)
		}
		return releaseCIResponse(req)
	})
	_, _, err := client.inspectReleaseCI(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "main", "abc123")
	require.ErrorContains(t, err, "selected workflow jobs")
}
