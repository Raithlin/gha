package model

// Repository represents a GitHub repository.
type Repository struct {
	ID          int64
	Name        string
	FullName    string
	Owner       string
	Private     bool
	HTMLURL     string
	Description string
	Fork        bool
	URL         string
}

// PullRequest represents a GitHub pull request.
type PullRequest struct {
	ID          int64
	Number      int
	Title       string
	Body        string
	State       string // open, closed, merged
	User        string // login
	CreatedAt   string // time.Time in ISO format
	UpdatedAt   string // time.Time in ISO format
	ClosedAt    string // time.Time in ISO format
	MergedAt    string // time.Time in ISO format
	Head        string // ref
	Base        string // ref
	Mergeable   *bool
	MergeableState string
	Comments    int
	ReviewComments int
	Commits     int
	Additions   int
	Deletions   int
	ChangedFiles int
}

// Issue represents a GitHub issue.
type Issue struct {
	ID          int64
	Number      int
	Title       string
	Body        string
	State       string // open, closed
	User        string // login
	CreatedAt   string
	UpdatedAt   string
	ClosedAt    string
	Labels      []string
	Assignee    string
}

// Comment represents a comment on an issue or pull request.
type Comment struct {
	ID        int64
	Body      string
	User      string // login
	CreatedAt string
	UpdatedAt string
}

// Review represents a pull request review.
type Review struct {
	ID         int64
	User       string // login
	Body       string
	State      string // approved, changes_requested, commented, dismissed
	CommitID   string
	SubmittedAt string
}