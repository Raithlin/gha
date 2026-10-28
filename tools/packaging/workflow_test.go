package packaging

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPackageWorkflowIsReusableAndManual(t *testing.T) {
	workflow := readPackageWorkflow(t, "packages.yml")
	require.Contains(t, workflow.On, "workflow_call")
	require.Contains(t, workflow.On, "workflow_dispatch")
	require.NotContains(t, workflow.On, "pull_request", "PR opt-in belongs in its caller")
	require.NotContains(t, workflow.On, "push", "ordinary pushes must not run installation tests")
	require.Equal(t, "read", workflow.Permissions["contents"])
	require.Equal(t, "${{ github.workflow }}-${{ github.ref }}", workflow.Concurrency.Group)
	require.True(t, workflow.Concurrency.Cancel)
	for _, job := range []string{"build", "linux", "macos"} {
		require.Contains(t, workflow.Jobs, job)
	}
}

func TestPackagePROptIn(t *testing.T) {
	workflow := readPackageWorkflow(t, "package-pr.yml")
	require.Len(t, workflow.On, 1)
	require.ElementsMatch(t, []string{"opened", "reopened", "synchronize", "labeled"},
		workflow.On["pull_request"].Types)
	require.Empty(t, workflow.On["pull_request"].Paths, "labels work regardless of changed paths")
	require.Equal(t, "read", workflow.Permissions["contents"])
	// Concurrency belongs in the called workflow: an unrelated label event
	// must not cancel a currently running opted-in installation matrix.
	require.Empty(t, workflow.Concurrency.Group)
	require.Len(t, workflow.Jobs, 1)
	job := workflow.Jobs["packages"]
	require.Equal(t, "./.github/workflows/packages.yml", job.Uses)
	require.Equal(t, "contains(github.event.pull_request.labels.*.name, 'ci:packages') && (github.event.action != 'labeled' || github.event.label.name == 'ci:packages')", job.If)
	require.Empty(t, job.Secrets, "PR package tests do not need publication secrets")
}

func TestReleaseWaitsForPackageInstallation(t *testing.T) {
	workflow := readPackageWorkflow(t, "release.yml")
	packages := workflow.Jobs["package-installation"]
	require.Equal(t, "./.github/workflows/packages.yml", packages.Uses)
	require.Equal(t, "github.event_name == 'push'", packages.If,
		"manual release dispatch must remain a credential-only check")
	require.Equal(t, "read", packages.Permissions["contents"])
	require.Empty(t, packages.Secrets)
	release := workflow.Jobs["release"]
	var needs []string
	require.NoError(t, release.Needs.Decode(&needs))
	require.ElementsMatch(t, []string{"tap-access", "package-installation"}, needs)
	require.Equal(t, "github.event_name == 'push'", workflow.Jobs["release"].If)
	require.NotContains(t, workflow.Jobs["release"].If, "always()",
		"a failed or skipped installation gate must prevent publication")
}

type packageWorkflow struct {
	On map[string]struct {
		Types []string
		Paths []string
	} `yaml:"on"`
	Permissions map[string]string
	Concurrency struct {
		Group  string
		Cancel bool `yaml:"cancel-in-progress"`
	}
	Jobs map[string]struct {
		If          string
		Uses        string
		Needs       yaml.Node
		Permissions map[string]string
		Secrets     any
	}
}

func readPackageWorkflow(t *testing.T, name string) packageWorkflow {
	t.Helper()
	data, err := os.ReadFile("../../.github/workflows/" + name)
	require.NoError(t, err)
	var workflow packageWorkflow
	require.NoError(t, yaml.Unmarshal(data, &workflow))
	return workflow
}
