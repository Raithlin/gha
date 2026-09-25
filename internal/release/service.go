// Package release coordinates guarded annotated-tag publication.
package release

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/raithlin/gha/pkg/model"
)

// SchemaVersion identifies the release-publication result contract.
const SchemaVersion = "v1"

var semver = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)

// WorkflowState describes the configured tag trigger.
type WorkflowState struct {
	State      string `json:"state" yaml:"state"`
	Path       string `json:"path,omitempty" yaml:"path,omitempty"`
	TagPattern string `json:"tag_pattern,omitempty" yaml:"tag_pattern,omitempty"`
	Message    string `json:"message,omitempty" yaml:"message,omitempty"`
}

// WorkflowObservation never equates a successful push with a completed release.
type WorkflowObservation struct {
	State   string `json:"state" yaml:"state"`
	URL     string `json:"url,omitempty" yaml:"url,omitempty"`
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
}

// CheckoutState records local and freshly read origin state for a checkout.
type CheckoutState struct {
	Repository   model.RepositoryRef
	Commit       string
	Branch       string
	Clean        bool
	OriginCommit string
	LocalTag     string
	Workflow     WorkflowState
}

// ProviderState records provider safety signals.
type ProviderState struct {
	Repository  *model.Repository
	Checks      []*model.CheckRun
	ChecksError error
	RemoteTag   string
	Workflow    WorkflowState
}

// Plan is the stable machine and terminal contract for one proposed tag.
type Plan struct {
	SchemaVersion   string              `json:"schema_version" yaml:"schema_version"`
	Repository      model.RepositoryRef `json:"repository" yaml:"repository"`
	Version         string              `json:"version" yaml:"version"`
	Tag             string              `json:"tag" yaml:"tag"`
	Commit          string              `json:"commit" yaml:"commit"`
	Branch          string              `json:"branch" yaml:"branch"`
	OriginRef       string              `json:"origin_ref" yaml:"origin_ref"`
	DryRun          bool                `json:"dry_run" yaml:"dry_run"`
	Ready           bool                `json:"ready" yaml:"ready"`
	Blockers        []string            `json:"blockers" yaml:"blockers"`
	Checks          []*model.CheckRun   `json:"checks" yaml:"checks"`
	Workflow        WorkflowState       `json:"workflow" yaml:"workflow"`
	LocalTag        string              `json:"local_tag" yaml:"local_tag"`
	OriginTag       string              `json:"origin_tag" yaml:"origin_tag"`
	ReleaseWorkflow WorkflowObservation `json:"release_workflow" yaml:"release_workflow"`
}

// Checkout performs local and Git-origin operations.
type Checkout interface {
	Inspect(context.Context, string) (CheckoutState, error)
	CreateTag(context.Context, string, string) error
	PushTag(context.Context, string) error
}

// Provider supplies safety data and workflow observation.
type Provider interface {
	Inspect(context.Context, model.RepositoryRef, string, string) (ProviderState, error)
	Observe(context.Context, model.RepositoryRef, string, string, string) (WorkflowObservation, error)
}

// Service coordinates the release preflight and write.
type Service struct {
	checkout Checkout
	provider Provider
}

// NewService creates a release workflow from checkout and provider boundaries.
func NewService(checkout Checkout, provider Provider) *Service {
	return &Service{checkout: checkout, provider: provider}
}

// Provider returns the configured provider for a checkout selected by --path.
func (s *Service) Provider() Provider { return s.provider }

// Prepare performs read-only checks and returns the exact planned effects.
func (s *Service) Prepare(ctx context.Context, repository model.RepositoryRef, version, _ string, dryRun bool) (*Plan, error) {
	if s == nil || s.checkout == nil || s.provider == nil {
		return nil, fmt.Errorf("release publication is not configured")
	}
	if !semver.MatchString(version) {
		return nil, fmt.Errorf("version %q is not SemVer", version)
	}
	tag := version
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	local, err := s.checkout.Inspect(ctx, tag)
	if err != nil {
		return nil, fmt.Errorf("inspect checkout: %w", err)
	}
	remote, err := s.provider.Inspect(ctx, repository, local.Commit, tag)
	if err != nil {
		return nil, fmt.Errorf("inspect provider: %w", err)
	}
	plan := &Plan{SchemaVersion: SchemaVersion, Repository: repository, Version: version, Tag: tag, Commit: local.Commit, Branch: local.Branch, OriginRef: "refs/tags/" + tag, DryRun: dryRun, Blockers: []string{}, Checks: remote.Checks, Workflow: local.Workflow, LocalTag: "planned", OriginTag: "planned", ReleaseWorkflow: WorkflowObservation{State: "not_triggered"}}
	plan.Blockers = releaseBlockers(repository, local, remote)
	plan.Ready = len(plan.Blockers) == 0
	return plan, nil
}

func releaseBlockers(repository model.RepositoryRef, local CheckoutState, remote ProviderState) []string {
	blockers := make([]string, 0)
	if local.Repository != repository {
		blockers = append(blockers, "checkout origin does not match selected repository")
	}
	blockers = append(blockers, providerBlockers(local.Branch, remote.Repository)...)
	if !local.Clean {
		blockers = append(blockers, "checkout has uncommitted changes")
	}
	if local.Commit == "" || local.OriginCommit != local.Commit {
		blockers = append(blockers, "selected commit is not the current origin branch tip")
	}
	if local.LocalTag != "" {
		blockers = append(blockers, "local tag already exists")
	}
	if remote.RemoteTag != "" {
		blockers = append(blockers, "origin tag already exists")
	}
	if local.Workflow.State != "available" {
		blockers = append(blockers, "tag-triggered release workflow is unavailable")
	}
	if checkBlocker := releaseCheckBlocker(remote.Checks, remote.ChecksError); checkBlocker != "" {
		blockers = append(blockers, checkBlocker)
	}
	return blockers
}

func providerBlockers(branch string, repository *model.Repository) []string {
	blockers := make([]string, 0, 2)
	if branch == "" || repository == nil || branch != repository.DefaultBranch {
		blockers = append(blockers, "checkout must be on the provider default branch")
	}
	if repository == nil || repository.Permissions == nil || (!repository.Permissions.Push && !repository.Permissions.Admin) {
		blockers = append(blockers, "provider push permission is unavailable or denied")
	}
	return blockers
}

func releaseCheckBlocker(checks []*model.CheckRun, checkErr error) string {
	if checkErr != nil || len(checks) == 0 {
		return "required CI checks are unavailable"
	}
	for _, check := range checks {
		if check == nil || check.Status != "completed" || check.Conclusion != "success" {
			return "CI check is not successful"
		}
	}
	return ""
}

// Publish rechecks the plan before creating or pushing the annotated tag.
func (s *Service) Publish(ctx context.Context, plan *Plan) error {
	if plan == nil || !plan.Ready || plan.DryRun {
		return fmt.Errorf("release plan is not ready for publication")
	}
	fresh, err := s.Prepare(ctx, plan.Repository, plan.Version, "", false)
	if err != nil {
		return err
	}
	if !fresh.Ready || fresh.Commit != plan.Commit {
		return fmt.Errorf("release preflight changed; run --dry-run again")
	}
	if err := s.checkout.CreateTag(ctx, plan.Tag, plan.Commit); err != nil {
		plan.LocalTag = "failed"
		return fmt.Errorf("create annotated tag: %w", err)
	}
	plan.LocalTag = "completed"
	if err := s.checkout.PushTag(ctx, plan.Tag); err != nil {
		plan.OriginTag = "failed"
		return fmt.Errorf("push tag to origin: %w", err)
	}
	plan.OriginTag = "completed"
	observation, err := s.provider.Observe(ctx, plan.Repository, plan.Workflow.Path, plan.Tag, plan.Commit)
	if err != nil {
		plan.ReleaseWorkflow = WorkflowObservation{State: "unavailable", Message: err.Error()}
		return nil
	}
	plan.ReleaseWorkflow = observation
	return nil
}
