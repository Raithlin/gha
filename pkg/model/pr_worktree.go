package model

// PRWorktreeSchemaVersion identifies the PR checkout and retirement contract.
const PRWorktreeSchemaVersion = "v1"

// PRWorktree combines PR review and local safety with observed lifecycle effects.
type PRWorktree struct {
	AssociationState   string              `json:"association_state" yaml:"association_state"`
	SchemaVersion      string              `json:"schema_version" yaml:"schema_version"`
	Operation          string              `json:"operation" yaml:"operation"`
	Repository         RepositoryRef       `json:"repository" yaml:"repository"`
	PullRequest        *PullRequest        `json:"pull_request" yaml:"pull_request"`
	Branch             string              `json:"branch" yaml:"branch"`
	Path               string              `json:"path" yaml:"path"`
	DryRun             bool                `json:"dry_run" yaml:"dry_run"`
	Decision           string              `json:"decision" yaml:"decision"`
	State              string              `json:"state" yaml:"state"`
	Fetch              string              `json:"fetch" yaml:"fetch"`
	BranchState        string              `json:"branch_state" yaml:"branch_state"`
	WorktreeState      string              `json:"worktree_state" yaml:"worktree_state"`
	Worktree           *Worktree           `json:"worktree,omitempty" yaml:"worktree,omitempty"`
	Comparison         BranchComparison    `json:"comparison" yaml:"comparison"`
	ProviderState      string              `json:"provider_state" yaml:"provider_state"`
	ReviewState        ProviderSignal      `json:"review_state" yaml:"review_state"`
	Review             *ReviewSummary      `json:"review,omitempty" yaml:"review,omitempty"`
	Push               PRPushTarget        `json:"push" yaml:"push"`
	Reasons            []string            `json:"reasons" yaml:"reasons"`
	RecommendedActions []RecommendedAction `json:"recommended_actions" yaml:"recommended_actions"`
}

// PRPushTarget is an explicit destination for updating the existing PR, never
// a request to publish another branch or create another pull request.
type PRPushTarget struct {
	State      string        `json:"state" yaml:"state"`
	Repository RepositoryRef `json:"repository" yaml:"repository"`
	URL        string        `json:"url,omitempty" yaml:"url,omitempty"`
	Refspec    string        `json:"refspec,omitempty" yaml:"refspec,omitempty"`
}
