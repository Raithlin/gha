package model

// CleanupExecutionSchemaVersion identifies selected cleanup effects.
const CleanupExecutionSchemaVersion = "v1"

// CleanupMergeEvidence binds a fresh provider assessment to a local branch tip.
// It is inspection evidence, never a reusable authorization to delete a ref.
type CleanupMergeEvidence struct {
	State       string        `json:"state" yaml:"state"`
	Reason      string        `json:"reason" yaml:"reason"`
	Message     string        `json:"message" yaml:"message"`
	Repository  RepositoryRef `json:"repository" yaml:"repository"`
	ObservedAt  string        `json:"observed_at" yaml:"observed_at"`
	PullRequest *PullRequest  `json:"pull_request,omitempty" yaml:"pull_request,omitempty"`
}

// CleanupExecution records only explicitly selected branch effects. Retained
// branches are safe non-actions; failed effects require recovery before retry.
type CleanupExecution struct {
	SchemaVersion string                 `json:"schema_version" yaml:"schema_version"`
	Selection     string                 `json:"selection,omitempty" yaml:"selection,omitempty"`
	DryRun        bool                   `json:"dry_run" yaml:"dry_run"`
	Results       []*CleanupBranchResult `json:"results" yaml:"results"`
}

// CleanupBranchResult records the immutable tips and observed cleanup effects.
type CleanupBranchResult struct {
	Name         string                `json:"name" yaml:"name"`
	SHA          string                `json:"sha" yaml:"sha"`
	OriginSHA    string                `json:"origin_sha,omitempty" yaml:"origin_sha,omitempty"`
	Forced       bool                  `json:"forced" yaml:"forced"`
	State        string                `json:"state" yaml:"state"`
	Local        string                `json:"local" yaml:"local"`
	Origin       string                `json:"origin" yaml:"origin"`
	Worktree     string                `json:"worktree" yaml:"worktree"`
	WorktreePath string                `json:"worktree_path,omitempty" yaml:"worktree_path,omitempty"`
	Reason       string                `json:"reason" yaml:"reason"`
	Recovery     string                `json:"recovery,omitempty" yaml:"recovery,omitempty"`
	Evidence     *CleanupMergeEvidence `json:"merge_evidence,omitempty" yaml:"merge_evidence,omitempty"`
}
