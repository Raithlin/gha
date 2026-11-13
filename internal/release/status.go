package release

import (
	"context"
	"fmt"
	"time"

	"github.com/raithlin/gha/pkg/model"
)

// Status reports independently observed release publication evidence.
type Status struct {
	SchemaVersion string              `json:"schema_version" yaml:"schema_version"`
	Repository    model.RepositoryRef `json:"repository" yaml:"repository"`
	Tag           string              `json:"tag" yaml:"tag"`
	TagState      string              `json:"tag_state" yaml:"tag_state"`
	Commit        string              `json:"commit,omitempty" yaml:"commit,omitempty"`
	State         string              `json:"state" yaml:"state"`
	Workflow      StatusWorkflow      `json:"workflow" yaml:"workflow"`
	HostedRelease HostedRelease       `json:"hosted_release" yaml:"hosted_release"`
	Wait          WaitResult          `json:"wait" yaml:"wait"`
}

// StatusWorkflow identifies one exact tag and commit workflow attempt.
type StatusWorkflow struct {
	State      string   `json:"state" yaml:"state"`
	Path       string   `json:"path,omitempty" yaml:"path,omitempty"`
	Candidates []string `json:"candidates,omitempty" yaml:"candidates,omitempty"`
	ID         int64    `json:"id,omitempty" yaml:"id,omitempty"`
	RunID      int64    `json:"run_id,omitempty" yaml:"run_id,omitempty"`
	Attempt    int      `json:"attempt,omitempty" yaml:"attempt,omitempty"`
	Status     string   `json:"status,omitempty" yaml:"status,omitempty"`
	Conclusion string   `json:"conclusion,omitempty" yaml:"conclusion,omitempty"`
	URL        string   `json:"url,omitempty" yaml:"url,omitempty"`
	Message    string   `json:"message,omitempty" yaml:"message,omitempty"`
	Truncated  bool     `json:"truncated" yaml:"truncated"`
}

// HostedRelease records GitHub Release availability independently of Actions.
type HostedRelease struct {
	State      string `json:"state" yaml:"state"`
	URL        string `json:"url,omitempty" yaml:"url,omitempty"`
	Draft      bool   `json:"draft" yaml:"draft"`
	Prerelease bool   `json:"prerelease" yaml:"prerelease"`
	Message    string `json:"message,omitempty" yaml:"message,omitempty"`
}

// WaitResult says whether observation reached a terminal state.
type WaitResult struct {
	State        string `json:"state" yaml:"state"`
	Observations int    `json:"observations" yaml:"observations"`
}

// StatusProvider reads the remote release facts without changing state.
type StatusProvider interface {
	ReadStatus(context.Context, model.RepositoryRef, string, string) (*Status, error)
}

// StatusService provides one-shot and bounded repeated reads.
type StatusService struct{ provider StatusProvider }

// NewStatusService creates a read-only release status workflow.
func NewStatusService(provider StatusProvider) *StatusService {
	return &StatusService{provider: provider}
}

// Check reads one current release status observation.
func (s *StatusService) Check(ctx context.Context, repository model.RepositoryRef, tag, workflow string) (*Status, error) {
	if s == nil || s.provider == nil {
		return nil, fmt.Errorf("release status provider is not configured")
	}
	result, err := s.provider.ReadStatus(ctx, repository, tag, workflow)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("release status provider returned no result")
	}
	result.SchemaVersion, result.Repository, result.Tag = SchemaVersion, repository, tag
	result.Wait = WaitResult{State: "not_requested", Observations: 1}
	return result, nil
}

// Wait preserves the latest observation when time expires or the caller cancels.
func (s *StatusService) Wait(ctx context.Context, repository model.RepositoryRef, tag, workflow string, timeout, interval time.Duration) (*Status, error) {
	if timeout <= 0 || interval <= 0 {
		return nil, fmt.Errorf("wait timeout and interval must be positive")
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var latest *Status
	for observations := 1; ; observations++ {
		result, err := s.Check(ctx, repository, tag, workflow)
		if err != nil {
			if latest != nil && ctx.Err() != nil {
				return finishWait(latest, observations-1, ctx.Err()), nil
			}
			return nil, err
		}
		latest = result
		if result.State == "published" || result.State == "failed" || result.State == "cancelled" {
			result.Wait = WaitResult{State: "completed", Observations: observations}
			return result, nil
		}
		select {
		case <-ctx.Done():
			return finishWait(latest, observations, ctx.Err()), nil
		case <-time.After(interval):
		}
	}
}

func finishWait(result *Status, observations int, err error) *Status {
	state := "cancelled"
	if err == context.DeadlineExceeded {
		state = "timed_out"
	}
	result.Wait = WaitResult{State: state, Observations: observations}
	return result
}
