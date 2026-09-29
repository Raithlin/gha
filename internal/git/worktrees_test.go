package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorktreeServiceListsCheckoutStatusAndBoundsResults(t *testing.T) {
	root := worktreeTestRepository(t)
	feature := filepath.Join(t.TempDir(), "feature checkout")
	runGit(t, root, "worktree", "add", "--quiet", "-b", "feature/test", feature)

	inventory, err := NewWorktreeService(root).List(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "v1", inventory.SchemaVersion)
	assert.Equal(t, 2, inventory.Total)
	assert.True(t, inventory.Truncated)
	require.Len(t, inventory.Worktrees, 1)
	assert.True(t, inventory.Worktrees[0].Main)
	assert.True(t, inventory.Worktrees[0].Current)
	assert.Equal(t, "clean", inventory.Worktrees[0].StatusState)

	inventory, err = NewWorktreeService(feature).List(context.Background(), 10)
	require.NoError(t, err)
	assert.Len(t, inventory.Worktrees, 2)
	assert.Equal(t, feature, inventory.Worktrees[1].Path)
	assert.Equal(t, "feature/test", inventory.Worktrees[1].Branch)
	assert.True(t, inventory.Worktrees[1].Current)
	assert.False(t, inventory.Worktrees[1].Main)
	assert.Equal(t, "clean", inventory.Worktrees[1].StatusState)
}

func TestWorktreeServiceRejectsInvalidLimitsAndRepositories(t *testing.T) {
	_, err := NewWorktreeService(t.TempDir()).List(context.Background(), 0)
	assert.ErrorContains(t, err, "at least 1")

	_, err = NewWorktreeService(t.TempDir()).List(context.Background(), 1)
	assert.ErrorContains(t, err, "list Git worktrees")
}

func TestWorktreeServiceReportsDirtyAndLockedWorktrees(t *testing.T) {
	root := worktreeTestRepository(t)
	feature := filepath.Join(t.TempDir(), "feature")
	runGit(t, root, "worktree", "add", "--quiet", "-b", "feature/test", feature)
	require.NoError(t, os.WriteFile(filepath.Join(feature, "untracked.txt"), []byte("dirty"), 0o600))
	runGit(t, root, "worktree", "lock", "--reason", "in use", feature)

	inventory, err := NewWorktreeService(root).List(context.Background(), 10)
	require.NoError(t, err)
	for _, worktree := range inventory.Worktrees {
		if worktree.Path == feature {
			assert.True(t, worktree.Locked)
			assert.Equal(t, "in use", worktree.LockReason)
			assert.Equal(t, "dirty", worktree.StatusState)
			return
		}
	}
	t.Fatal("linked worktree missing from inventory")
}

func TestParseWorktreeListHandlesDetachedLockedAndPrunableRecords(t *testing.T) {
	parsed, err := parseWorktreeList("worktree /repo\x00HEAD abc\x00branch refs/heads/main\x00\x00" +
		"worktree /repo detached\x00HEAD def\x00detached\x00locked keep\x00\x00" +
		"worktree /gone\x00HEAD ghi\x00prunable gitdir file is missing\x00\x00")
	require.NoError(t, err)
	require.Len(t, parsed, 3)
	assert.Equal(t, "main", parsed[0].Branch)
	assert.True(t, parsed[1].Detached)
	assert.True(t, parsed[1].Locked)
	assert.Equal(t, "keep", parsed[1].LockReason)
	assert.True(t, parsed[2].Prunable)
	assert.Equal(t, "gitdir file is missing", parsed[2].PrunableReason)
}

func TestWorktreeServicePlansAndAddsNewBranch(t *testing.T) {
	root := worktreeTestRepository(t)
	target := filepath.Join(t.TempDir(), "new-worktree")
	service := NewWorktreeService(root)
	options := WorktreeAddOptions{Path: target, Branch: "feature/new", From: "HEAD", NewBranch: true}

	plan, err := service.PlanAdd(context.Background(), options)
	require.NoError(t, err)
	assert.Equal(t, "add", plan.Operation)
	assert.Equal(t, "planned", plan.State)
	assert.True(t, plan.DryRun)
	assert.NoDirExists(t, target)

	result, err := service.Add(context.Background(), options)
	require.NoError(t, err)
	assert.Equal(t, "completed", result.State)
	assert.False(t, result.DryRun)
	assert.Equal(t, "feature/new", strings.TrimSpace(runGit(t, target, "branch", "--show-current")))
}

func TestWorktreeServiceAddsAnExistingLocalBranch(t *testing.T) {
	root := worktreeTestRepository(t)
	runGit(t, root, "branch", "feature/existing")
	target := filepath.Join(t.TempDir(), "existing-worktree")
	result, err := NewWorktreeService(root).Add(context.Background(), WorktreeAddOptions{Path: target, Branch: "feature/existing"})
	require.NoError(t, err)
	assert.Equal(t, "completed", result.State)
	assert.Equal(t, "feature/existing", strings.TrimSpace(runGit(t, target, "branch", "--show-current")))
}

func TestWorktreeServicePlansAndRemovesCleanLinkedWorktree(t *testing.T) {
	root := worktreeTestRepository(t)
	target := filepath.Join(t.TempDir(), "feature")
	runGit(t, root, "branch", "feature/remove")
	runGit(t, root, "worktree", "add", "--quiet", target, "feature/remove")
	service := NewWorktreeService(root)

	plan, err := service.PlanRemove(context.Background(), target)
	require.NoError(t, err)
	assert.Equal(t, "remove", plan.Operation)
	assert.Equal(t, "planned", plan.State)
	assert.True(t, plan.DryRun)
	assert.DirExists(t, target)

	result, err := service.Remove(context.Background(), target)
	require.NoError(t, err)
	assert.Equal(t, "completed", result.State)
	assert.NoDirExists(t, target)
	inventory, err := service.List(context.Background(), 10)
	require.NoError(t, err)
	assert.Len(t, inventory.Worktrees, 1)
}

func TestWorktreeServiceRejectsUnsafeAddAndRemovePlans(t *testing.T) {
	root := worktreeTestRepository(t)
	service := NewWorktreeService(root)
	target := filepath.Join(t.TempDir(), "target")
	options := WorktreeAddOptions{Path: target, Branch: "main", NewBranch: true}
	_, err := service.PlanAdd(context.Background(), options)
	assert.ErrorContains(t, err, "already exists")

	options = WorktreeAddOptions{Path: target, Branch: "bad branch", NewBranch: true}
	_, err = service.PlanAdd(context.Background(), options)
	assert.ErrorContains(t, err, "invalid branch")

	options = WorktreeAddOptions{Path: filepath.Join(t.TempDir(), "missing", "child"), Branch: "feature/new", NewBranch: true}
	_, err = service.PlanAdd(context.Background(), options)
	assert.ErrorContains(t, err, "parent directory")

	_, err = service.PlanRemove(context.Background(), root)
	assert.ErrorContains(t, err, "main worktree")
	_, err = service.PlanRemove(context.Background(), filepath.Join(t.TempDir(), "not-registered"))
	assert.ErrorContains(t, err, "not registered")

	linked := filepath.Join(t.TempDir(), "current-linked")
	runGit(t, root, "worktree", "add", "--quiet", "-b", "feature/current", linked)
	_, err = NewWorktreeService(linked).PlanRemove(context.Background(), linked)
	assert.ErrorContains(t, err, "selected checkout")
}

func TestWorktreeServiceValidatesAddOptionsAndBranchSelection(t *testing.T) {
	root := worktreeTestRepository(t)
	service := NewWorktreeService(root)
	target := filepath.Join(t.TempDir(), "target")
	tests := []struct {
		name    string
		options WorktreeAddOptions
		want    string
	}{
		{name: "empty path", options: WorktreeAddOptions{Branch: "feature/new", NewBranch: true}, want: "path must not be empty"},
		{name: "empty branch", options: WorktreeAddOptions{Path: target, NewBranch: true}, want: "select a local branch"},
		{name: "option like branch", options: WorktreeAddOptions{Path: target, Branch: "-f", NewBranch: true}, want: "invalid branch name"},
		{name: "option like start point", options: WorktreeAddOptions{Path: target, Branch: "feature/new", From: "-f", NewBranch: true}, want: "invalid start point"},
		{name: "from without new branch", options: WorktreeAddOptions{Path: target, Branch: "main", From: "HEAD"}, want: "--from requires --new-branch"},
		{name: "missing local branch", options: WorktreeAddOptions{Path: target, Branch: "feature/missing"}, want: "does not exist"},
		{name: "invalid start point", options: WorktreeAddOptions{Path: target, Branch: "feature/new", From: "missing-ref", NewBranch: true}, want: "does not resolve to a commit"},
		{name: "new branch already exists", options: WorktreeAddOptions{Path: target, Branch: "main", NewBranch: true}, want: "already exists"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.PlanAdd(context.Background(), test.options)
			assert.ErrorContains(t, err, test.want)
		})
	}
}

func TestWorktreeServiceRejectsBareAddAndNonDirectoryParent(t *testing.T) {
	bare := t.TempDir()
	runGit(t, bare, "init", "--quiet", "--bare", bare)
	_, err := NewWorktreeService(bare).PlanAdd(context.Background(), WorktreeAddOptions{
		Path: filepath.Join(t.TempDir(), "linked"), Branch: "feature/new", NewBranch: true,
	})
	assert.ErrorContains(t, err, "non-bare Git checkout")

	root := worktreeTestRepository(t)
	file := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o600))
	_, err = NewWorktreeService(root).PlanAdd(context.Background(), WorktreeAddOptions{
		Path: filepath.Join(file, "child"), Branch: "feature/new", NewBranch: true,
	})
	assert.ErrorContains(t, err, "inspect worktree path")
}

func TestWorktreeServiceRejectsCheckedOutBranchDirtyAndLockedRemoval(t *testing.T) {
	root := worktreeTestRepository(t)
	target := filepath.Join(t.TempDir(), "feature")
	runGit(t, root, "worktree", "add", "--quiet", "-b", "feature/active", target)
	service := NewWorktreeService(root)

	_, err := service.PlanAdd(context.Background(), WorktreeAddOptions{Path: filepath.Join(t.TempDir(), "another"), Branch: "feature/active"})
	assert.ErrorContains(t, err, "checked out")

	require.NoError(t, os.WriteFile(filepath.Join(target, "dirty.txt"), []byte("dirty"), 0o600))
	_, err = service.PlanRemove(context.Background(), target)
	assert.ErrorContains(t, err, "dirty")
	require.NoError(t, os.Remove(filepath.Join(target, "dirty.txt")))
	runGit(t, root, "worktree", "lock", target)
	_, err = service.PlanRemove(context.Background(), target)
	assert.ErrorContains(t, err, "locked")
}

func TestWorktreeServiceFindsBranchCheckout(t *testing.T) {
	root := worktreeTestRepository(t)
	target := filepath.Join(t.TempDir(), "feature")
	runGit(t, root, "worktree", "add", "--quiet", "-b", "feature/active", target)
	path, err := NewWorktreeService(root).BranchPath(context.Background(), "feature/active")
	require.NoError(t, err)
	assert.Equal(t, target, path)
	path, err = NewWorktreeService(root).BranchPath(context.Background(), "missing")
	require.NoError(t, err)
	assert.Empty(t, path)
}

func TestWorktreeServiceFindsBranchInAnotherCheckout(t *testing.T) {
	root := worktreeTestRepository(t)
	linked := filepath.Join(t.TempDir(), "linked")
	runGit(t, root, "worktree", "add", "--quiet", "-b", "feature/active", linked)

	path, err := NewWorktreeService(root).OtherBranchPath(context.Background(), "feature/active")
	require.NoError(t, err)
	assert.Equal(t, linked, path)

	path, err = NewWorktreeService(linked).OtherBranchPath(context.Background(), "main")
	require.NoError(t, err)
	assert.Equal(t, root, path)

	path, err = NewWorktreeService(root).OtherBranchPath(context.Background(), "not-checked-out")
	require.NoError(t, err)
	assert.Empty(t, path)
}

func TestWorktreeServiceRejectsPrunableWorktreeRemoval(t *testing.T) {
	root := worktreeTestRepository(t)
	target := filepath.Join(t.TempDir(), "gone")
	runGit(t, root, "worktree", "add", "--quiet", "-b", "feature/gone", target)
	require.NoError(t, os.RemoveAll(target))

	_, err := NewWorktreeService(root).PlanRemove(context.Background(), target)
	assert.ErrorContains(t, err, "prunable")
}

func TestParseWorktreeListRejectsMalformedRecords(t *testing.T) {
	_, err := parseWorktreeList("HEAD abc\x00")
	assert.ErrorContains(t, err, "appears before a worktree path")

	_, err = parseWorktreeList("worktree \x00HEAD abc\x00\x00")
	assert.ErrorContains(t, err, "missing its path")

	_, err = parseWorktreeList("worktree /repo\x00\x00")
	assert.ErrorContains(t, err, "missing its HEAD")
}

func TestWorktreeHelpersCoverEmptyInventoryAndRepositoryErrors(t *testing.T) {
	parsed, err := parseWorktreeList("")
	require.NoError(t, err)
	assert.Empty(t, parsed)
	assert.Nil(t, currentEntry(nil))

	service := NewWorktreeService(t.TempDir())
	_, err = service.BranchPath(context.Background(), "main")
	assert.ErrorContains(t, err, "list Git worktrees")
	_, err = service.OtherBranchPath(context.Background(), "main")
	assert.ErrorContains(t, err, "list Git worktrees")
	_, err = service.branchPaths(context.Background())
	assert.ErrorContains(t, err, "list Git worktrees")
	_, err = service.PlanAdd(context.Background(), WorktreeAddOptions{Path: "linked", Branch: "feature/new", NewBranch: true})
	assert.ErrorContains(t, err, "list Git worktrees")
	_, err = service.Add(context.Background(), WorktreeAddOptions{Path: "linked", Branch: "feature/new", NewBranch: true})
	assert.ErrorContains(t, err, "list Git worktrees")
	_, err = service.PlanRemove(context.Background(), "linked")
	assert.ErrorContains(t, err, "list Git worktrees")
	_, err = service.Remove(context.Background(), "linked")
	assert.ErrorContains(t, err, "list Git worktrees")
	_, err = service.removalTarget(context.Background(), " ")
	assert.ErrorContains(t, err, "path must not be empty")
}

func TestValidateNewWorktreePathReportsSymlinkResolutionErrors(t *testing.T) {
	loop := filepath.Join(t.TempDir(), "loop")
	require.NoError(t, os.Symlink(loop, loop))
	err := validateNewWorktreePath(filepath.Join(loop, "child"))
	assert.ErrorContains(t, err, "inspect worktree path")
}

func TestWorktreeMutationsReportGitExecutionFailures(t *testing.T) {
	root := worktreeTestRepository(t)
	addTarget := filepath.Join(t.TempDir(), "add-fails")
	removeRoot := worktreeTestRepository(t)
	removeTarget := filepath.Join(t.TempDir(), "remove-fails")
	runGit(t, removeRoot, "worktree", "add", "--quiet", "-b", "feature/remove-fails", removeTarget)

	installGitFailureWrapper(t, "add")
	_, err := NewWorktreeService(root).Add(context.Background(), WorktreeAddOptions{
		Path: addTarget, Branch: "feature/add-fails", NewBranch: true,
	})
	assert.ErrorContains(t, err, "add worktree")
	assert.NoDirExists(t, addTarget)

	installGitFailureWrapper(t, "remove")
	_, err = NewWorktreeService(removeRoot).Remove(context.Background(), removeTarget)
	assert.ErrorContains(t, err, "remove worktree")
	assert.DirExists(t, removeTarget)
}

func TestWorktreeBranchLookupSurfacesUnexpectedShowRefFailure(t *testing.T) {
	root := worktreeTestRepository(t)
	installGitFailureWrapper(t, "show-ref")
	_, err := NewWorktreeService(root).PlanAdd(context.Background(), WorktreeAddOptions{
		Path: filepath.Join(t.TempDir(), "linked"), Branch: "feature/new", NewBranch: true,
	})
	assert.ErrorContains(t, err, "check local branch")
}

func installGitFailureWrapper(t *testing.T, command string) {
	t.Helper()
	gitPath, err := exec.LookPath("git")
	require.NoError(t, err)
	binDirectory := t.TempDir()
	wrapper := "#!/bin/sh\n"
	switch command {
	case "add", "remove":
		wrapper += "if [ \"$1\" = \"worktree\" ] && [ \"$2\" = \"" + command + "\" ]; then echo simulated failure >&2; exit 1; fi\n"
	default:
		wrapper += "if [ \"$1\" = \"" + command + "\" ]; then echo simulated failure >&2; exit 2; fi\n"
	}
	quotedGitPath := "'" + strings.ReplaceAll(gitPath, "'", "'\\''") + "'"
	wrapper += "exec " + quotedGitPath + " \"$@\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(binDirectory, "git"), []byte(wrapper), 0o700))
	t.Setenv("PATH", binDirectory+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestWorktreeServiceListsBareRepository(t *testing.T) {
	bare := t.TempDir()
	runGit(t, bare, "init", "--quiet", "--bare", bare)
	inventory, err := NewWorktreeService(bare).List(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, inventory.Worktrees, 1)
	assert.True(t, inventory.Worktrees[0].Bare)
	assert.True(t, inventory.Worktrees[0].Main)
	assert.Equal(t, "not_applicable", inventory.Worktrees[0].StatusState)
}

func worktreeTestRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "--quiet", "-b", "main", root)
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	runGit(t, root, "commit", "--quiet", "--allow-empty", "-m", "initial")
	return root
}
