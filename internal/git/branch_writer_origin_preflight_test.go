package git

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchWriterOriginPreflightUsesConfiguredPushURL(t *testing.T) {
	workdir, fetchRemote := branchValidationRepository(t)
	pushRemote := t.TempDir()
	runGit(t, pushRemote, "init", "--bare")
	runGit(t, workdir, "branch", "on-write")
	runGit(t, workdir, "push", fetchRemote, "on-write")
	runGit(t, workdir, "branch", "-D", "on-write")
	runGit(t, workdir, "branch", "only-on-push")
	runGit(t, workdir, "push", pushRemote, "only-on-push")
	runGit(t, workdir, "branch", "-D", "only-on-push")
	runGit(t, workdir, "config", "remote.origin.pushurl", pushRemote)

	writer := NewBranchWriter(workdir)
	ctx := context.Background()
	require.NoError(t, writer.ValidateCreate(ctx, "on-write", "main", true))
	assert.ErrorContains(t, writer.ValidateCreate(ctx, "only-on-push", "main", true), "already exists on origin")
	assert.ErrorContains(t, writer.ValidateDelete(ctx, "on-write", false, true), "does not exist on origin")
	require.NoError(t, writer.ValidateDelete(ctx, "only-on-push", false, true))
}

func TestBranchWriterOriginPreflightUsesRewrittenPushURL(t *testing.T) {
	workdir, fetchRemote := branchValidationRepository(t)
	pushRemote := t.TempDir()
	runGit(t, pushRemote, "init", "--bare")
	runGit(t, workdir, "branch", "only-on-push")
	runGit(t, workdir, "push", pushRemote, "only-on-push")
	runGit(t, workdir, "branch", "-D", "only-on-push")
	runGit(t, workdir, "config", "url."+pushRemote+".pushInsteadOf", fetchRemote)

	assert.Equal(t, pushRemote, strings.TrimSpace(runGit(t, workdir, "remote", "get-url", "--push", "origin")))
	assert.ErrorContains(t, NewBranchWriter(workdir).ValidateCreate(context.Background(), "only-on-push", "main", true), "already exists on origin")
}

func TestBranchWriterOriginPreflightRejectsMultiplePushURLs(t *testing.T) {
	workdir, _ := branchValidationRepository(t)
	firstRemote := t.TempDir()
	secondRemote := t.TempDir()
	runGit(t, firstRemote, "init", "--bare")
	runGit(t, secondRemote, "init", "--bare")
	runGit(t, workdir, "config", "--add", "remote.origin.pushurl", firstRemote)
	runGit(t, workdir, "config", "--add", "remote.origin.pushurl", secondRemote)

	err := NewBranchWriter(workdir).ValidateCreate(context.Background(), "feature", "main", true)
	assert.ErrorContains(t, err, "multiple push URLs")
}
