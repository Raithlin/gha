package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReviewCommandShowsHelpWithoutConfiguration(t *testing.T) {
	root := NewRootCmd(nil, nil)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetArgs([]string{"review"})

	require.NoError(t, root.Execute())
	assert.Contains(t, output.String(), "Review pull requests from GitHub repositories")
}

func TestReviewCommandRejectsConflictingModes(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.SetArgs([]string{"review", "123", "--mine"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be combined")
}
