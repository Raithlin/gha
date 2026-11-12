package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/raithlin/gha/internal/release"
	"github.com/raithlin/gha/pkg/model"
)

// ReadStatus inspects one tag, its matching workflow attempt, and hosted release.
func (c *GitHubClient) ReadStatus(ctx context.Context, repository model.RepositoryRef, tag, workflow string) (*release.Status, error) {
	result := &release.Status{Tag: tag, Repository: repository, State: "undiscovered", TagState: "unavailable",
		Workflow: release.StatusWorkflow{State: "undiscovered", Path: workflow}, HostedRelease: release.HostedRelease{State: "absent"}}
	base := fmt.Sprintf("repos/%s/%s", repository.Owner, repository.Name)
	commit, state, message := c.releaseTagCommit(ctx, base, tag)
	result.TagState, result.Commit = state, commit
	if state == "unavailable" {
		result.State, result.Workflow.State, result.Workflow.Message = "unavailable", "unavailable", message
		result.HostedRelease.State = "unavailable"
		return result, nil
	}
	if state == "absent" {
		return result, nil
	}
	result.Workflow = c.releaseStatusWorkflow(ctx, base, tag, commit, workflow)
	result.HostedRelease = c.hostedRelease(ctx, base, tag)
	result.State = releaseStatusState(result.Workflow.State, result.HostedRelease.State)
	return result, nil
}

func (c *GitHubClient) releaseTagCommit(ctx context.Context, base, tag string) (string, string, string) {
	var ref struct {
		Object struct{ Type, SHA string } `json:"object"`
	}
	found, err := c.releaseStatusGet(ctx, base+"/git/ref/tags/"+url.PathEscape(tag), &ref)
	if err != nil {
		return "", "unavailable", err.Error()
	}
	if !found {
		return "", "absent", ""
	}
	object := ref.Object
	for depth := 0; object.Type == "tag" && depth < 4; depth++ {
		var annotated struct {
			Object struct{ Type, SHA string } `json:"object"`
		}
		found, err = c.releaseStatusGet(ctx, base+"/git/tags/"+url.PathEscape(object.SHA), &annotated)
		if err != nil || !found {
			return "", "unavailable", "cannot resolve annotated tag"
		}
		object = annotated.Object
	}
	if object.Type != "commit" || object.SHA == "" {
		return "", "unavailable", "tag does not resolve to a commit"
	}
	return object.SHA, "available", ""
}

type statusRun struct {
	ID         int64  `json:"id"`
	WorkflowID int64  `json:"workflow_id"`
	RunAttempt int    `json:"run_attempt"`
	Path       string `json:"path"`
	HeadBranch string `json:"head_branch"`
	HeadSHA    string `json:"head_sha"`
	Event      string `json:"event"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	URL        string `json:"html_url"`
}

func (c *GitHubClient) releaseStatusWorkflow(ctx context.Context, base, tag, commit, selectedPath string) release.StatusWorkflow {
	result := release.StatusWorkflow{State: "undiscovered", Path: selectedPath}
	query := url.Values{"event": {"push"}, "head_sha": {commit}, "per_page": {"100"}}
	var raw struct {
		TotalCount int         `json:"total_count"`
		Runs       []statusRun `json:"workflow_runs"`
	}
	found, err := c.releaseStatusGet(ctx, base+"/actions/runs?"+query.Encode(), &raw)
	if err != nil {
		result.State, result.Message = "unavailable", err.Error()
		return result
	}
	if !found {
		result.State, result.Message = "unavailable", "workflow run inventory is unavailable"
		return result
	}
	result.Truncated = raw.TotalCount > len(raw.Runs)
	chosen, state := selectStatusRun(raw.Runs, tag, commit, selectedPath)
	if state == "undiscovered" {
		if result.Truncated {
			result.State, result.Message = "unavailable", "matching run may be outside bounded results"
		}
		return result
	}
	if state == "ambiguous" {
		result.State, result.Message = state, "multiple workflows match; pass --workflow"
		return result
	}
	result.Path, result.ID, result.RunID, result.Attempt = chosen.Path, chosen.WorkflowID, chosen.ID, chosen.RunAttempt
	result.Status, result.Conclusion, result.URL = chosen.Status, chosen.Conclusion, chosen.URL
	switch {
	case chosen.Status != "completed":
		result.State = "pending"
	case chosen.Conclusion == "success":
		result.State = "succeeded"
	case chosen.Conclusion == "cancelled":
		result.State = "cancelled"
	default:
		result.State = "failed"
	}
	return result
}

func selectStatusRun(runs []statusRun, tag, commit, path string) (statusRun, string) {
	var chosen statusRun
	for _, run := range runs {
		if run.HeadSHA != commit || run.HeadBranch != tag || run.Event != "push" || path != "" && run.Path != path {
			continue
		}
		if chosen.ID != 0 && path == "" && run.WorkflowID != chosen.WorkflowID {
			return statusRun{}, "ambiguous"
		}
		if run.ID > chosen.ID || run.ID == chosen.ID && run.RunAttempt > chosen.RunAttempt {
			chosen = run
		}
	}
	if chosen.ID == 0 {
		return chosen, "undiscovered"
	}
	return chosen, "available"
}

func (c *GitHubClient) hostedRelease(ctx context.Context, base, tag string) release.HostedRelease {
	result := release.HostedRelease{State: "absent"}
	var value struct {
		TagName    string `json:"tag_name"`
		URL        string `json:"html_url"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	found, err := c.releaseStatusGet(ctx, base+"/releases/tags/"+url.PathEscape(tag), &value)
	if err != nil {
		result.State, result.Message = "unavailable", err.Error()
		return result
	}
	if !found {
		return result
	}
	if value.TagName != tag {
		result.State, result.Message = "unavailable", "hosted release tag does not match"
		return result
	}
	result.URL, result.Draft, result.Prerelease = value.URL, value.Draft, value.Prerelease
	result.State = "available"
	if value.Draft {
		result.State = "draft"
	}
	return result
}

func releaseStatusState(workflow, hosted string) string {
	switch workflow {
	case "pending", "failed", "cancelled", "undiscovered":
		return workflow
	case "succeeded":
		if hosted == "available" {
			return "published"
		}
		if hosted == "absent" || hosted == "draft" {
			return "succeeded"
		}
	}
	return "unavailable"
}

func (c *GitHubClient) releaseStatusGet(ctx context.Context, endpoint string, target any) (bool, error) {
	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == http.StatusNotFound {
		if err := resp.Body.Close(); err != nil {
			return false, err
		}
		return false, nil
	}
	if err := c.decodeResponse(resp, target); err != nil {
		return false, fmt.Errorf("read %s: %w", strings.Split(endpoint, "?")[0], err)
	}
	return true, nil
}

var _ release.StatusProvider = (*GitHubClient)(nil)
