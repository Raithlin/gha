// Package github implements the GitHub code host provider.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/raithlin/gha/internal/interfaces"
	"github.com/raithlin/gha/pkg/model"
)

// GitHubAPIVersion pins requests to the REST API contract this client supports.
const GitHubAPIVersion = "2022-11-28"

const githubAcceptHeader = "application/vnd.github+json, application/vnd.github.text+json"

// GitHubClient implements the CodeHostProvider interface for GitHub.
//
//revive:disable-next-line:exported
type GitHubClient struct {
	HTTPClient *http.Client
	BaseURL    *url.URL
	Token      string
}

var _ interfaces.CodeHostProvider = (*GitHubClient)(nil)

// NewGitHubClient creates a new GitHub client with the given token.
func NewGitHubClient(token string) (*GitHubClient, error) {
	baseURL, err := url.Parse("https://api.github.com/")
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	return &GitHubClient{
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		BaseURL: baseURL,
		Token:   token,
	}, nil
}

// Helper function to create a new request with authentication.
func (c *GitHubClient) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	rel, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	u := c.BaseURL.ResolveReference(rel)
	var req *http.Request
	var err2 error

	if body != nil {
		// Enbody the body as JSON
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		req, err2 = http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(jsonData))
		if err2 != nil {
			return nil, err2
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err2 = http.NewRequestWithContext(ctx, method, u.String(), nil)
		if err2 != nil {
			return nil, err2
		}
	}

	// Set headers
	// Request GitHub's text representation as well as the raw Markdown body.
	// Terminal output uses body_text so HTML embedded in a PR description is not
	// emitted verbatim; structured output retains body for API consumers.
	req.Header.Set("Accept", githubAcceptHeader)
	req.Header.Set("X-GitHub-Api-Version", GitHubAPIVersion)
	if c.Token != "" {
		req.Header.Set("Authorization", "token "+c.Token)
	}
	req.Header.Set("User-Agent", "GHA-GitHub-Assistant")

	return req, nil
}

// Helper function to decode JSON response into a struct.
func (c *GitHubClient) decodeResponse(resp *http.Response, v interface{}) (returnErr error) {
	defer func() {
		if err := resp.Body.Close(); err != nil && returnErr == nil {
			returnErr = fmt.Errorf("close response body: %w", err)
		}
	}()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// GetAuthenticatedUser returns the user associated with the current token.
func (c *GitHubClient) GetAuthenticatedUser(ctx context.Context) (*model.User, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "user", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated user: %w", err)
	}

	var user model.User
	if err := c.decodeResponse(resp, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// ListRepositories returns a list of repositories for the authenticated user.
func (c *GitHubClient) ListRepositories(ctx context.Context) ([]*model.Repository, error) {
	req, err := c.newRequest(ctx, "GET", "user/repos", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	var repos []*model.Repository
	if err := c.decodeResponse(resp, &repos); err != nil {
		return nil, err
	}

	return repos, nil
}

// GetRepository returns a single repository by owner and name.
func (c *GitHubClient) GetRepository(ctx context.Context, owner, repo string) (*model.Repository, error) {
	path := fmt.Sprintf("repos/%s/%s", owner, repo)
	req, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository %s/%s: %w", owner, repo, err)
	}

	var repository *model.Repository
	if err := c.decodeResponse(resp, &repository); err != nil {
		return nil, err
	}

	return repository, nil
}

// InspectBranchSafety returns the GitHub safety facts relevant to one branch.
// Endpoint failures stay attached to their individual signals so callers can
// still use the facts GitHub did return.
func (c *GitHubClient) InspectBranchSafety(ctx context.Context, repository model.RepositoryRef, branch string) (model.BranchSafety, error) {
	safety := model.BranchSafety{Provider: "github", CheckedAt: time.Now().UTC().Format(time.RFC3339)}
	repo, repoErr := c.GetRepository(ctx, repository.Owner, repository.Name)
	if repoErr != nil {
		safety.DefaultBranch = unavailableSignal(repoErr)
		safety.Permissions = unavailableSignal(repoErr)
	} else {
		if repo.DefaultBranch == "" {
			safety.DefaultBranch = model.ProviderSignal{State: "unavailable", Message: "GitHub did not report the default branch"}
		} else {
			isDefault := branch == repo.DefaultBranch
			safety.DefaultBranch = model.ProviderSignal{State: "available"}
			safety.DefaultBranchName = repo.DefaultBranch
			safety.IsDefault = &isDefault
		}
		if repo.Permissions == nil {
			safety.Permissions = model.ProviderSignal{State: "unavailable", Message: "GitHub did not report caller permissions"}
		} else {
			canPush := repo.Permissions.Push || repo.Permissions.Admin
			safety.Permissions = model.ProviderSignal{State: "available"}
			safety.CanPush = &canPush
		}
	}

	protected, err := c.getBranchProtection(ctx, repository, branch)
	if err != nil {
		safety.Protection = unavailableSignal(err)
	} else {
		safety.Protection = model.ProviderSignal{State: "available"}
		safety.Protected = &protected
	}

	prs, err := c.openBranchPullRequests(ctx, repository, branch)
	if err != nil {
		safety.Requests = unavailableSignal(err)
		safety.Merge = unavailableSignal(err)
		return safety, nil
	}
	safety.Requests = model.ProviderSignal{State: "available"}
	safety.OpenPullRequests = prs
	if len(prs) != 1 {
		safety.Merge = model.ProviderSignal{State: "not_applicable", Message: "mergeability is reported only when exactly one open pull request targets this branch"}
		return safety, nil
	}
	pr, err := c.GetPullRequest(ctx, repository.Owner, repository.Name, prs[0].Number)
	if err != nil {
		safety.Merge = unavailableSignal(err)
		return safety, nil
	}
	if pr.Mergeable == nil {
		safety.Merge = model.ProviderSignal{State: "unavailable", Message: "GitHub did not report mergeability"}
		return safety, nil
	}
	safety.Merge = model.ProviderSignal{State: "available"}
	safety.Mergeable = pr.Mergeable
	return safety, nil
}

func (c *GitHubClient) openBranchPullRequests(ctx context.Context, repository model.RepositoryRef, branch string) ([]*model.PullRequest, error) {
	prs := make([]*model.PullRequest, 0)
	for page := 1; ; page++ {
		results, err := c.ListPullRequests(ctx, repository.Owner, repository.Name, interfaces.ListPRsOptions{
			State: "open", Head: repository.Owner + ":" + branch, PerPage: 100, Page: page,
		})
		if err != nil {
			return nil, err
		}
		prs = append(prs, results...)
		if len(results) < 100 {
			return prs, nil
		}
	}
}

func (c *GitHubClient) getBranchProtection(ctx context.Context, repository model.RepositoryRef, branch string) (bool, error) {
	path := fmt.Sprintf("repos/%s/%s/branches/%s", repository.Owner, repository.Name, url.PathEscape(branch))
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return false, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("get branch %s/%s:%s: %w", repository.Owner, repository.Name, branch, err)
	}
	var response struct {
		Protected bool `json:"protected"`
	}
	if err := c.decodeResponse(resp, &response); err != nil {
		return false, err
	}
	return response.Protected, nil
}

func unavailableSignal(err error) model.ProviderSignal {
	return model.ProviderSignal{State: "unavailable", Message: err.Error()}
}

// ListPullRequests returns a list of pull requests for a repository.
func (c *GitHubClient) ListPullRequests(ctx context.Context, owner, repo string, opts interfaces.ListPRsOptions) ([]*model.PullRequest, error) {
	// Build query parameters
	q := url.Values{}
	if opts.State != "" {
		q.Set("state", opts.State)
	}
	if opts.Head != "" {
		q.Set("head", opts.Head)
	}
	if opts.Base != "" {
		q.Set("base", opts.Base)
	}
	if opts.Sort != "" {
		q.Set("sort", opts.Sort)
	}
	if opts.Direction != "" {
		q.Set("direction", opts.Direction)
	}
	if opts.Since != "" {
		q.Set("since", opts.Since)
	}
	if opts.PerPage > 0 {
		q.Set("per_page", fmt.Sprintf("%d", opts.PerPage))
	}
	if opts.Page > 0 {
		q.Set("page", fmt.Sprintf("%d", opts.Page))
	}

	path := fmt.Sprintf("repos/%s/%s/pulls?%s", owner, repo, q.Encode())
	req, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests for %s/%s: %w", owner, repo, err)
	}

	var prs []*model.PullRequest
	if err := c.decodeResponse(resp, &prs); err != nil {
		return nil, err
	}

	return prs, nil
}

// GetPullRequest returns a single pull request by number.
func (c *GitHubClient) GetPullRequest(ctx context.Context, owner, repo string, number int) (*model.PullRequest, error) {
	path := fmt.Sprintf("repos/%s/%s/pulls/%d", owner, repo, number)
	req, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request %s/%s#%d: %w", owner, repo, number, err)
	}

	var pr *model.PullRequest
	if err := c.decodeResponse(resp, &pr); err != nil {
		return nil, err
	}

	return pr, nil
}

// CreatePullRequest creates a new pull request.
func (c *GitHubClient) CreatePullRequest(ctx context.Context, owner, repo string, input *model.PullRequestInput) (*model.PullRequest, error) {
	path := fmt.Sprintf("repos/%s/%s/pulls", owner, repo)
	req, err := c.newRequest(ctx, http.MethodPost, path, input)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	var createdPR *model.PullRequest
	if err := c.decodeResponse(resp, &createdPR); err != nil {
		return nil, err
	}

	return createdPR, nil
}

// CompareBranches reports how far head has diverged from base without changing
// provider or local Git state.
func (c *GitHubClient) CompareBranches(ctx context.Context, owner, repo, base, head string) (*model.BranchComparison, error) {
	path := fmt.Sprintf("repos/%s/%s/compare/%s...%s", owner, repo, url.PathEscape(base), url.PathEscape(head))
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("compare branches: %w", err)
	}
	var comparison struct {
		Status   string `json:"status"`
		AheadBy  int    `json:"ahead_by"`
		BehindBy int    `json:"behind_by"`
	}
	if err := c.decodeResponse(resp, &comparison); err != nil {
		return nil, err
	}
	return &model.BranchComparison{State: comparison.Status, AheadBy: comparison.AheadBy, BehindBy: comparison.BehindBy}, nil
}

// UpdatePullRequest updates an existing pull request.
func (c *GitHubClient) UpdatePullRequest(ctx context.Context, owner, repo string, number int, input *model.PullRequestInput) (*model.PullRequest, error) {
	path := fmt.Sprintf("repos/%s/%s/pulls/%d", owner, repo, number)
	req, err := c.newRequest(ctx, http.MethodPatch, path, input)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update pull request: %w", err)
	}

	var updatedPR *model.PullRequest
	if err := c.decodeResponse(resp, &updatedPR); err != nil {
		return nil, err
	}

	return updatedPR, nil
}

// ListIssues returns a list of issues for a repository.
func (c *GitHubClient) ListIssues(ctx context.Context, owner, repo string, opts interfaces.ListIssuesOptions) ([]*model.Issue, error) {
	// Build query parameters
	q := url.Values{}
	if opts.State != "" {
		q.Set("state", opts.State)
	}
	if len(opts.Labels) > 0 {
		q.Set("labels", joinStrings(opts.Labels, ","))
	}
	if opts.Sort != "" {
		q.Set("sort", opts.Sort)
	}
	if opts.Direction != "" {
		q.Set("direction", opts.Direction)
	}
	if opts.Since != "" {
		q.Set("since", opts.Since)
	}
	if opts.PerPage > 0 {
		q.Set("per_page", fmt.Sprintf("%d", opts.PerPage))
	}
	if opts.Page > 0 {
		q.Set("page", fmt.Sprintf("%d", opts.Page))
	}
	if opts.Assignee != "" {
		q.Set("assignee", opts.Assignee)
	}

	path := fmt.Sprintf("repos/%s/%s/issues?%s", owner, repo, q.Encode())
	req, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list issues for %s/%s: %w", owner, repo, err)
	}

	var issues []*model.Issue
	if err := c.decodeResponse(resp, &issues); err != nil {
		return nil, err
	}

	return issues, nil
}

// GetIssue returns a single issue by number.
func (c *GitHubClient) GetIssue(ctx context.Context, owner, repo string, number int) (*model.Issue, error) {
	path := fmt.Sprintf("repos/%s/%s/issues/%d", owner, repo, number)
	req, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue %s/%s#%d: %w", owner, repo, number, err)
	}

	var issue *model.Issue
	if err := c.decodeResponse(resp, &issue); err != nil {
		return nil, err
	}

	return issue, nil
}

// AddComment adds a comment to an issue or pull request.
func (c *GitHubClient) AddComment(ctx context.Context, owner, repo string, number int, body string) (*model.Comment, error) {
	path := fmt.Sprintf("repos/%s/%s/issues/%d/comments", owner, repo, number)
	comment := struct {
		Body string `json:"body"`
	}{Body: body}
	req, err := c.newRequest(ctx, "POST", path, comment)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to add comment: %w", err)
	}

	var commentResp *model.Comment
	if err := c.decodeResponse(resp, &commentResp); err != nil {
		return nil, err
	}

	return commentResp, nil
}

// ListReviews returns a list of reviews for a pull request.
func (c *GitHubClient) ListReviews(ctx context.Context, owner, repo string, number int) ([]*model.Review, error) {
	const perPage = 100
	var reviews []*model.Review
	for page := 1; ; page++ {
		path := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews?per_page=%d&page=%d", owner, repo, number, perPage, page)
		req, err := c.newRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to list reviews for pull request %s/%s#%d: %w", owner, repo, number, err)
		}

		var reviewPage []*model.Review
		if err := c.decodeResponse(resp, &reviewPage); err != nil {
			return nil, err
		}
		reviews = append(reviews, reviewPage...)
		if len(reviewPage) < perPage {
			return reviews, nil
		}
	}
}

// ListCheckRuns returns the latest CI check runs for a commit SHA.
func (c *GitHubClient) ListCheckRuns(ctx context.Context, owner, repo, ref string) ([]*model.CheckRun, error) {
	path := fmt.Sprintf("repos/%s/%s/commits/%s/check-runs?filter=latest&per_page=100", owner, repo, ref)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list check runs for %s/%s@%s: %w", owner, repo, ref, err)
	}

	var result struct {
		CheckRuns []*model.CheckRun `json:"check_runs"`
	}
	if err := c.decodeResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.CheckRuns, nil
}

// ListReviewThreads returns every review thread for a pull request using
// GitHub's GraphQL API, which is the API that exposes thread resolution.
func (c *GitHubClient) ListReviewThreads(ctx context.Context, owner, repo string, number int) ([]*model.ReviewThread, error) {
	const query = `query($owner: String!, $repo: String!, $number: Int!, $cursor: String) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      reviewThreads(first: 100, after: $cursor) {
        nodes { isResolved }
        pageInfo { hasNextPage endCursor }
      }
    }
  }
}`

	threads := make([]*model.ReviewThread, 0)
	var cursor *string
	for {
		requestBody := struct {
			Query     string `json:"query"`
			Variables struct {
				Owner  string  `json:"owner"`
				Repo   string  `json:"repo"`
				Number int     `json:"number"`
				Cursor *string `json:"cursor"`
			} `json:"variables"`
		}{Query: query}
		requestBody.Variables.Owner = owner
		requestBody.Variables.Repo = repo
		requestBody.Variables.Number = number
		requestBody.Variables.Cursor = cursor

		req, err := c.newRequest(ctx, http.MethodPost, "graphql", requestBody)
		if err != nil {
			return nil, err
		}
		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to list review threads for pull request %s/%s#%d: %w", owner, repo, number, err)
		}

		var result struct {
			Data struct {
				Repository *struct {
					PullRequest *struct {
						ReviewThreads struct {
							Nodes []struct {
								IsResolved bool `json:"isResolved"`
							} `json:"nodes"`
							PageInfo struct {
								HasNextPage bool    `json:"hasNextPage"`
								EndCursor   *string `json:"endCursor"`
							} `json:"pageInfo"`
						} `json:"reviewThreads"`
					} `json:"pullRequest"`
				} `json:"repository"`
			} `json:"data"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		if err := c.decodeResponse(resp, &result); err != nil {
			return nil, err
		}
		if len(result.Errors) > 0 {
			return nil, fmt.Errorf("GitHub GraphQL error: %s", result.Errors[0].Message)
		}
		if result.Data.Repository == nil || result.Data.Repository.PullRequest == nil {
			return nil, fmt.Errorf("GitHub GraphQL response did not include pull request %s/%s#%d", owner, repo, number)
		}

		page := result.Data.Repository.PullRequest.ReviewThreads
		for _, node := range page.Nodes {
			threads = append(threads, &model.ReviewThread{IsResolved: node.IsResolved})
		}
		if !page.PageInfo.HasNextPage {
			return threads, nil
		}
		if page.PageInfo.EndCursor == nil || *page.PageInfo.EndCursor == "" {
			return nil, fmt.Errorf("GitHub GraphQL response has another review-thread page without a cursor")
		}
		cursor = page.PageInfo.EndCursor
	}
}

// SubmitReview submits a review for a pull request.
func (c *GitHubClient) SubmitReview(ctx context.Context, owner, repo string, number int, input *model.ReviewInput) (*model.Review, error) {
	path := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews", owner, repo, number)
	req, err := c.newRequest(ctx, http.MethodPost, path, input)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to submit review: %w", err)
	}

	var submittedReview *model.Review
	if err := c.decodeResponse(resp, &submittedReview); err != nil {
		return nil, err
	}

	return submittedReview, nil
}

// Helper function to join strings with a separator
func joinStrings(elems []string, sep string) string {
	if len(elems) == 0 {
		return ""
	}
	if len(elems) == 1 {
		return elems[0]
	}
	result := elems[0]
	for _, elem := range elems[1:] {
		result += sep + elem
	}
	return result
}
