package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/raithlin/gha/internal/release"
	"github.com/raithlin/gha/pkg/model"
)

// Inspect supplies release permission, CI, and exact remote-tag facts.
func (c *GitHubClient) Inspect(ctx context.Context, repository model.RepositoryRef, commit, tag string) (release.ProviderState, error) {
	var state release.ProviderState
	repo, err := c.GetRepository(ctx, repository.Owner, repository.Name)
	if err != nil {
		return state, err
	}
	state.Repository = repo
	checks, err := c.ListCheckRuns(ctx, repository.Owner, repository.Name, commit)
	state.Checks, state.ChecksError = checks, err
	ref := fmt.Sprintf("repos/%s/%s/git/ref/tags/%s", repository.Owner, repository.Name, url.PathEscape(tag))
	req, err := c.newRequest(ctx, http.MethodGet, ref, nil)
	if err != nil {
		return state, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return state, err
	}
	if resp.StatusCode == http.StatusNotFound {
		if err := resp.Body.Close(); err != nil {
			return state, err
		}
		return state, nil
	}
	var result struct {
		Ref string `json:"ref"`
	}
	if err := c.decodeResponse(resp, &result); err != nil {
		return state, fmt.Errorf("inspect origin tag: %w", err)
	}
	state.RemoteTag = result.Ref
	return state, nil
}

// Observe reads the tag-triggered workflow once, preserving asynchronous state.
func (c *GitHubClient) Observe(ctx context.Context, repository model.RepositoryRef, workflowPath, tag, commit string) (release.WorkflowObservation, error) {
	query := url.Values{"event": {"push"}, "head_sha": {commit}, "per_page": {"30"}}
	endpoint := fmt.Sprintf("repos/%s/%s/actions/workflows/%s/runs?%s", repository.Owner, repository.Name, url.PathEscape(path.Base(workflowPath)), query.Encode())
	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return release.WorkflowObservation{}, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return release.WorkflowObservation{}, err
	}
	var result struct {
		Runs []struct {
			HeadBranch string `json:"head_branch"`
			Event      string `json:"event"`
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
			URL        string `json:"html_url"`
		} `json:"workflow_runs"`
	}
	if err := c.decodeResponse(resp, &result); err != nil {
		return release.WorkflowObservation{}, err
	}
	for _, run := range result.Runs {
		if run.HeadBranch != tag || run.Event != "push" {
			continue
		}
		state := "triggered"
		if run.Status == "completed" {
			if run.Conclusion == "success" {
				state = "completed"
			} else {
				state = "failed"
			}
		}
		return release.WorkflowObservation{State: state, URL: run.URL}, nil
	}
	return release.WorkflowObservation{State: "unavailable", Message: "tag push succeeded but no matching release workflow run is visible yet"}, nil
}

var _ release.Provider = (*GitHubClient)(nil)
