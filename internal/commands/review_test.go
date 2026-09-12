package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReviewCommandRequiresPullRequestNumber(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.SetArgs([]string{"review"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestReviewCommandRejectsListingModes(t *testing.T) {
	root := NewRootCmd(nil, nil)
	root.SetArgs([]string{"review", "123", "--mine"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown flag")
}
