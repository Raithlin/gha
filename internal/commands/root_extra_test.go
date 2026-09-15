package commands

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootHelpListsAvailableCommands(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetArgs([]string{"--help"})

	require.NoError(t, root.Execute())

	help := output.String()
	assert.Contains(t, help, "Version: dev")
	assert.Contains(t, help, "Available Commands:")
	for _, command := range []string{
		"agent",
		"analyze",
		"branch",
		"branches",
		"capabilities",
		"dashboard",
		"pr",
		"prs",
		"release",
		"releases",
		"review",
		"version",
	} {
		assert.Contains(t, help, command)
	}
}

func TestRootVersionFlagUsesBuildVersion(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetArgs([]string{"--version"})

	require.NoError(t, root.Execute())
	assert.Equal(t, "gha version dev\n", output.String())
}

func TestExecuteRunsTheRootCommandTree(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"gha", "version", "--format", "json"}

	require.NoError(t, Execute(nil, nil, nil))
}
