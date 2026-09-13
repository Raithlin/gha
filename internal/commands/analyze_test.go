package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/pkg/model"
)

type failingRepositoryAnalyzer struct{}

func (failingRepositoryAnalyzer) Analyze(context.Context, int) (*model.RepositoryAnalysis, error) {
	return nil, errors.New("fatal: not a git repository")
}

func TestAnalyzeCommandRendersStructuredErrors(t *testing.T) {
	command := newAnalyzeCmd(failingRepositoryAnalyzer{})
	var diagnostics bytes.Buffer
	command.SetErr(&diagnostics)
	command.SetArgs([]string{"--format", "json"})

	err := command.Execute()

	require.Error(t, err)
	assert.True(t, IsReportedError(err))
	var commandError model.CommandError
	require.NoError(t, json.Unmarshal(diagnostics.Bytes(), &commandError))
	assert.Equal(t, "repository_analysis_failed", commandError.Code)
	assert.Contains(t, commandError.Message, "not a git repository")
}

func TestAnalyzeCommandValidatesLimitBeforeInspectingGit(t *testing.T) {
	command := newAnalyzeCmd(failingRepositoryAnalyzer{})
	command.SetArgs([]string{"--limit", "101"})

	err := command.Execute()

	require.Error(t, err)
	assert.ErrorContains(t, err, "limit must be between 1 and 100")
}

func TestAnalyzeCommandInspectsExplicitLocalPath(t *testing.T) {
	checkout := t.TempDir()
	require.NoError(t, exec.Command("git", "init", "--quiet", "-b", "main", checkout).Run())

	command := newAnalyzeCmd(nil)
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"--path", checkout, "--format", "json"})

	require.NoError(t, command.Execute())
	var analysis model.RepositoryAnalysis
	require.NoError(t, json.Unmarshal(output.Bytes(), &analysis))
	assert.Equal(t, checkout, analysis.Path)
	assert.Equal(t, "unborn", analysis.Head.State)
}
