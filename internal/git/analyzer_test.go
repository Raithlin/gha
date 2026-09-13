package git

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/pkg/model"
)

func TestAnalyzerCombinesBoundedLocalGitFacts(t *testing.T) {
	workdir := t.TempDir()
	runGit(t, workdir, "init", "-b", "main")
	runGit(t, workdir, "config", "user.email", "test@example.com")
	runGit(t, workdir, "config", "user.name", "Test User")
	require.NoError(t, os.WriteFile(workdir+"/small.txt", []byte("small\n"), 0o644))
	require.NoError(t, os.WriteFile(workdir+"/large.txt", []byte(strings.Repeat("x", 200)), 0o644))
	runGit(t, workdir, "add", "small.txt", "large.txt")
	runGit(t, workdir, "commit", "-m", "initial")
	runGit(t, workdir, "commit", "--allow-empty", "-m", "second")

	require.NoError(t, os.WriteFile(workdir+"/small.txt", []byte("unstaged\n"), 0o644))
	require.NoError(t, os.WriteFile(workdir+"/large.txt", []byte(strings.Repeat("y", 300)), 0o644))
	runGit(t, workdir, "add", "large.txt")
	require.NoError(t, os.WriteFile(workdir+"/untracked.txt", []byte("untracked\n"), 0o644))

	analysis, err := NewAnalyzer(workdir).Analyze(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, model.RepositoryAnalysisSchemaVersion, analysis.SchemaVersion)
	assert.Equal(t, workdir, analysis.Path)
	assert.Equal(t, "available", analysis.Head.State)
	assert.Equal(t, "main", analysis.Head.Branch)
	assert.Equal(t, 2, analysis.Head.Commits)
	assert.Equal(t, "dirty", analysis.Worktree.State)
	assert.Equal(t, 1, analysis.Worktree.Staged)
	assert.Equal(t, 1, analysis.Worktree.Unstaged)
	assert.Equal(t, 1, analysis.Worktree.Untracked)
	assert.Len(t, analysis.Worktree.Changes, 1)
	assert.True(t, analysis.Worktree.ChangesTruncated)
	assert.Len(t, analysis.RecentCommits, 1)
	assert.True(t, analysis.RecentCommitsTruncated)
	assert.Equal(t, "second", analysis.RecentCommits[0].Subject)
	assert.Equal(t, "available", analysis.LargestFilesSignal.State)
	require.Len(t, analysis.LargestFiles, 1)
	assert.Equal(t, "large.txt", analysis.LargestFiles[0].Path)
	assert.EqualValues(t, 200, analysis.LargestFiles[0].Bytes)
	assert.True(t, analysis.LargestFilesTruncated)
}

func TestAnalyzerReportsUnbornHeadWithoutFailing(t *testing.T) {
	workdir := t.TempDir()
	runGit(t, workdir, "init", "-b", "main")

	analysis, err := NewAnalyzer(workdir).Analyze(context.Background(), 30)

	require.NoError(t, err)
	assert.Equal(t, "unborn", analysis.Head.State)
	assert.Empty(t, analysis.RecentCommits)
	assert.Equal(t, "unavailable", analysis.LargestFilesSignal.State)
	assert.Empty(t, analysis.LargestFiles)
}
