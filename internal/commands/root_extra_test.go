package commands

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecuteRunsTheRootCommandTree(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"gha", "version", "--format", "json"}

	require.NoError(t, Execute(nil, nil, nil))
}
