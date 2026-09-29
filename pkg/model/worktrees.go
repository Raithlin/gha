package model

// WorktreeInventorySchemaVersion identifies the stable worktree-list contract.
const WorktreeInventorySchemaVersion = "v1"

// WorktreeMutationSchemaVersion identifies the guarded worktree-write contract.
const WorktreeMutationSchemaVersion = "v1"

// Worktree describes one checkout registered in a repository's worktree set.
type Worktree struct {
	Path           string `json:"path" yaml:"path"`
	Head           string `json:"head" yaml:"head"`
	Branch         string `json:"branch,omitempty" yaml:"branch,omitempty"`
	Detached       bool   `json:"detached" yaml:"detached"`
	Bare           bool   `json:"bare" yaml:"bare"`
	Current        bool   `json:"current" yaml:"current"`
	Main           bool   `json:"main" yaml:"main"`
	Locked         bool   `json:"locked" yaml:"locked"`
	LockReason     string `json:"lock_reason,omitempty" yaml:"lock_reason,omitempty"`
	Prunable       bool   `json:"prunable" yaml:"prunable"`
	PrunableReason string `json:"prunable_reason,omitempty" yaml:"prunable_reason,omitempty"`
	StatusState    string `json:"status_state" yaml:"status_state"`
	StatusMessage  string `json:"status_message,omitempty" yaml:"status_message,omitempty"`
}

// WorktreeInventory is a bounded, local-only inventory of registered worktrees.
type WorktreeInventory struct {
	SchemaVersion string     `json:"schema_version" yaml:"schema_version"`
	Path          string     `json:"path" yaml:"path"`
	Limit         int        `json:"limit" yaml:"limit"`
	Total         int        `json:"total" yaml:"total"`
	Truncated     bool       `json:"truncated" yaml:"truncated"`
	Worktrees     []Worktree `json:"worktrees" yaml:"worktrees"`
}

// WorktreeMutation records the result of adding or removing a worktree.
type WorktreeMutation struct {
	SchemaVersion string `json:"schema_version" yaml:"schema_version"`
	Operation     string `json:"operation" yaml:"operation"`
	Path          string `json:"path" yaml:"path"`
	Branch        string `json:"branch,omitempty" yaml:"branch,omitempty"`
	From          string `json:"from,omitempty" yaml:"from,omitempty"`
	DryRun        bool   `json:"dry_run" yaml:"dry_run"`
	State         string `json:"state" yaml:"state"`
}
