package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/release"
	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/require"
)

type statusProviderStub struct{ result *release.Status }

func (p statusProviderStub) ReadStatus(_ context.Context, _ model.RepositoryRef, _, _ string) (*release.Status, error) {
	return p.result, nil
}

func TestReleaseStatusJSONAndTextShareResult(t *testing.T) {
	provider := statusProviderStub{result: &release.Status{State: "published", TagState: "available", Commit: "abc123",
		Workflow:      release.StatusWorkflow{State: "succeeded", RunID: 7, Attempt: 2, Path: ".github/workflows/release.yml"},
		HostedRelease: release.HostedRelease{State: "available", URL: "https://example.test/release"}}}
	for _, format := range []string{"json", "text"} {
		command := newReleaseStatusCmd(git.NewRepositoryResolver("acme/tool"), provider)
		command.SetArgs([]string{"v1.2.3", "--format", format})
		var stdout, stderr bytes.Buffer
		command.SetOut(&stdout)
		command.SetErr(&stderr)
		require.NoError(t, command.Execute())
		require.Empty(t, stderr.String())
		if format == "json" {
			var got release.Status
			require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
			require.Equal(t, "published", got.State)
			require.Equal(t, int64(7), got.Workflow.RunID)
		} else {
			require.Contains(t, stdout.String(), "published")
			require.Contains(t, stdout.String(), "abc123")
			require.Contains(t, stdout.String(), "7")
		}
	}
}

func TestReleaseStatusRejectsInvalidWaitArguments(t *testing.T) {
	for _, args := range [][]string{{"v1.2.3", "--wait", "--timeout", "0s"}, {"v1.2.3", "--timeout", "1m"}, {"bad tag"}} {
		command := newReleaseStatusCmd(git.NewRepositoryResolver("acme/tool"), statusProviderStub{result: &release.Status{State: "pending"}})
		command.SetArgs(args)
		require.Error(t, command.Execute())
	}
}

func TestReleaseStatusWaitRendersPartialObservationWhenContextCancels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	provider := statusProviderStub{result: &release.Status{State: "pending", TagState: "available", Commit: "abc123"}}
	command := newReleaseStatusCmd(git.NewRepositoryResolver("acme/tool"), provider)
	command.SetContext(ctx)
	command.SetArgs([]string{"v1.2.3", "--wait", "--format", "json"})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	cancel()
	require.NoError(t, command.Execute())
	var result release.Status
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Equal(t, "cancelled", result.Wait.State)
	require.Equal(t, "abc123", result.Commit)
}
