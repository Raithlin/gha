// Package model defines the stable data contracts exposed by gha.
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

// User represents an account supplied by a code host.
type User struct {
	Login string `json:"login" yaml:"login"`
	ID    int64  `json:"id" yaml:"id"`
	Type  string `json:"type" yaml:"type"`
}

// Repository represents a GitHub repository.
type Repository struct {
	ID            int64                  `json:"id" yaml:"id"`
	Name          string                 `json:"name" yaml:"name"`
	FullName      string                 `json:"full_name" yaml:"full_name"`
	Owner         User                   `json:"owner" yaml:"owner"`
	Private       bool                   `json:"private" yaml:"private"`
	HTMLURL       string                 `json:"html_url" yaml:"html_url"`
	Description   string                 `json:"description" yaml:"description"`
	Fork          bool                   `json:"fork" yaml:"fork"`
	URL           string                 `json:"url" yaml:"url"`
	DefaultBranch string                 `json:"default_branch" yaml:"default_branch"`
	Permissions   *RepositoryPermissions `json:"permissions" yaml:"permissions"`
}

// RepositoryPermissions describes the permissions GitHub reports for the
// current caller. Providers that cannot supply this information leave it nil.
type RepositoryPermissions struct {
	Admin bool `json:"admin" yaml:"admin"`
	Push  bool `json:"push" yaml:"push"`
	Pull  bool `json:"pull" yaml:"pull"`
}

// BranchRef represents the branch details included in a pull request.
type BranchRef struct {
	Label string `json:"label" yaml:"label"`
	Ref   string `json:"ref" yaml:"ref"`
	SHA   string `json:"sha" yaml:"sha"`
}

// Branch represents a local or remote-tracking branch.
type Branch struct {
	Name              string `json:"name" yaml:"name"`
	SHA               string `json:"sha" yaml:"sha"`
	Current           bool   `json:"current,omitempty" yaml:"current,omitempty"`
	Upstream          string `json:"upstream,omitempty" yaml:"upstream,omitempty"`
	DivergenceState   string `json:"divergence_state" yaml:"divergence_state"`
	DivergenceMessage string `json:"divergence_message,omitempty" yaml:"divergence_message,omitempty"`
	Ahead             *int   `json:"ahead,omitempty" yaml:"ahead,omitempty"`
	Behind            *int   `json:"behind,omitempty" yaml:"behind,omitempty"`
}

// PullRequest represents GHA's provider-neutral pull request model.
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

// ReleaseNotesSchemaVersion identifies the stable schema for generated release notes.
const ReleaseNotesSchemaVersion = "v1"

// ReleaseListSchemaVersion identifies the stable schema for published-release listings.
const ReleaseListSchemaVersion = "v1"

// PullRequestListSchemaVersion identifies the stable schema for bounded pull request lists.
const PullRequestListSchemaVersion = "v1"

// PullRequestPreparationSchemaVersion identifies the stable preflight contract
// used before a pull request is created.
const PullRequestPreparationSchemaVersion = "v1"

// BranchInventorySchemaVersion identifies the stable schema for branch inventory.
const BranchInventorySchemaVersion = "v1"

// BranchInspectionSchemaVersion identifies the stable schema for inspecting a
// single branch and its provider safety signals.
const BranchInspectionSchemaVersion = "v1"

// BranchMutationSchemaVersion identifies the stable schema for branch writes.
const BranchMutationSchemaVersion = "v1"

// BranchPublicationSchemaVersion identifies the stable schema for guarded
// publication of an existing local branch.
const BranchPublicationSchemaVersion = "v1"

// BranchCleanupSchemaVersion identifies the stable schema for reviewed local
// branch cleanup candidates.
const BranchCleanupSchemaVersion = "v1"

// RepositoryAnalysisSchemaVersion identifies the stable schema for an offline
// local Git repository analysis.
const RepositoryAnalysisSchemaVersion = "v1"

// VersionInfoSchemaVersion identifies the stable schema for installed-build
// identification.
const VersionInfoSchemaVersion = "v1"

// AgentInstallationListSchemaVersion identifies the installed-agent inventory contract.
const AgentInstallationListSchemaVersion = "v1"

// AgentInstallationList inventories coding-agent harnesses configured by GHA.
type AgentInstallationList struct {
	SchemaVersion string              `json:"schema_version" yaml:"schema_version"`
	Agents        []AgentInstallation `json:"agents" yaml:"agents"`
}

// GuidanceUpdateSchemaVersion identifies the guidance refresh result contract.
const GuidanceUpdateSchemaVersion = "v1"

// GuidanceUpdate reports the published guidance refresh plan and outcome.
type GuidanceUpdate struct {
	SchemaVersion string                 `json:"schema_version" yaml:"schema_version"`
	LatestVersion string                 `json:"latest_version" yaml:"latest_version"`
	BinaryVersion string                 `json:"binary_version" yaml:"binary_version"`
	BinaryUpdated bool                   `json:"binary_updated" yaml:"binary_updated"`
	DryRun        bool                   `json:"dry_run" yaml:"dry_run"`
	SourceState   string                 `json:"source_state" yaml:"source_state"`
	SourceMessage string                 `json:"source_message,omitempty" yaml:"source_message,omitempty"`
	Targets       []GuidanceUpdateTarget `json:"targets" yaml:"targets"`
}

// GuidanceUpdateTarget is one recorded agent installation selected for refresh.
type GuidanceUpdateTarget struct {
	AgentID          string `json:"agent_id" yaml:"agent_id"`
	AgentName        string `json:"agent_name" yaml:"agent_name"`
	SkillPath        string `json:"skill_path" yaml:"skill_path"`
	InstructionsPath string `json:"instructions_path,omitempty" yaml:"instructions_path,omitempty"`
	State            string `json:"state" yaml:"state"`
}

// AgentInstallation records the managed destinations and their current file presence.
type AgentInstallation struct {
	ID               string `json:"id" yaml:"id"`
	Name             string `json:"name" yaml:"name"`
	SkillPath        string `json:"skill_path" yaml:"skill_path"`
	InstructionsPath string `json:"instructions_path" yaml:"instructions_path"`
	SkillState       string `json:"skill_state" yaml:"skill_state"`
	GuidanceState    string `json:"guidance_state" yaml:"guidance_state"`
}

// VersionInfo identifies the installed GHA build for support and automation.
type VersionInfo struct {
	SchemaVersion string `json:"schema_version" yaml:"schema_version"`
	Version       string `json:"version" yaml:"version"`
	Commit        string `json:"commit" yaml:"commit"`
	Date          string `json:"date" yaml:"date"`
}

// BranchInventory contains bounded local and origin branch views from Git.
type BranchInventory struct {
	SchemaVersion   string        `json:"schema_version" yaml:"schema_version"`
	Limit           int           `json:"limit" yaml:"limit"`
	Origin          string        `json:"origin,omitempty" yaml:"origin,omitempty"`
	OriginState     string        `json:"origin_state" yaml:"origin_state"`
	OriginRefresh   OriginRefresh `json:"origin_refresh" yaml:"origin_refresh"`
	Local           []*Branch     `json:"local" yaml:"local"`
	LocalTruncated  bool          `json:"local_truncated" yaml:"local_truncated"`
	OriginBranches  []*Branch     `json:"origin_branches" yaml:"origin_branches"`
	OriginTruncated bool          `json:"origin_truncated" yaml:"origin_truncated"`
}

// OriginRefresh reports whether the inventory refreshed remote-tracking refs
// during this invocation. A refresh is never implicit.
type OriginRefresh struct {
	State string `json:"state" yaml:"state"`
}

// ProviderSignal makes capability failures and unsupported provider data
// explicit instead of treating missing data as a negative result.
type ProviderSignal struct {
	State   string `json:"state" yaml:"state"`
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
}

// BranchSafety contains provider-enriched facts that are useful before a
// branch is changed or removed. A value is meaningful only when its paired
// signal has state "available".
type BranchSafety struct {
	Provider          string         `json:"provider,omitempty" yaml:"provider,omitempty"`
	CheckedAt         string         `json:"checked_at,omitempty" yaml:"checked_at,omitempty"`
	Requests          ProviderSignal `json:"requests" yaml:"requests"`
	OpenPullRequests  []*PullRequest `json:"open_pull_requests" yaml:"open_pull_requests"`
	Protection        ProviderSignal `json:"protection" yaml:"protection"`
	Protected         *bool          `json:"protected" yaml:"protected"`
	Permissions       ProviderSignal `json:"permissions" yaml:"permissions"`
	CanPush           *bool          `json:"can_push" yaml:"can_push"`
	DefaultBranch     ProviderSignal `json:"default_branch" yaml:"default_branch"`
	DefaultBranchName string         `json:"default_branch_name,omitempty" yaml:"default_branch_name,omitempty"`
	IsDefault         *bool          `json:"is_default" yaml:"is_default"`
	Merge             ProviderSignal `json:"merge" yaml:"merge"`
	Mergeable         *bool          `json:"mergeable" yaml:"mergeable"`
}

// BranchInspection combines the selected branch's local Git state with
// independently retrievable provider safety signals.
type BranchInspection struct {
	SchemaVersion string         `json:"schema_version" yaml:"schema_version"`
	Name          string         `json:"name" yaml:"name"`
	Repository    *RepositoryRef `json:"repository,omitempty" yaml:"repository,omitempty"`
	Origin        string         `json:"origin,omitempty" yaml:"origin,omitempty"`
	OriginState   string         `json:"origin_state" yaml:"origin_state"`
	Local         *Branch        `json:"local" yaml:"local"`
	OriginBranch  *Branch        `json:"origin_branch" yaml:"origin_branch"`
	Safety        BranchSafety   `json:"safety" yaml:"safety"`
}

// BranchMutation records the explicit local and origin effects of a branch
// write. A target state is one of completed, planned, or not_requested.
type BranchMutation struct {
	SchemaVersion string `json:"schema_version" yaml:"schema_version"`
	Operation     string `json:"operation" yaml:"operation"`
	Name          string `json:"name" yaml:"name"`
	NewName       string `json:"new_name,omitempty" yaml:"new_name,omitempty"`
	From          string `json:"from,omitempty" yaml:"from,omitempty"`
	CheckedOut    string `json:"checked_out,omitempty" yaml:"checked_out,omitempty"`
	DryRun        bool   `json:"dry_run" yaml:"dry_run"`
	Local         string `json:"local" yaml:"local"`
	Origin        string `json:"origin" yaml:"origin"`
}

// TagPublication reports the exact commit and local/origin tag effects.
type TagPublication struct {
	SchemaVersion string   `json:"schema_version" yaml:"schema_version"`
	Tag           string   `json:"tag" yaml:"tag"`
	Commit        string   `json:"commit" yaml:"commit"`
	DryRun        bool     `json:"dry_run" yaml:"dry_run"`
	Ready         bool     `json:"ready" yaml:"ready"`
	Blockers      []string `json:"blockers" yaml:"blockers"`
	Local         string   `json:"local" yaml:"local"`
	Origin        string   `json:"origin" yaml:"origin"`
}

// TagPublicationSchemaVersion identifies the stable tag publication result.
const TagPublicationSchemaVersion = "v1"

// BranchPublication records the preflight and result of publishing an existing
// local branch to origin. Local and origin branch data are read without a
// fetch; OriginState makes their freshness explicit.
type BranchPublication struct {
	SchemaVersion string         `json:"schema_version" yaml:"schema_version"`
	Repository    *RepositoryRef `json:"repository,omitempty" yaml:"repository,omitempty"`
	Name          string         `json:"name" yaml:"name"`
	Origin        string         `json:"origin,omitempty" yaml:"origin,omitempty"`
	OriginState   string         `json:"origin_state" yaml:"origin_state"`
	Target        string         `json:"target" yaml:"target"`
	Local         *Branch        `json:"local" yaml:"local"`
	OriginBranch  *Branch        `json:"origin_branch" yaml:"origin_branch"`
	Permissions   ProviderSignal `json:"permissions" yaml:"permissions"`
	CanPush       *bool          `json:"can_push" yaml:"can_push"`
	DryRun        bool           `json:"dry_run" yaml:"dry_run"`
	Publication   string         `json:"publication" yaml:"publication"`
}

// BranchCleanupCandidate records one bounded local branch review. Reason is
// either the rule that made it a candidate or the reason it was excluded.
type BranchCleanupCandidate struct {
	Name   string  `json:"name" yaml:"name"`
	Local  *Branch `json:"local" yaml:"local"`
	Reason string  `json:"reason" yaml:"reason"`
}

// BranchCleanup is a read-only review of local branches against a selected
// base. It never treats branch age or missing provider data as a cleanup fact.
type BranchCleanup struct {
	SchemaVersion string                    `json:"schema_version" yaml:"schema_version"`
	Rule          string                    `json:"rule" yaml:"rule"`
	Base          string                    `json:"base" yaml:"base"`
	Limit         int                       `json:"limit" yaml:"limit"`
	Truncated     bool                      `json:"truncated" yaml:"truncated"`
	Candidates    []*BranchCleanupCandidate `json:"candidates" yaml:"candidates"`
	Excluded      []*BranchCleanupCandidate `json:"excluded" yaml:"excluded"`
}

// AnalysisSignal makes a partial local analysis explicit. A value is useful
// only when State is available.
type AnalysisSignal struct {
	State   string `json:"state" yaml:"state"`
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
}

// AnalysisHead identifies the checked-out commit. Branch is empty when HEAD is
// detached or unborn.
type AnalysisHead struct {
	State   string `json:"state" yaml:"state"`
	Branch  string `json:"branch,omitempty" yaml:"branch,omitempty"`
	SHA     string `json:"sha,omitempty" yaml:"sha,omitempty"`
	Commits int    `json:"commits" yaml:"commits"`
}

// WorktreeChange is one path reported by Git's porcelain status. OriginalPath
// is populated for renames and copies.
type WorktreeChange struct {
	Path           string `json:"path" yaml:"path"`
	OriginalPath   string `json:"original_path,omitempty" yaml:"original_path,omitempty"`
	IndexStatus    string `json:"index_status" yaml:"index_status"`
	WorktreeStatus string `json:"worktree_status" yaml:"worktree_status"`
}

// WorktreeSummary describes uncommitted local work without inspecting ignored
// files. Counts always include entries omitted from Changes by Limit.
type WorktreeSummary struct {
	State            string           `json:"state" yaml:"state"`
	Staged           int              `json:"staged" yaml:"staged"`
	Unstaged         int              `json:"unstaged" yaml:"unstaged"`
	Untracked        int              `json:"untracked" yaml:"untracked"`
	Conflicted       int              `json:"conflicted" yaml:"conflicted"`
	Changes          []WorktreeChange `json:"changes" yaml:"changes"`
	ChangesTruncated bool             `json:"changes_truncated" yaml:"changes_truncated"`
}

// RepositoryStorage is Git's local object-database estimate. It deliberately
// excludes a working tree and any remote state.
type RepositoryStorage struct {
	LooseObjects int `json:"loose_objects" yaml:"loose_objects"`
	LooseKiB     int `json:"loose_kib" yaml:"loose_kib"`
	PackedKiB    int `json:"packed_kib" yaml:"packed_kib"`
}

// LocalCommit is a bounded history entry from the local HEAD only.
type LocalCommit struct {
	SHA        string `json:"sha" yaml:"sha"`
	Subject    string `json:"subject" yaml:"subject"`
	AuthoredAt string `json:"authored_at" yaml:"authored_at"`
}

// LargestFile is a tracked blob from HEAD. It does not describe uncommitted
// working-tree content.
type LargestFile struct {
	Path  string `json:"path" yaml:"path"`
	Bytes int64  `json:"bytes" yaml:"bytes"`
}

// RepositoryAnalysis combines local Git facts into one bounded, offline
// snapshot. RecentCommits and LargestFiles apply only to HEAD and are marked
// unavailable for an unborn repository.
type RepositoryAnalysis struct {
	SchemaVersion          string            `json:"schema_version" yaml:"schema_version"`
	AnalyzedAt             string            `json:"analyzed_at" yaml:"analyzed_at"`
	Path                   string            `json:"path" yaml:"path"`
	Limit                  int               `json:"limit" yaml:"limit"`
	Head                   AnalysisHead      `json:"head" yaml:"head"`
	Worktree               WorktreeSummary   `json:"worktree" yaml:"worktree"`
	Storage                RepositoryStorage `json:"storage" yaml:"storage"`
	RecentCommits          []LocalCommit     `json:"recent_commits" yaml:"recent_commits"`
	RecentCommitsTruncated bool              `json:"recent_commits_truncated" yaml:"recent_commits_truncated"`
	LargestFiles           []LargestFile     `json:"largest_files" yaml:"largest_files"`
	LargestFilesTruncated  bool              `json:"largest_files_truncated" yaml:"largest_files_truncated"`
	LargestFilesSignal     AnalysisSignal    `json:"largest_files_signal" yaml:"largest_files_signal"`
}

// ErrorSchemaVersion identifies the stable schema for structured command errors.
const ErrorSchemaVersion = "v1"

// CommandError describes a command failure for structured output formats.
type CommandError struct {
	SchemaVersion string `json:"schema_version" yaml:"schema_version"`
	Code          string `json:"code" yaml:"code"`
	Message       string `json:"message" yaml:"message"`
}

// ReleaseNotes contains the merged pull requests included in a release window.
// It is generated locally and never creates or publishes a GitHub release.
type ReleaseNotes struct {
	SchemaVersion string         `json:"schema_version" yaml:"schema_version"`
	Repository    RepositoryRef  `json:"repository" yaml:"repository"`
	Since         string         `json:"since" yaml:"since"`
	Limit         int            `json:"limit" yaml:"limit"`
	Truncated     bool           `json:"truncated" yaml:"truncated"`
	PullRequests  []*PullRequest `json:"pull_requests" yaml:"pull_requests"`
	Contributors  []User         `json:"contributors" yaml:"contributors"`
}

// Release represents a published provider release without provider-specific payloads.
// Draft releases are intentionally excluded from ReleaseList results.
type Release struct {
	ID              int64  `json:"id" yaml:"id"`
	TagName         string `json:"tag_name" yaml:"tag_name"`
	Name            string `json:"name" yaml:"name"`
	HTMLURL         string `json:"html_url" yaml:"html_url"`
	TargetCommitish string `json:"target_commitish" yaml:"target_commitish"`
	Draft           bool   `json:"draft" yaml:"draft"`
	Prerelease      bool   `json:"prerelease" yaml:"prerelease"`
	Author          User   `json:"author" yaml:"author"`
	CreatedAt       string `json:"created_at" yaml:"created_at"`
	PublishedAt     string `json:"published_at" yaml:"published_at"`
}

// ReleaseList contains a bounded listing of published releases. Truncated is
// true when more published releases were available than Limit permits.
type ReleaseList struct {
	SchemaVersion string        `json:"schema_version" yaml:"schema_version"`
	Repository    RepositoryRef `json:"repository" yaml:"repository"`
	Limit         int           `json:"limit" yaml:"limit"`
	Truncated     bool          `json:"truncated" yaml:"truncated"`
	Releases      []*Release    `json:"releases" yaml:"releases"`
}

// PullRequestList contains a bounded pull request query and makes omitted
// results explicit for automation clients.
type PullRequestList struct {
	SchemaVersion string         `json:"schema_version" yaml:"schema_version"`
	Repository    RepositoryRef  `json:"repository" yaml:"repository"`
	Limit         int            `json:"limit" yaml:"limit"`
	Truncated     bool           `json:"truncated" yaml:"truncated"`
	PullRequests  []*PullRequest `json:"pull_requests" yaml:"pull_requests"`
}

// BranchComparison describes the provider's comparison of a proposed pull
// request head against its base. State is available only when AheadBy and
// BehindBy came from the provider.
type BranchComparison struct {
	State    string `json:"state" yaml:"state"`
	Message  string `json:"message,omitempty" yaml:"message,omitempty"`
	AheadBy  int    `json:"ahead_by" yaml:"ahead_by"`
	BehindBy int    `json:"behind_by" yaml:"behind_by"`
}

// PullRequestPreparation is the decision-ready plan for creating one pull
// request. Creation is planned until the provider confirms the write.
type PullRequestPreparation struct {
	SchemaVersion        string              `json:"schema_version" yaml:"schema_version"`
	Repository           RepositoryRef       `json:"repository" yaml:"repository"`
	Title                string              `json:"title" yaml:"title"`
	Body                 string              `json:"body" yaml:"body"`
	Draft                PullRequestDraft    `json:"draft" yaml:"draft"`
	Head                 string              `json:"head" yaml:"head"`
	Base                 string              `json:"base" yaml:"base"`
	DryRun               bool                `json:"dry_run" yaml:"dry_run"`
	Creation             string              `json:"creation" yaml:"creation"`
	Comparison           BranchComparison    `json:"comparison" yaml:"comparison"`
	Permissions          ProviderSignal      `json:"permissions" yaml:"permissions"`
	CanPush              *bool               `json:"can_push" yaml:"can_push"`
	ExistingPullRequests []*PullRequest      `json:"existing_pull_requests" yaml:"existing_pull_requests"`
	ExistingRequests     ProviderSignal      `json:"existing_requests" yaml:"existing_requests"`
	RiskSignals          []RiskSignal        `json:"risk_signals" yaml:"risk_signals"`
	RecommendedActions   []RecommendedAction `json:"recommended_actions" yaml:"recommended_actions"`
	CreatedPullRequest   *PullRequest        `json:"created_pull_request,omitempty" yaml:"created_pull_request,omitempty"`
}

// PullRequestDraft is a bounded local proposal and the signals used to create it.
type PullRequestDraft struct {
	State            string   `json:"state" yaml:"state"`
	CommitSubjects   []string `json:"commit_subjects" yaml:"commit_subjects"`
	ChangedFiles     []string `json:"changed_files" yaml:"changed_files"`
	CommitsTruncated bool     `json:"commits_truncated" yaml:"commits_truncated"`
	FilesTruncated   bool     `json:"files_truncated" yaml:"files_truncated"`
	Message          string   `json:"message,omitempty" yaml:"message,omitempty"`
}

// CapabilitiesSchemaVersion identifies the stable schema for the command inventory.
const CapabilitiesSchemaVersion = "v1"

// Capability describes whether an installed command can safely be used by an agent.
type Capability struct {
	Command       string   `json:"command" yaml:"command"`
	Status        string   `json:"status" yaml:"status"`
	ReadOnly      bool     `json:"read_only" yaml:"read_only"`
	Formats       []string `json:"formats,omitempty" yaml:"formats,omitempty"`
	SchemaVersion string   `json:"schema_version,omitempty" yaml:"schema_version,omitempty"`
	Notes         string   `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// Capabilities inventories the commands built into this version of GHA.
type Capabilities struct {
	SchemaVersion string       `json:"schema_version" yaml:"schema_version"`
	Commands      []Capability `json:"commands" yaml:"commands"`
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
