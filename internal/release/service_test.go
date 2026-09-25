package release

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/pkg/model"
)

type fakeCheckout struct {
	state                          CheckoutState
	created, pushed                bool
	inspectErr, createErr, pushErr error
}

func (f *fakeCheckout) Inspect(context.Context, string) (CheckoutState, error) {
	return f.state, f.inspectErr
}
func (f *fakeCheckout) CreateTag(context.Context, string, string) error {
	f.created = true
	return f.createErr
}
func (f *fakeCheckout) PushTag(context.Context, string) error { f.pushed = true; return f.pushErr }

type fakeProvider struct {
	repository             *model.Repository
	checks                 []*model.CheckRun
	workflow               WorkflowState
	remoteTag              string
	checkErr               error
	inspectErr, observeErr error
}

func (f *fakeProvider) Inspect(context.Context, model.RepositoryRef, string, string) (ProviderState, error) {
	return ProviderState{Repository: f.repository, Checks: f.checks, Workflow: f.workflow, RemoteTag: f.remoteTag, ChecksError: f.checkErr}, f.inspectErr
}
func (f *fakeProvider) Observe(context.Context, model.RepositoryRef, string, string, string) (WorkflowObservation, error) {
	return WorkflowObservation{State: "triggered"}, f.observeErr
}

func readyFixtures() (*fakeCheckout, *fakeProvider) {
	return &fakeCheckout{state: CheckoutState{Repository: model.RepositoryRef{Owner: "acme", Name: "tool"}, Commit: "abc123", Branch: "main", Clean: true, OriginCommit: "abc123", Workflow: WorkflowState{State: "available", Path: ".github/workflows/release.yml", TagPattern: "v*"}}}, &fakeProvider{
		repository: &model.Repository{DefaultBranch: "main", Permissions: &model.RepositoryPermissions{Push: true}},
		checks:     []*model.CheckRun{{Name: "test", Status: "completed", Conclusion: "success"}},
		workflow:   WorkflowState{State: "available", Path: ".github/workflows/release.yml", TagPattern: "v*"},
	}
}

func TestPrepareIsReadOnlyAndNamesExactEffects(t *testing.T) {
	checkout, provider := readyFixtures()
	service := NewService(checkout, provider)
	plan, err := service.Prepare(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "1.2.3", "", true)
	require.NoError(t, err)
	assert.Equal(t, "v1.2.3", plan.Tag)
	assert.Equal(t, "abc123", plan.Commit)
	assert.Equal(t, "refs/tags/v1.2.3", plan.OriginRef)
	assert.Equal(t, "planned", plan.LocalTag)
	assert.Equal(t, "planned", plan.OriginTag)
	assert.True(t, plan.Ready)
	assert.False(t, checkout.created)
	assert.False(t, checkout.pushed)
}

func TestPrepareFailsClosedOnUnsafeSignals(t *testing.T) {
	tests := []struct {
		name   string
		change func(*fakeCheckout, *fakeProvider)
	}{
		{"dirty checkout", func(c *fakeCheckout, _ *fakeProvider) { c.state.Clean = false }},
		{"unpublished commit", func(c *fakeCheckout, _ *fakeProvider) { c.state.OriginCommit = "other" }},
		{"existing remote tag", func(_ *fakeCheckout, p *fakeProvider) { p.remoteTag = "abc123" }},
		{"failed CI", func(_ *fakeCheckout, p *fakeProvider) { p.checks[0].Conclusion = "failure" }},
		{"unknown CI", func(_ *fakeCheckout, p *fakeProvider) { p.checkErr = errors.New("offline") }},
		{"missing trigger", func(c *fakeCheckout, _ *fakeProvider) { c.state.Workflow.State = "unavailable" }},
		{"missing reviewed notes", func(c *fakeCheckout, _ *fakeProvider) {
			c.state.ReleaseNotes = NotesState{State: "unavailable", Path: "docs/releases/v1.2.3.md"}
		}},
		{"invalid notes configuration", func(c *fakeCheckout, _ *fakeProvider) {
			c.state.ReleaseNotes = NotesState{State: "unavailable", Message: "directory is empty"}
		}},
		{"wrong repository", func(c *fakeCheckout, _ *fakeProvider) { c.state.Repository.Name = "other" }},
		{"wrong branch", func(c *fakeCheckout, _ *fakeProvider) { c.state.Branch = "feature" }},
		{"existing local tag", func(c *fakeCheckout, _ *fakeProvider) { c.state.LocalTag = "v1.2.3" }},
		{"missing permission", func(_ *fakeCheckout, p *fakeProvider) { p.repository.Permissions = nil }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checkout, provider := readyFixtures()
			test.change(checkout, provider)
			plan, err := NewService(checkout, provider).Prepare(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "1.2.3", "", true)
			require.NoError(t, err)
			assert.False(t, plan.Ready)
			assert.NotEmpty(t, plan.Blockers)
		})
	}
}

func TestPrepareValidatesVersionAndWrapsInspectionErrors(t *testing.T) {
	checkout, provider := readyFixtures()
	service := NewService(checkout, provider)
	_, err := service.Prepare(context.Background(), model.RepositoryRef{}, "release-one", "", true)
	assert.ErrorContains(t, err, "SemVer")
	checkout.inspectErr = errors.New("git failed")
	_, err = service.Prepare(context.Background(), model.RepositoryRef{}, "1.2.3", "", true)
	assert.ErrorContains(t, err, "inspect checkout")
	checkout.inspectErr = nil
	provider.inspectErr = errors.New("api failed")
	_, err = service.Prepare(context.Background(), model.RepositoryRef{}, "1.2.3", "", true)
	assert.ErrorContains(t, err, "inspect provider")
}

func TestPublishReportsPartialEffectsAndUnavailableObservation(t *testing.T) {
	for _, test := range []struct {
		name                       string
		change                     func(*fakeCheckout, *fakeProvider)
		local, origin, observation string
	}{
		{"tag failure", func(c *fakeCheckout, _ *fakeProvider) { c.createErr = errors.New("tag failed") }, "failed", "planned", "not_triggered"},
		{"push failure", func(c *fakeCheckout, _ *fakeProvider) { c.pushErr = errors.New("push failed") }, "completed", "failed", "not_triggered"},
		{"observation failure", func(_ *fakeCheckout, p *fakeProvider) { p.observeErr = errors.New("actions unavailable") }, "completed", "completed", "unavailable"},
	} {
		t.Run(test.name, func(t *testing.T) {
			checkout, provider := readyFixtures()
			service := NewService(checkout, provider)
			plan, err := service.Prepare(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "1.2.3", "", false)
			require.NoError(t, err)
			test.change(checkout, provider)
			_ = service.Publish(context.Background(), plan)
			assert.Equal(t, test.local, plan.LocalTag)
			assert.Equal(t, test.origin, plan.OriginTag)
			assert.Equal(t, test.observation, plan.ReleaseWorkflow.State)
		})
	}
}

func TestPublishRequiresReadyPlan(t *testing.T) {
	checkout, provider := readyFixtures()
	service := NewService(checkout, provider)
	plan, err := service.Prepare(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "1.2.3", "", false)
	require.NoError(t, err)
	require.NoError(t, service.Publish(context.Background(), plan))
	assert.True(t, checkout.created)
	assert.True(t, checkout.pushed)
	assert.Equal(t, "completed", plan.OriginTag)
	assert.Equal(t, "triggered", plan.ReleaseWorkflow.State)

	plan.Ready = false
	assert.Error(t, service.Publish(context.Background(), plan))
}

func TestPublishRechecksCommitAndRejectsDryRun(t *testing.T) {
	checkout, provider := readyFixtures()
	service := NewService(checkout, provider)
	assert.Equal(t, provider, service.Provider())
	plan, err := service.Prepare(context.Background(), model.RepositoryRef{Owner: "acme", Name: "tool"}, "1.2.3", "", true)
	require.NoError(t, err)
	assert.ErrorContains(t, service.Publish(context.Background(), plan), "not ready")
	plan.DryRun = false
	checkout.state.Commit = "changed"
	checkout.state.OriginCommit = "changed"
	assert.ErrorContains(t, service.Publish(context.Background(), plan), "preflight changed")
	assert.False(t, checkout.created)
	checkout.inspectErr = errors.New("offline")
	assert.ErrorContains(t, service.Publish(context.Background(), plan), "inspect checkout")
}

func TestPrepareRejectsMissingService(t *testing.T) {
	var service *Service
	_, err := service.Prepare(context.Background(), model.RepositoryRef{}, "1.2.3", "", true)
	require.ErrorContains(t, err, "not configured")
}
