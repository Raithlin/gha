package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"

	"github.com/raithlin/gha/internal/release"
	"github.com/raithlin/gha/pkg/model"
)

// Release inventories are bounded; an incomplete response is unavailable, never ready.
const releaseCILimit = 100

func (c *GitHubClient) releaseGet(ctx context.Context, endpoint string, result any) error {
	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	if resp.Header.Get("Link") != "" { // All release lists request the maximum page size.
		if err := resp.Body.Close(); err != nil {
			return fmt.Errorf("close incomplete inventory: %w", err)
		}
		return fmt.Errorf("release CI inventory exceeds the %d item bound", releaseCILimit)
	}
	return c.decodeResponse(resp, result)
}

func (c *GitHubClient) inspectReleaseCI(ctx context.Context, repo model.RepositoryRef, branch, commit string) (release.CIState, []*model.CheckRun, error) {
	base := fmt.Sprintf("repos/%s/%s", repo.Owner, repo.Name)
	ci := release.CIState{Policy: release.CISignal{State: "available"}, Workflows: release.CISignal{State: "available"}, RequiredChecks: []release.RequiredCheck{}, Runs: []release.WorkflowRun{}}
	required, err := c.releaseRequiredChecks(ctx, base, branch)
	ci.RequiredChecks = required
	if err != nil {
		ci.Policy = release.CISignal{State: "unavailable", Message: err.Error()}
	}
	runs, err := c.releaseWorkflowRuns(ctx, base, branch, commit)
	ci.Runs = runs
	if err != nil {
		ci.Workflows = release.CISignal{State: "unavailable", Message: err.Error()}
	}
	checks, err := c.releaseChecks(ctx, base, commit, runs)
	if err != nil {
		return ci, checks, err
	}
	statuses, err := c.releaseStatuses(ctx, base, commit)
	return ci, append(checks, statuses...), err
}

type releaseBranchProtection struct {
	Protected  *bool `json:"protected"`
	Protection *struct {
		Enabled *bool `json:"enabled"`
	} `json:"protection"`
}

func (p releaseBranchProtection) classicEnabled() bool {
	// The overall protected flag includes rulesets. Only an explicit classic state
	// can override it; missing metadata must still require a verified policy read.
	if p.Protection != nil && p.Protection.Enabled != nil {
		return *p.Protection.Enabled
	}
	return p.Protected != nil && *p.Protected
}

func (c *GitHubClient) releaseRequiredChecks(ctx context.Context, base, branch string) ([]release.RequiredCheck, error) {
	var metadata releaseBranchProtection
	if err := c.releaseGet(ctx, base+"/branches/"+url.PathEscape(branch), &metadata); err != nil {
		return nil, fmt.Errorf("read branch protection state: %w", err)
	}
	if metadata.Protected == nil {
		return nil, fmt.Errorf("branch protection state was not returned")
	}
	required := []release.RequiredCheck{}
	if metadata.classicEnabled() {
		var protection struct {
			Required *struct {
				Contexts []string                `json:"contexts"`
				Checks   []release.RequiredCheck `json:"checks"`
			} `json:"required_status_checks"`
		}
		if err := c.releaseGet(ctx, base+"/branches/"+url.PathEscape(branch)+"/protection", &protection); err != nil {
			return required, fmt.Errorf("read required branch checks: %w", err)
		}
		if protection.Required != nil {
			required = append(required, protection.Required.Checks...)
			for _, context := range protection.Required.Contexts {
				found := false
				for _, check := range required {
					if check.Context == context {
						found = true
					}
				}
				if !found {
					required = append(required, release.RequiredCheck{Context: context})
				}
			}
		}
	}
	var rules []struct {
		Type       string `json:"type"`
		Parameters struct {
			Checks []struct {
				Context       string `json:"context"`
				IntegrationID int64  `json:"integration_id"`
			} `json:"required_status_checks"`
		} `json:"parameters"`
	}
	if err := c.releaseGet(ctx, base+"/rules/branches/"+url.PathEscape(branch)+"?per_page=100", &rules); err != nil {
		return required, fmt.Errorf("read active branch rules: %w", err)
	}
	if len(rules) >= releaseCILimit {
		return required, fmt.Errorf("branch rules inventory reached the %d item bound", releaseCILimit)
	}
	for _, rule := range rules {
		if rule.Type == "workflows" {
			return required, fmt.Errorf("required-workflow rules cannot be evaluated")
		}
		for _, check := range rule.Parameters.Checks {
			required = append(required, release.RequiredCheck{Context: check.Context, AppID: check.IntegrationID})
		}
	}
	return required, nil
}

func (c *GitHubClient) releaseWorkflowRuns(ctx context.Context, base, branch, commit string) ([]release.WorkflowRun, error) {
	var result struct {
		Total *int                  `json:"total_count"`
		Runs  []release.WorkflowRun `json:"workflow_runs"`
	}
	query := url.Values{"head_sha": {commit}, "per_page": {"100"}}
	if err := c.releaseGet(ctx, base+"/actions/runs?"+query.Encode(), &result); err != nil {
		return nil, err
	}
	if result.Total == nil || result.Runs == nil {
		return nil, fmt.Errorf("workflow run inventory is unavailable")
	}
	if *result.Total > len(result.Runs) {
		return nil, fmt.Errorf("workflow run inventory is truncated")
	}
	selected := map[int64]release.WorkflowRun{}
	for _, run := range result.Runs {
		if !relevantReleaseRun(run, branch, commit) {
			continue
		}
		if run.ID == 0 || run.WorkflowID == 0 || run.CheckSuiteID == 0 || run.RunAttempt < 1 {
			return nil, fmt.Errorf("workflow identity or attempt is unavailable")
		}
		old := selected[run.WorkflowID]
		if run.ID > old.ID || (run.ID == old.ID && run.RunAttempt > old.RunAttempt) {
			selected[run.WorkflowID] = run
		}
	}
	runs := make([]release.WorkflowRun, 0, len(selected))
	for _, run := range selected {
		runs = append(runs, run)
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].WorkflowID < runs[j].WorkflowID })
	return runs, nil
}

func (c *GitHubClient) releaseChecks(ctx context.Context, base, commit string, runs []release.WorkflowRun) ([]*model.CheckRun, error) {
	var result struct {
		Total  *int              `json:"total_count"`
		Checks []*model.CheckRun `json:"check_runs"`
	}
	endpoint := base + "/commits/" + url.PathEscape(commit) + "/check-runs?filter=latest&per_page=100"
	if err := c.releaseGet(ctx, endpoint, &result); err != nil {
		return nil, err
	}
	if result.Total == nil || result.Checks == nil {
		return nil, fmt.Errorf("check run inventory is unavailable")
	}
	if *result.Total > len(result.Checks) {
		return nil, fmt.Errorf("check run inventory is truncated")
	}
	jobIDs, err := c.releaseAttemptChecks(ctx, base, commit, runs)
	if err != nil {
		return nil, err
	}
	checks := []*model.CheckRun{}
	for _, check := range result.Checks {
		relevant, err := relevantReleaseCheck(check, commit, runs, jobIDs)
		if err != nil {
			return checks, err
		}
		if !relevant {
			continue
		}
		checks = append(checks, check)
		delete(jobIDs, check.ID)
	}
	if len(jobIDs) > 0 {
		return checks, fmt.Errorf("check results for selected workflow jobs are unavailable")
	}
	return checks, nil
}

func (c *GitHubClient) releaseStatuses(ctx context.Context, base, commit string) ([]*model.CheckRun, error) {
	var result struct {
		Total    *int `json:"total_count"`
		Statuses []struct {
			Context string `json:"context"`
			State   string `json:"state"`
		} `json:"statuses"`
	}
	if err := c.releaseGet(ctx, base+"/commits/"+url.PathEscape(commit)+"/status?per_page=100", &result); err != nil {
		return nil, err
	}
	if result.Total == nil || result.Statuses == nil {
		return nil, fmt.Errorf("commit status inventory is unavailable")
	}
	if *result.Total > len(result.Statuses) {
		return nil, fmt.Errorf("commit status inventory is truncated")
	}
	checks := []*model.CheckRun{}
	for _, status := range result.Statuses {
		check := &model.CheckRun{Name: status.Context, HeadSHA: commit, Status: "completed", Conclusion: status.State}
		if status.State == "pending" {
			check.Status = "pending"
			check.Conclusion = ""
		}
		checks = append(checks, check)
	}
	return checks, nil
}

// An attempt's jobs identify its check runs, so older attempts cannot block or validate a release.
func (c *GitHubClient) releaseAttemptChecks(ctx context.Context, base, commit string, runs []release.WorkflowRun) (map[int64]bool, error) {
	ids := map[int64]bool{}
	for _, run := range runs {
		var result struct {
			Total *int `json:"total_count"`
			Jobs  []struct {
				HeadSHA     string `json:"head_sha"`
				CheckRunURL string `json:"check_run_url"`
			} `json:"jobs"`
		}
		endpoint := fmt.Sprintf("%s/actions/runs/%d/attempts/%d/jobs?per_page=100", base, run.ID, run.RunAttempt)
		if err := c.releaseGet(ctx, endpoint, &result); err != nil {
			return nil, fmt.Errorf("read workflow %s attempt %d: %w", run.Name, run.RunAttempt, err)
		}
		if result.Total == nil || result.Jobs == nil {
			return nil, fmt.Errorf("workflow attempt job inventory is unavailable")
		}
		if *result.Total > len(result.Jobs) {
			return nil, fmt.Errorf("workflow attempt job inventory is truncated")
		}
		for _, job := range result.Jobs {
			parsed, err := url.Parse(job.CheckRunURL)
			if err != nil {
				return nil, fmt.Errorf("invalid job check URL: %w", err)
			}
			id, err := strconv.ParseInt(path.Base(parsed.Path), 10, 64)
			if err != nil || id <= 0 || job.HeadSHA != commit {
				return nil, fmt.Errorf("workflow job identity is unavailable for release commit")
			}
			ids[id] = true
		}
	}
	return ids, nil
}

func relevantReleaseRun(run release.WorkflowRun, branch, commit string) bool {
	return run.HeadSHA == commit && run.HeadBranch == branch && run.Event != "pull_request" && run.Event != "pull_request_target"
}

func relevantReleaseCheck(check *model.CheckRun, commit string, runs []release.WorkflowRun, jobIDs map[int64]bool) (bool, error) {
	if check == nil || check.HeadSHA == "" || check.App == nil {
		return false, fmt.Errorf("check identity is unavailable")
	}
	if check.HeadSHA != commit {
		return false, nil
	}
	if check.App.Slug != "github-actions" {
		return true, nil
	}
	if check.CheckSuite == nil {
		return false, fmt.Errorf("workflow check suite is unavailable")
	}
	for _, run := range runs {
		if check.CheckSuite.ID == run.CheckSuiteID && jobIDs[check.ID] {
			return true, nil
		}
	}
	return false, nil
}
