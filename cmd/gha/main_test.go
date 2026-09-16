package main

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMainRunsAReadOnlyVersionCommand(t *testing.T) {
	originalArgs := os.Args
	originalStdout := os.Stdout
	t.Cleanup(func() {
		os.Args = originalArgs
		os.Stdout = originalStdout
	})

	read, write, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = write
	os.Args = []string{"gha", "version"}

	main()
	require.NoError(t, write.Close())
	output, err := io.ReadAll(read)
	require.NoError(t, err)
	require.NoError(t, read.Close())
	assert.Contains(t, string(output), "GHA version")
}

func TestRunReturnsCommandErrorsWithoutExitingTheProcess(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"gha", "not-a-command"}

	err := run()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}
