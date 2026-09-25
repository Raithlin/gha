package release

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runGit(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}

func TestGitCheckoutFindsTagTriggeredWorkflowAndWritesAnnotatedTag(t *testing.T) {
	directory := t.TempDir()
	remote := t.TempDir()
	runGit(t, remote, "init", "--bare")
	runGit(t, directory, "init", "-b", "main")
	runGit(t, directory, "config", "user.email", "test@example.com")
	runGit(t, directory, "config", "user.name", "Test User")
	require.NoError(t, os.MkdirAll(filepath.Join(directory, ".github", "workflows"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(directory, ".github", "workflows", "release.yml"), []byte("name: Release\non:\n  push:\n    tags:\n      - 'v*'\njobs: {}\n"), 0o644))
	runGit(t, directory, "add", ".github/workflows/release.yml")
	runGit(t, directory, "commit", "-m", "initial")
	runGit(t, directory, "remote", "add", "origin", "https://github.com/acme/tool.git")
	runGit(t, directory, "config", "url."+remote+".insteadOf", "https://github.com/acme/tool.git")
	runGit(t, directory, "push", "origin", "main")
	checkout := GitCheckout{Path: directory}
	state, err := checkout.Inspect(context.Background(), "v1.2.3")
	require.NoError(t, err)
	assert.True(t, state.Clean)
	assert.Equal(t, state.Commit, state.OriginCommit)
	assert.Equal(t, "available", state.Workflow.State)
	assert.Equal(t, "not_required", state.ReleaseNotes.State)
	require.NoError(t, checkout.CreateTag(context.Background(), "v1.2.3", state.Commit))
	require.NoError(t, checkout.PushTag(context.Background(), "v1.2.3"))
	state, err = checkout.Inspect(context.Background(), "v1.2.3")
	require.NoError(t, err)
	assert.Equal(t, "v1.2.3", state.LocalTag)
}

func TestGitCheckoutExplainsUnsafeRepositoryStates(t *testing.T) {
	_, err := (GitCheckout{Path: t.TempDir()}).Inspect(context.Background(), "v1.2.3")
	require.ErrorContains(t, err, "rev-parse")

	directory := t.TempDir()
	runGit(t, directory, "init", "-b", "main")
	runGit(t, directory, "config", "user.email", "test@example.com")
	runGit(t, directory, "config", "user.name", "Test User")
	runGit(t, directory, "commit", "--allow-empty", "-m", "initial")
	_, err = (GitCheckout{Path: directory}).Inspect(context.Background(), "v1.2.3")
	require.ErrorContains(t, err, "remote.origin.url")
	runGit(t, directory, "remote", "add", "origin", "https://example.com/acme/tool.git")
	_, err = (GitCheckout{Path: directory}).Inspect(context.Background(), "v1.2.3")
	require.ErrorContains(t, err, "not a GitHub repository")
	runGit(t, directory, "remote", "set-url", "origin", "https://github.com/acme/tool.git")
	runGit(t, directory, "config", "url."+t.TempDir()+".insteadOf", "https://github.com/acme/tool.git")
	_, err = (GitCheckout{Path: directory}).Inspect(context.Background(), "v1.2.3")
	require.ErrorContains(t, err, "origin branch tip")
	runGit(t, directory, "checkout", "--detach")
	_, err = (GitCheckout{Path: directory}).Inspect(context.Background(), "v1.2.3")
	require.ErrorContains(t, err, "checked-out branch")
}

func TestGitCheckoutWorkflowRequiresCommittedMatchingTrigger(t *testing.T) {
	directory := t.TempDir()
	runGit(t, directory, "init", "-b", "main")
	runGit(t, directory, "config", "user.email", "test@example.com")
	runGit(t, directory, "config", "user.name", "Test User")
	runGit(t, directory, "commit", "--allow-empty", "-m", "initial")
	checkout := GitCheckout{Path: directory}
	assert.Equal(t, "unavailable", checkout.workflow(context.Background(), "v1.2.3").State)
	require.NoError(t, os.MkdirAll(filepath.Join(directory, ".github", "workflows"), 0o755))
	workflowPath := filepath.Join(directory, ".github", "workflows", "release.yml")
	require.NoError(t, os.WriteFile(workflowPath, []byte("name: Release\non:\n  push:\n    tags: ['release-*']\n"), 0o644))
	runGit(t, directory, "add", ".github/workflows/release.yml")
	runGit(t, directory, "commit", "-m", "workflow")
	assert.Equal(t, "unavailable", checkout.workflow(context.Background(), "v1.2.3").State)
	require.NoError(t, os.WriteFile(workflowPath, []byte("on: ["), 0o644))
	runGit(t, directory, "add", ".github/workflows/release.yml")
	runGit(t, directory, "commit", "-m", "invalid workflow")
	assert.Equal(t, "unavailable", checkout.workflow(context.Background(), "v1.2.3").State)
	require.NoError(t, os.WriteFile(workflowPath, []byte("on:\n  push:\n    tags: ['v*']\n"), 0o644))
	otherPath := filepath.Join(directory, ".github", "workflows", "other.yaml")
	require.NoError(t, os.WriteFile(otherPath, []byte("on:\n  push:\n    tags: ['v*']\n"), 0o644))
	runGit(t, directory, "add", ".github/workflows")
	runGit(t, directory, "commit", "-m", "two workflows")
	assert.Equal(t, "unavailable", checkout.workflow(context.Background(), "v1.2.3").State)
	selected := GitCheckout{Path: directory, Workflow: ".github/workflows/other.yaml"}
	assert.Equal(t, ".github/workflows/other.yaml", selected.workflow(context.Background(), "v1.2.3").Path)
}

func TestGitCheckoutRequiresCommittedReviewedReleaseNotes(t *testing.T) {
	directory := t.TempDir()
	runGit(t, directory, "init", "-b", "main")
	runGit(t, directory, "config", "user.email", "test@example.com")
	runGit(t, directory, "config", "user.name", "Test User")
	require.NoError(t, os.MkdirAll(filepath.Join(directory, ".github", "workflows"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(directory, ".github", "workflows", "release.yml"), []byte("name: Release\non:\n  push:\n    tags: ['v*']\nenv:\n  GHA_RELEASE_NOTES_DIR: docs/releases\n"), 0o644))
	runGit(t, directory, "add", ".github/workflows/release.yml")
	runGit(t, directory, "commit", "-m", "workflow")
	checkout := GitCheckout{Path: directory}
	workflow := checkout.workflow(context.Background(), "v1.2.3")
	assert.Equal(t, "docs/releases", workflow.ReleaseNotesDir)
	assert.True(t, workflow.ReleaseNotesRequired)
	notes := checkout.releaseNotes(context.Background(), "v1.2.3", workflow)
	assert.Equal(t, "unavailable", notes.State)
	assert.Equal(t, "docs/releases/v1.2.3.md", notes.Path)

	require.NoError(t, os.MkdirAll(filepath.Join(directory, "docs", "releases"), 0o755))
	notesPath := filepath.Join(directory, "docs", "releases", "v1.2.3.md")
	require.NoError(t, os.WriteFile(notesPath, []byte("# v1.2.3\n\nChanges: done.\n"), 0o644))
	assert.Equal(t, "unavailable", checkout.releaseNotes(context.Background(), "v1.2.3", workflow).State)
	runGit(t, directory, "add", "docs/releases/v1.2.3.md")
	runGit(t, directory, "commit", "-m", "reviewed notes")
	assert.Equal(t, "available", checkout.releaseNotes(context.Background(), "v1.2.3", workflow).State)
	remote := t.TempDir()
	runGit(t, remote, "init", "--bare")
	runGit(t, directory, "remote", "add", "origin", "https://github.com/acme/tool.git")
	runGit(t, directory, "config", "url."+remote+".insteadOf", "https://github.com/acme/tool.git")
	runGit(t, directory, "push", "origin", "main")
	state, err := checkout.Inspect(context.Background(), "v1.2.3")
	require.NoError(t, err)
	assert.Equal(t, "available", state.ReleaseNotes.State)
	assert.Equal(t, "docs/releases/v1.2.3.md", state.ReleaseNotes.Path)
	require.NoError(t, os.WriteFile(notesPath, []byte("# REPLACE_ME_VERSION\n"), 0o644))
	runGit(t, directory, "add", "docs/releases/v1.2.3.md")
	runGit(t, directory, "commit", "-m", "unfinished notes")
	assert.Equal(t, "unavailable", checkout.releaseNotes(context.Background(), "v1.2.3", workflow).State)
	require.NoError(t, os.WriteFile(notesPath, []byte(" \n\t"), 0o644))
	runGit(t, directory, "add", "docs/releases/v1.2.3.md")
	runGit(t, directory, "commit", "-m", "empty notes")
	notes = checkout.releaseNotes(context.Background(), "v1.2.3", workflow)
	assert.Equal(t, "unavailable", notes.State)
	assert.Equal(t, "file is empty", notes.Message)
}

func TestGitCheckoutRejectsInvalidReleaseNotesDirectories(t *testing.T) {
	checkout := GitCheckout{Path: t.TempDir()}
	for _, directory := range []string{"..", "../notes", "/tmp/notes", `docs\releases`} {
		notes := checkout.releaseNotes(context.Background(), "v1.2.3", WorkflowState{ReleaseNotesDir: directory, ReleaseNotesRequired: true})
		assert.Equal(t, "unavailable", notes.State)
		assert.Contains(t, notes.Message, "repository-relative")
	}
}

func TestWorkflowReleaseNotesDirRejectsMalformedConfiguration(t *testing.T) {
	directory, configured := workflowReleaseNotesDir("env: [")
	assert.Empty(t, directory)
	assert.False(t, configured)
}

func TestGitCheckoutDoesNotIgnoreAnEmptyReleaseNotesDirectory(t *testing.T) {
	directory := t.TempDir()
	runGit(t, directory, "init", "-b", "main")
	runGit(t, directory, "config", "user.email", "test@example.com")
	runGit(t, directory, "config", "user.name", "Test User")
	require.NoError(t, os.MkdirAll(filepath.Join(directory, ".github", "workflows"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(directory, ".github", "workflows", "release.yml"), []byte("on:\n  push:\n    tags: ['v*']\nenv:\n  GHA_RELEASE_NOTES_DIR: ''\n"), 0o644))
	runGit(t, directory, "add", ".github/workflows/release.yml")
	runGit(t, directory, "commit", "-m", "workflow")
	checkout := GitCheckout{Path: directory}
	workflow := checkout.workflow(context.Background(), "v1.2.3")
	assert.Equal(t, "unavailable", checkout.releaseNotes(context.Background(), "v1.2.3", workflow).State)
}
