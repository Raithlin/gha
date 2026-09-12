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

// GitHubClient implements the GitHubProvider interface.
type GitHubClient struct {
	HTTPClient *http.Client
	BaseURL    *url.URL
	Token      string
}

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
	req.Header.Set("Accept", "application/vnd.github.text+json")
	if c.Token != "" {
		req.Header.Set("Authorization", "token "+c.Token)
	}
	req.Header.Set("User-Agent", "GHA-GitHub-Assistant")

	return req, nil
}

// Helper function to decode JSON response into a struct.
func (c *GitHubClient) decodeResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()

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
	path := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews", owner, repo, number)
	req, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list reviews for pull request %s/%s#%d: %w", owner, repo, number, err)
	}

	var reviews []*model.Review
	if err := c.decodeResponse(resp, &reviews); err != nil {
		return nil, err
	}

	return reviews, nil
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
