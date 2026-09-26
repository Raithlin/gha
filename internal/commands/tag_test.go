package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagPublishDryRunPlansExactCommitWithoutWriting(t *testing.T) {
	checkout, remote := mutationRepository(t)
	var out bytes.Buffer
	cmd := newTagCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"publish", "v1.2.3", "--path", checkout, "--commit", "HEAD", "--dry-run", "--format", "json"})
	require.NoError(t, cmd.Execute())
	var result model.TagPublication
	require.NoError(t, json.Unmarshal(out.Bytes(), &result))
	assert.True(t, result.Ready)
	assert.True(t, result.DryRun)
	assert.Equal(t, "planned", result.Local)
	assert.Equal(t, "planned", result.Origin)
	assertBranchMissing(t, remote, "v1.2.3")
	check := exec.Command("git", "-C", checkout, "show-ref", "--verify", "--quiet", "refs/tags/v1.2.3")
	assert.Error(t, check.Run())
}

func TestTagPublishRunsByDefaultAndPushesExactTag(t *testing.T) {
	checkout, remote := mutationRepository(t)
	cmd := newTagCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"publish", "v2", "--path", checkout, "--format", "json"})
	require.NoError(t, cmd.Execute())
	var result model.TagPublication
	require.NoError(t, json.Unmarshal(out.Bytes(), &result))
	assert.Equal(t, "completed", result.Local)
	assert.Equal(t, "completed", result.Origin)
	local := exec.Command("git", "-C", checkout, "rev-parse", "v2^{commit}")
	localOut, err := local.Output()
	require.NoError(t, err)
	remoteCmd := exec.Command("git", "--git-dir", remote, "rev-parse", "refs/tags/v2")
	remoteOut, err := remoteCmd.Output()
	require.NoError(t, err)
	assert.Equal(t, bytes.TrimSpace(localOut), bytes.TrimSpace(remoteOut))
}

func TestTagPublishRejectsInvalidTagAndCommit(t *testing.T) {
	checkout, _ := mutationRepository(t)
	for _, tc := range []struct{ name, commit, want string }{{"bad..tag", "HEAD", "invalid tag name"}, {"valid", "missing-ref", "resolve commit"}} {
		cmd := newTagCmd()
		cmd.SetArgs([]string{"publish", tc.name, "--path", checkout, "--commit", tc.commit, "--dry-run"})
		assert.ErrorContains(t, cmd.Execute(), tc.want)
	}
}

func TestTagPublishBlocksExistingLocalAndOriginTags(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "tag", "existing", "HEAD")
	runMutationGit(t, "", "--git-dir", remote, "tag", "remote-tag", "refs/heads/main")
	for _, name := range []string{"existing", "remote-tag"} {
		cmd := newTagCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"publish", name, "--path", checkout, "--dry-run", "--format", "json"})
		require.NoError(t, cmd.Execute())
		var got model.TagPublication
		require.NoError(t, json.Unmarshal(out.Bytes(), &got))
		assert.False(t, got.Ready)
		assert.NotEmpty(t, got.Blockers)
	}
}

func TestTagPublishReportsPartialLocalCompletionWhenPushFails(t *testing.T) {
	checkout, remote := mutationRepository(t)
	hook := remote + "/hooks/pre-receive"
	require.NoError(t, os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0o755))
	cmd := newTagCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"publish", "partial", "--path", checkout, "--format", "json"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "local tag created but origin push failed")
	var got model.TagPublication
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	assert.Equal(t, "completed", got.Local)
	assert.Equal(t, "failed", got.Origin)
}

func TestTagPublishReportsLocalCreationFailure(t *testing.T) {
	checkout, _ := mutationRepository(t)
	runMutationGit(t, checkout, "tag", "blocked/child", "HEAD")
	cmd := newTagCmd()
	cmd.SetArgs([]string{"publish", "blocked", "--path", checkout})
	assert.Error(t, cmd.Execute())
	check := exec.Command("git", "-C", checkout, "show-ref", "--verify", "--quiet", "refs/tags/blocked")
	assert.Error(t, check.Run())
}
