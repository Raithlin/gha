package interfaces

import (
	"context"

	"github.com/raithlin/gha/pkg/model"
)

// GitHubProvider defines the interface for GitHub API operations.
type GitHubProvider interface {
	// GetAuthenticatedUser returns the user associated with the current token.
	GetAuthenticatedUser(ctx context.Context) (*model.User, error)

	// ListRepositories returns a list of repositories for the authenticated user.
	ListRepositories(ctx context.Context) ([]*model.Repository, error)

	// GetRepository returns a single repository by owner and name.
	GetRepository(ctx context.Context, owner, repo string) (*model.Repository, error)

	// ListPullRequests returns a list of pull requests for a repository.
	ListPullRequests(ctx context.Context, owner, repo string, opts ListPRsOptions) ([]*model.PullRequest, error)

	// GetPullRequest returns a single pull request by number.
	GetPullRequest(ctx context.Context, owner, repo string, number int) (*model.PullRequest, error)

	// CreatePullRequest creates a new pull request.
	CreatePullRequest(ctx context.Context, owner, repo string, input *model.PullRequestInput) (*model.PullRequest, error)

	// UpdatePullRequest updates an existing pull request.
	UpdatePullRequest(ctx context.Context, owner, repo string, number int, input *model.PullRequestInput) (*model.PullRequest, error)

	// ListIssues returns a list of issues for a repository.
	ListIssues(ctx context.Context, owner, repo string, opts ListIssuesOptions) ([]*model.Issue, error)

	// GetIssue returns a single issue by number.
	GetIssue(ctx context.Context, owner, repo string, number int) (*model.Issue, error)

	// AddComment adds a comment to an issue or pull request.
	AddComment(ctx context.Context, owner, repo string, number int, body string) (*model.Comment, error)

	// ListReviews returns a list of reviews for a pull request.
	ListReviews(ctx context.Context, owner, repo string, number int) ([]*model.Review, error)

	// SubmitReview submits a review for a pull request.
	SubmitReview(ctx context.Context, owner, repo string, number int, input *model.ReviewInput) (*model.Review, error)
}

// ListPRsOptions contains optional parameters for listing pull requests.
type ListPRsOptions struct {
	State     string // open, closed, or all
	Head      string // filter by head user or branch
	Base      string // filter by base user or branch
	Sort      string // created, updated, popularity, long-running
	Direction string // asc, desc
	Since     string // ISO 8601 timestamp
	PerPage   int    // number of results per page (max 100)
	Page      int    // page number (1-indexed)
}

// ListIssuesOptions contains optional parameters for listing issues.
type ListIssuesOptions struct {
	State     string   // open, closed, or all
	Labels    []string // list of label names to filter by
	Sort      string   // created, updated, comments
	Direction string   // asc, desc
	Since     string   // ISO 8601 timestamp
	PerPage   int      // number of results per page (max 100)
	Page      int      // page number (1-indexed)
	Assignee  string   // GitHub username, or "*" for any assigned issue
}
