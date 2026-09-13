package model

// RepositoryRef identifies a repository independently of any provider.
type RepositoryRef struct {
	Owner string `json:"owner" yaml:"owner"`
	Name  string `json:"name" yaml:"name"`
}

// String returns the conventional owner/name representation.
func (r RepositoryRef) String() string {
	return r.Owner + "/" + r.Name
}

// User represents a GitHub user.
type User struct {
	Login string `json:"login" yaml:"login"`
	ID    int64  `json:"id" yaml:"id"`
	Type  string `json:"type" yaml:"type"`
}

// Repository represents a GitHub repository.
type Repository struct {
	ID          int64  `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	FullName    string `json:"full_name" yaml:"full_name"`
	Owner       User   `json:"owner" yaml:"owner"`
	Private     bool   `json:"private" yaml:"private"`
	HTMLURL     string `json:"html_url" yaml:"html_url"`
	Description string `json:"description" yaml:"description"`
	Fork        bool   `json:"fork" yaml:"fork"`
	URL         string `json:"url" yaml:"url"`
}

// BranchRef represents the branch details included in a pull request.
type BranchRef struct {
	Label string `json:"label" yaml:"label"`
	Ref   string `json:"ref" yaml:"ref"`
	SHA   string `json:"sha" yaml:"sha"`
}

// PullRequest represents a GitHub pull request.
type PullRequest struct {
	ID                 int64     `json:"id" yaml:"id"`
	Number             int       `json:"number" yaml:"number"`
	Title              string    `json:"title" yaml:"title"`
	Body               string    `json:"body" yaml:"body"`
	BodyText           string    `json:"body_text,omitempty" yaml:"body_text,omitempty"`
	State              string    `json:"state" yaml:"state"`
	User               User      `json:"user" yaml:"user"`
	CreatedAt          string    `json:"created_at" yaml:"created_at"`
	UpdatedAt          string    `json:"updated_at" yaml:"updated_at"`
	ClosedAt           string    `json:"closed_at" yaml:"closed_at"`
	MergedAt           string    `json:"merged_at" yaml:"merged_at"`
	Head               BranchRef `json:"head" yaml:"head"`
	Base               BranchRef `json:"base" yaml:"base"`
	Mergeable          *bool     `json:"mergeable" yaml:"mergeable"`
	MergeableState     string    `json:"mergeable_state" yaml:"mergeable_state"`
	Comments           int       `json:"comments" yaml:"comments"`
	ReviewComments     int       `json:"review_comments" yaml:"review_comments"`
	Commits            int       `json:"commits" yaml:"commits"`
	Additions          int       `json:"additions" yaml:"additions"`
	Deletions          int       `json:"deletions" yaml:"deletions"`
	ChangedFiles       int       `json:"changed_files" yaml:"changed_files"`
	RequestedReviewers []User    `json:"requested_reviewers" yaml:"requested_reviewers"`
}

// PullRequestInput contains the fields accepted when creating or updating a pull request.
type PullRequestInput struct {
	Title string `json:"title,omitempty" yaml:"title,omitempty"`
	Body  string `json:"body,omitempty" yaml:"body,omitempty"`
	Head  string `json:"head,omitempty" yaml:"head,omitempty"`
	Base  string `json:"base,omitempty" yaml:"base,omitempty"`
	State string `json:"state,omitempty" yaml:"state,omitempty"`
}

// PullRequestReference indicates that an issue is also a pull request.
type PullRequestReference struct {
	URL string `json:"url" yaml:"url"`
}

// Label represents a GitHub issue label.
type Label struct {
	Name string `json:"name" yaml:"name"`
}

// Issue represents a GitHub issue.
type Issue struct {
	ID          int64                 `json:"id" yaml:"id"`
	Number      int                   `json:"number" yaml:"number"`
	Title       string                `json:"title" yaml:"title"`
	Body        string                `json:"body" yaml:"body"`
	State       string                `json:"state" yaml:"state"`
	User        User                  `json:"user" yaml:"user"`
	CreatedAt   string                `json:"created_at" yaml:"created_at"`
	UpdatedAt   string                `json:"updated_at" yaml:"updated_at"`
	ClosedAt    string                `json:"closed_at" yaml:"closed_at"`
	Labels      []Label               `json:"labels" yaml:"labels"`
	Assignee    *User                 `json:"assignee" yaml:"assignee"`
	PullRequest *PullRequestReference `json:"pull_request" yaml:"pull_request"`
}

// Comment represents a comment on an issue or pull request.
type Comment struct {
	ID        int64  `json:"id" yaml:"id"`
	Body      string `json:"body" yaml:"body"`
	User      User   `json:"user" yaml:"user"`
	CreatedAt string `json:"created_at" yaml:"created_at"`
	UpdatedAt string `json:"updated_at" yaml:"updated_at"`
}

// Review represents a pull request review.
type Review struct {
	ID          int64  `json:"id" yaml:"id"`
	User        User   `json:"user" yaml:"user"`
	Body        string `json:"body" yaml:"body"`
	State       string `json:"state" yaml:"state"`
	CommitID    string `json:"commit_id" yaml:"commit_id"`
	SubmittedAt string `json:"submitted_at" yaml:"submitted_at"`
}

// CheckRun represents a GitHub check run associated with a commit.
type CheckRun struct {
	Name       string `json:"name" yaml:"name"`
	Status     string `json:"status" yaml:"status"`
	Conclusion string `json:"conclusion" yaml:"conclusion"`
}

// ReviewThread represents the resolved state of a pull request discussion.
type ReviewThread struct {
	IsResolved bool `json:"is_resolved" yaml:"is_resolved"`
}

// ReviewSummarySchemaVersion identifies the stable schema for review summaries.
const ReviewSummarySchemaVersion = "v1"

// ReviewSummary contains the decision-ready result of inspecting a pull request.
type ReviewSummary struct {
	SchemaVersion      string              `json:"schema_version" yaml:"schema_version"`
	PullRequest        *PullRequest        `json:"pull_request" yaml:"pull_request"`
	Reviews            []*Review           `json:"reviews" yaml:"reviews"`
	Readiness          ReviewReadiness     `json:"readiness" yaml:"readiness"`
	RiskSignals        []RiskSignal        `json:"risk_signals" yaml:"risk_signals"`
	RecommendedActions []RecommendedAction `json:"recommended_actions" yaml:"recommended_actions"`
}

// ReviewReadiness describes the available signals relevant to merging a pull request.
type ReviewReadiness struct {
	Mergeable          *bool  `json:"mergeable" yaml:"mergeable"`
	MergeableState     string `json:"mergeable_state" yaml:"mergeable_state"`
	CIStatus           string `json:"ci_status" yaml:"ci_status"`
	ReviewThreadsState string `json:"review_threads_state" yaml:"review_threads_state"`
	ApprovedBy         []User `json:"approved_by" yaml:"approved_by"`
	ChangesRequestedBy []User `json:"changes_requested_by" yaml:"changes_requested_by"`
	PendingReviewers   []User `json:"pending_reviewers" yaml:"pending_reviewers"`
}

// RiskSignal identifies a review-relevant concern and its severity.
type RiskSignal struct {
	Kind     string `json:"kind" yaml:"kind"`
	Severity string `json:"severity" yaml:"severity"`
	Detail   string `json:"detail" yaml:"detail"`
}

// RecommendedAction is a safe next step derived from the available review signals.
type RecommendedAction struct {
	Action    string `json:"action" yaml:"action"`
	Reason    string `json:"reason" yaml:"reason"`
	Reviewers []User `json:"reviewers,omitempty" yaml:"reviewers,omitempty"`
}

// ReviewInput contains the fields accepted when submitting a pull request review.
type ReviewInput struct {
	Body  string `json:"body,omitempty" yaml:"body,omitempty"`
	Event string `json:"event,omitempty" yaml:"event,omitempty"`
}
