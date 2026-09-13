package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchesCommandShowsHelpWithoutConfiguration(t *testing.T) {
	root := NewRootCmd(nil, nil, nil)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetArgs([]string{"branches", "--help"})

	require.NoError(t, root.Execute())
	assert.Contains(t, output.String(), "Inspect local branches and the remote-tracking branches for origin")
	assert.Contains(t, output.String(), "--limit")
}

func TestBranchesCommandValidatesLimitBeforeInspectingGit(t *testing.T) {
	command := newBranchesCmd(nil)
	command.SetArgs([]string{"--limit", "0"})

	err := command.Execute()

	require.Error(t, err)
	assert.ErrorContains(t, err, "limit must be between 1 and 100")
}
