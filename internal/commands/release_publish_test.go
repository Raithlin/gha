package commands

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/release"
	"github.com/raithlin/gha/pkg/model"
)

type publicationCheckout struct {
	dirty    bool
	tagError bool
}

func (c publicationCheckout) Inspect(context.Context, string) (release.CheckoutState, error) {
	return release.CheckoutState{Repository: model.RepositoryRef{Owner: "acme", Name: "tool"}, Commit: "abc123", Branch: "main", Clean: !c.dirty, OriginCommit: "abc123", Workflow: release.WorkflowState{State: "available", Path: ".github/workflows/release.yml", TagPattern: "v*"}}, nil
}
func (c publicationCheckout) CreateTag(context.Context, string, string) error {
	if c.tagError {
		return fmt.Errorf("tag failed")
	}
	return nil
}
func (publicationCheckout) PushTag(context.Context, string) error { return nil }

type publicationProvider struct{}

func (publicationProvider) Inspect(context.Context, model.RepositoryRef, string, string) (release.ProviderState, error) {
	return release.ProviderState{Repository: &model.Repository{DefaultBranch: "main", Permissions: &model.RepositoryPermissions{Push: true}}, Checks: []*model.CheckRun{{Name: "test", Status: "completed", Conclusion: "success"}}}, nil
}
func (publicationProvider) Observe(context.Context, model.RepositoryRef, string, string, string) (release.WorkflowObservation, error) {
	return release.WorkflowObservation{State: "triggered"}, nil
}

func TestReleasePublishRequiresConfirmationAndRendersDryRun(t *testing.T) {
	service := release.NewService(publicationCheckout{}, publicationProvider{})
	command := newReleasePublishCmd(git.NewRepositoryResolver("acme/tool"), service)
	command.SetContext(context.Background())
	command.SetArgs([]string{"1.2.3", "--format", "json"})
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	err := command.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--confirm-origin")

	command = newReleasePublishCmd(git.NewRepositoryResolver("acme/tool"), service)
	command.SetContext(context.Background())
	command.SetArgs([]string{"1.2.3", "--dry-run", "--format", "json"})
	output.Reset()
	command.SetOut(&output)
	require.NoError(t, command.Execute())
	assert.Contains(t, output.String(), `"tag": "v1.2.3"`)
	assert.Contains(t, output.String(), `"ready": true`)
}

func TestReleasePublishReportsConfirmedEffectsAndBlocksUnsafePlan(t *testing.T) {
	for _, test := range []struct {
		name                  string
		checkout              publicationCheckout
		wantError, wantOutput string
	}{
		{"published", publicationCheckout{}, "", `"origin_tag": "completed"`},
		{"blocked", publicationCheckout{dirty: true}, "preflight blocked", ""},
		{"partial failure", publicationCheckout{tagError: true}, "tag failed", `"local_tag": "failed"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := release.NewService(test.checkout, publicationProvider{})
			command := newReleasePublishCmd(git.NewRepositoryResolver("acme/tool"), service)
			command.SetContext(context.Background())
			command.SetArgs([]string{"1.2.3", "--confirm-origin", "--format", "json"})
			var rendered bytes.Buffer
			command.SetOut(&rendered)
			command.SetErr(&rendered)
			err := command.Execute()
			if test.wantError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.wantError)
			}
			if test.wantOutput != "" {
				assert.Contains(t, rendered.String(), test.wantOutput)
			}
		})
	}
}

func TestReleasePublishPathUsesSelectedCheckout(t *testing.T) {
	service := release.NewService(publicationCheckout{}, publicationProvider{})
	command := newReleasePublishCmd(git.NewRepositoryResolver(""), service)
	command.SetContext(context.Background())
	command.SetArgs([]string{"1.2.3", "--repo", "acme/tool", "--path", t.TempDir(), "--dry-run"})
	err := command.Execute()
	require.ErrorContains(t, err, "inspect checkout")
}

func TestReleasePublishRequiresConfiguredServicesAndRepository(t *testing.T) {
	for _, test := range []struct {
		name     string
		resolver *git.RepositoryResolver
		service  *release.Service
		args     []string
		want     string
	}{
		{"missing resolver", nil, nil, []string{"1.2.3", "--dry-run"}, "resolution is not configured"},
		{"missing service", git.NewRepositoryResolver("acme/tool"), nil, []string{"1.2.3", "--dry-run"}, "publication is not configured"},
		{"invalid repository", git.NewRepositoryResolver(""), release.NewService(publicationCheckout{}, publicationProvider{}), []string{"1.2.3", "--repo", "invalid", "--dry-run"}, "invalid repository"},
	} {
		t.Run(test.name, func(t *testing.T) {
			command := newReleasePublishCmd(test.resolver, test.service)
			command.SetContext(context.Background())
			command.SetArgs(test.args)
			err := command.Execute()
			require.ErrorContains(t, err, test.want)
		})
	}
}
