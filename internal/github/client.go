package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

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
// If token is empty, it will try to read from the environment variable GHA_GITHUB_TOKEN.
func NewGitHubClient(token string) (*GitHubClient, error) {
	if token == "" {
		token = os.Getenv("GHA_GITHUB_TOKEN")
	}
	if token == "" {
		return nil, fmt.Errorf("GitHub token is required (set GHA_GITHUB_TOKEN environment variable)")
	}

	baseURL, err := url.Parse("https://api.github.com/")
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	return &GitHubClient{
		HTTPClient: &http.Client{},
		BaseURL:    baseURL,
		Token:      token,
	}, nil
}

// Helper function to create a new request with authentication.
func (c *GitHubClient) newRequest(ctx context.Context, method, path string) (*http.Request, error) {
	rel, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	u := c.BaseURL.ResolveReference(rel)
	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", "token "+c.Token)
	req.Header.Set("User-Agent", "GHA-GitHub-Assistant")

	return req, nil
}

// ListRepositories implements the GitHubProvider interface.
func (c *GitHubClient) ListRepositories(ctx context.Context) ([]*model.Repository, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetRepository implements the GitHubProvider interface.
func (c *GitHubClient) GetRepository(ctx context.Context, owner, repo string) (*model.Repository, error) {
	return nil, fmt.Errorf("not implemented")
}

// ListPullRequests implements the GitHubProvider interface.
func (c *GitHubClient) ListPullRequests(ctx context.Context, owner, repo string, opts interfaces.ListPRsOptions) ([]*model.PullRequest, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetPullRequest implements the GitHubProvider interface.
func (c *GitHubClient) GetPullRequest(ctx context.Context, owner, repo string, number int) (*model.PullRequest, error) {
	return nil, fmt.Errorf("not implemented")
}

// CreatePullRequest implements the GitHubProvider interface.
func (c *GitHubClient) CreatePullRequest(ctx context.Context, owner, repo string, pr *model.PullRequest) (*model.PullRequest, error) {
	return nil, fmt.Errorf("not implemented")
}

// UpdatePullRequest implements the GitHubProvider interface.
func (c *GitHubClient) UpdatePullRequest(ctx context.Context, owner, repo string, number int, pr *model.PullRequest) (*model.PullRequest, error) {
	return nil, fmt.Errorf("not implemented")
}

// ListIssues implements the GitHubProvider interface.
func (c *GitHubClient) ListIssues(ctx context.Context, owner, repo string, opts interfaces.ListIssuesOptions) ([]*model.Issue, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetIssue implements the GitHubProvider interface.
func (c *GitHubClient) GetIssue(ctx context.Context, owner, repo string, number int) (*model.Issue, error) {
	return nil, fmt.Errorf("not implemented")
}

// AddComment implements the GitHubProvider interface.
func (c *GitHubClient) AddComment(ctx context.Context, owner, repo string, number int, body string) (*model.Comment, error) {
	return nil, fmt.Errorf("not implemented")
}

// ListReviews implements the GitHubProvider interface.
func (c *GitHubClient) ListReviews(ctx context.Context, owner, repo string, number int) ([]*model.Review, error) {
	return nil, fmt.Errorf("not implemented")
}

// SubmitReview implements the GitHubProvider interface.
func (c *GitHubClient) SubmitReview(ctx context.Context, owner, repo string, number int, review *model.Review) (*model.Review, error) {
	return nil, fmt.Errorf("not implemented")
}