package git

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchWriterPreflightUsesExactLocalBranchNamesWithMatchingTag(t *testing.T) {
	workdir, _ := branchValidationRepository(t)
	runGit(t, workdir, "branch", "feature")
	runGit(t, workdir, "tag", "feature")
	writer := NewBranchWriter(workdir)
	ctx := context.Background()

	assert.ErrorContains(t, writer.ValidateCreate(ctx, "feature", "", false), "already exists locally")
	require.NoError(t, writer.ValidateRename(ctx, "feature", "renamed", false))
	require.NoError(t, writer.ValidateDelete(ctx, "feature", true, false))
}

func TestBranchWriterPreflightDeleteUsesUpstreamBeforeSwitchTarget(t *testing.T) {
	workdir, _ := branchValidationRepository(t)
	runGit(t, workdir, "switch", "-c", "feature")
	runGit(t, workdir, "commit", "--allow-empty", "-m", "feature work")
	runGit(t, workdir, "push", "-u", "origin", "feature")
	runGit(t, workdir, "switch", "main")
	writer := NewBranchWriter(workdir)
	ctx := context.Background()

	// The feature tip is on its upstream, though main is still behind.
	require.NoError(t, writer.ValidateLocalDelete(ctx, "feature", false, "main"))
	require.NoError(t, writer.DeleteLocal(ctx, "feature", false))
}

func TestBranchWriterPreflightDeleteRejectsUpstreamBehindMergedSwitchTarget(t *testing.T) {
	workdir, _ := branchValidationRepository(t)
	runGit(t, workdir, "switch", "-c", "feature")
	runGit(t, workdir, "push", "-u", "origin", "feature")
	runGit(t, workdir, "commit", "--allow-empty", "-m", "feature work")
	runGit(t, workdir, "switch", "main")
	runGit(t, workdir, "merge", "--ff-only", "feature")
	writer := NewBranchWriter(workdir)
	ctx := context.Background()

	assert.ErrorContains(t, writer.ValidateLocalDelete(ctx, "feature", false, "main"), "not fully merged")
	assert.ErrorContains(t, writer.DeleteLocal(ctx, "feature", false), "not fully merged")
}

func TestBranchWriterPreflightDeleteFallsBackWhenUpstreamRefIsMissing(t *testing.T) {
	workdir, _ := branchValidationRepository(t)
	runGit(t, workdir, "branch", "feature")
	runGit(t, workdir, "push", "-u", "origin", "feature")
	runGit(t, workdir, "update-ref", "-d", "refs/remotes/origin/feature")
	writer := NewBranchWriter(workdir)
	ctx := context.Background()

	// Git retains the upstream setting, but -d checks HEAD when that ref is gone.
	require.NoError(t, writer.ValidateLocalDelete(ctx, "feature", false, ""))
	require.NoError(t, writer.DeleteLocal(ctx, "feature", false))
}
