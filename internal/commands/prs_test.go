package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPRsCommandShowsHelpWithoutConfiguration(t *testing.T) {
	root := NewRootCmd(nil, nil)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetArgs([]string{"prs", "--help"})

	require.NoError(t, root.Execute())
	assert.Contains(t, output.String(), "List pull requests in a GitHub repository")
	assert.Contains(t, output.String(), "--queue")
}

func TestPRsCommandRejectsConflictingModes(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.SetArgs([]string{"prs", "--mine", "--queue"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "use only one")
}

func TestPRsCommandRejectsListFiltersWithSpecialMode(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.SetArgs([]string{"prs", "--mine", "--state", "all"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be combined with list filters")
}

func TestPRsCommandAllowsLimitWithSpecialMode(t *testing.T) {
	command := newPRsCmd(nil, nil)
	require.NoError(t, command.Flags().Set("limit", "10"))

	assert.False(t, hasListFilters(command))
}

func TestPRsCommandValidatesListFiltersBeforeResolvingRepository(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.SetArgs([]string{"prs", "--state", "merged"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported state")
}
