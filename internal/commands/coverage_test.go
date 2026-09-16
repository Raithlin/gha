package commands

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/internal/branch"
	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/internal/review"
	"github.com/raithlin/gha/pkg/model"
)

type commandFailingWriter struct{}

func (commandFailingWriter) Write([]byte) (int, error) { return 0, errors.New("writer failed") }

func TestBranchCommandHelpersCoverUnavailableAndSafeStates(t *testing.T) {
	command := &cobra.Command{}
	command.SetContext(context.Background())
	command.Flags().String("format", "json", "")
	publication, format, err := prepareBranchPublication(command, nil, nil, "feature", "", "", true)
	assert.Nil(t, publication)
	assert.Equal(t, output.JSON, format)
	assert.ErrorContains(t, err, "not configured")

	command = &cobra.Command{}
	command.Flags().String("format", "invalid", "")
	_, _, err = prepareBranchPublication(command, nil, nil, "feature", "", "", true)
	assert.ErrorContains(t, err, "unsupported format")

	assert.ErrorContains(t, validateBranchPublication(&model.BranchPublication{}), "unavailable")
	denied := false
	assert.ErrorContains(t, validateBranchPublication(&model.BranchPublication{Permissions: model.ProviderSignal{State: "available"}, CanPush: &denied}), "denied")
	assert.Equal(t, "not_requested", targetState(false, "planned"))
	assert.Equal(t, "planned", targetState(true, "planned"))

	writer := git.NewBranchWriter(t.TempDir())
	switchTo, err := currentBranchDeleteSwitch(context.Background(), writer, "feature", false, "")
	require.NoError(t, err)
	assert.Empty(t, switchTo)
	_, err = requireRemoteDestructionSafety(command, nil, nil, "", "", "feature")
	assert.ErrorContains(t, err, "cannot verify")
}

func TestRefreshPublishedBranchFallsBackWhenInspectionFails(t *testing.T) {
	publication := &model.BranchPublication{
		Name:   "feature",
		Target: "origin/feature",
		Local:  &model.Branch{},
	}
	refreshPublishedBranch(context.Background(), publication, t.TempDir())
	assert.Equal(t, "origin/feature", publication.Local.Upstream)
	assert.Equal(t, "unavailable", publication.Local.DivergenceState)
}

func TestAgentFileHelpersPreservePermissionsAndRejectInvalidPaths(t *testing.T) {
	path := filepath.Join(t.TempDir(), "skill", "SKILL.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o600))
	require.NoError(t, writeFileAtomically(path, []byte("new")))
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "new", string(content))
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	blocked := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocked, []byte("not a directory"), 0o644))
	assert.Error(t, writeFileAtomically(filepath.Join(blocked, "child"), []byte("content")))
	_, err = withManagedGuidance([]byte("<!-- gha:begin -->"))
	assert.ErrorContains(t, err, "without <!-- gha:end -->")
}

func TestCommandValidationErrorsAreStructuredAndDoNotPanic(t *testing.T) {
	command := newBranchShowCmd(nil, nil)
	command.SetArgs([]string{"feature", "--format", "json"})
	err := command.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "branch inspection is not configured")

	command = newReviewCmd(nil, nil)
	command.SetArgs([]string{"zero", "--format", "json"})
	err = command.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid pull request number")

	command = newPRPrepareCmd(nil, nil)
	command.SetArgs([]string{"--title", "title", "--head", "feature"})
	err = command.Execute()
	assert.ErrorContains(t, err, "not configured")
}

func TestReleaseCommandValidationAndTimestampFormats(t *testing.T) {
	location := time.FixedZone("test", 2*60*60)
	for _, value := range []string{"2026-09-01T09:30:00Z", "2026-09-01T09:30:00", "2026-09-01T09:30", "2026-09-01"} {
		timestamp, err := parseReleaseSince(value, location)
		require.NoError(t, err, value)
		assert.False(t, timestamp.IsZero())
	}
	_, err := parseReleaseSince("not-a-date", time.UTC)
	assert.ErrorContains(t, err, "invalid --since")

	command := newReleaseCreateNotesCmd(nil, nil)
	command.SetArgs([]string{"--since", "2026-09-01", "--limit", "0", "--format", "json"})
	err = command.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "limit must be between 1 and 100")
}

func TestRemoteDestructionSafetyExplainsEveryBlockedProviderState(t *testing.T) {
	checkout, _ := mutationRepository(t)
	command := &cobra.Command{}
	command.SetContext(context.Background())
	resolver := git.NewRepositoryResolver("acme/project")
	canPush, denied := true, false
	tests := []struct {
		name   string
		safety model.BranchSafety
		want   string
	}{
		{"permission unavailable", model.BranchSafety{}, "permission is unavailable or denied"},
		{"permission denied", model.BranchSafety{Permissions: model.ProviderSignal{State: "available"}, CanPush: &denied}, "permission is unavailable or denied"},
		{"default unavailable", model.BranchSafety{Permissions: model.ProviderSignal{State: "available"}, CanPush: &canPush}, "default-branch status is unavailable"},
		{"origin default", model.BranchSafety{Permissions: model.ProviderSignal{State: "available"}, CanPush: &canPush, DefaultBranch: model.ProviderSignal{State: "available"}, IsDefault: &canPush}, "origin default branch"},
		{"protection unavailable", model.BranchSafety{Permissions: model.ProviderSignal{State: "available"}, CanPush: &canPush, DefaultBranch: model.ProviderSignal{State: "available"}, IsDefault: &denied}, "protection status is unavailable"},
		{"protected", model.BranchSafety{Permissions: model.ProviderSignal{State: "available"}, CanPush: &canPush, DefaultBranch: model.ProviderSignal{State: "available"}, IsDefault: &denied, Protection: model.ProviderSignal{State: "available"}, Protected: &canPush}, "protected origin branch"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := branch.NewService(git.NewBranchLister(checkout), mutationSafetyProvider{safety: test.safety})
			_, err := requireRemoteDestructionSafety(command, service, resolver, checkout, "acme/project", "main")
			assert.ErrorContains(t, err, test.want)
		})
	}

	safe := model.BranchSafety{Permissions: model.ProviderSignal{State: "available"}, CanPush: &canPush, DefaultBranch: model.ProviderSignal{State: "available"}, DefaultBranchName: "main", IsDefault: &denied, Protection: model.ProviderSignal{State: "available"}, Protected: &denied}
	service := branch.NewService(git.NewBranchLister(checkout), mutationSafetyProvider{safety: safe})
	defaultBranch, err := requireRemoteDestructionSafety(command, service, resolver, checkout, "acme/project", "main")
	require.NoError(t, err)
	assert.Equal(t, "main", defaultBranch)
}

func TestAgentUninstallReportsUnremovableManagedTargets(t *testing.T) {
	root := t.TempDir()
	guidanceDirectory := filepath.Join(root, "AGENTS.md")
	require.NoError(t, os.Mkdir(guidanceDirectory, 0o755))
	_, err := uninstallAgentGuidance(agentInstallation{name: "Codex", instructionsPath: guidanceDirectory, skillPath: filepath.Join(root, "missing", "SKILL.md")})
	assert.ErrorContains(t, err, "read Codex guidance")

	skillDirectory := filepath.Join(root, "skills", "gha", "SKILL.md")
	require.NoError(t, os.MkdirAll(skillDirectory, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDirectory, "keep"), []byte("content"), 0o644))
	_, err = removeAgentSkill(skillDirectory)
	assert.Error(t, err)
}

func TestReleaseListingAndBranchCommandsValidateBeforeAnyWrite(t *testing.T) {
	command := newReleasesCmd(nil, nil)
	command.SetArgs([]string{"--limit", "101"})
	err := command.Execute()
	assert.ErrorContains(t, err, "limit must be between 1 and 100")

	command = newReleasesCmd(review.NewService(nil), git.NewRepositoryResolver("acme/project"))
	command.SetArgs([]string{"--repo", "acme/project"})
	err = command.Execute()
	assert.ErrorContains(t, err, "release listing is not configured")

	branchCommand := newBranchCreateCmd()
	branchCommand.SetArgs([]string{"feature", "--format", "unknown"})
	err = branchCommand.Execute()
	assert.ErrorContains(t, err, "unsupported format")

	branchCommand = newBranchRenameCmd(nil, nil)
	branchCommand.SetArgs([]string{"feature", "new", "--origin", "--dry-run"})
	require.NoError(t, branchCommand.Execute())
}

func TestAgentCommandErrorPathsKeepWritesGuarded(t *testing.T) {
	command := newAgentInstallCmd()
	command.SetIn(bytes.NewBuffer(nil))
	command.SetArgs([]string{"--dry-run"})
	err := command.Execute()
	assert.ErrorContains(t, err, "read agent selection")

	command = newAgentInstallCmd()
	command.SetArgs([]string{"--agent", "invalid", "--dry-run"})
	err = command.Execute()
	assert.ErrorContains(t, err, "invalid agent")

	command = newAgentInstallCmd()
	command.SetOut(commandFailingWriter{})
	command.SetArgs([]string{"--agent", "codex", "--dry-run"})
	err = command.Execute()
	assert.ErrorContains(t, err, "writer failed")

	target := agentInstallation{name: "Codex", instructionsPath: filepath.Join(t.TempDir(), "AGENTS.md"), skillPath: filepath.Join(t.TempDir(), "SKILL.md")}
	err = uninstallAgentTarget(commandFailingWriter{}, target, true)
	assert.ErrorContains(t, err, "writer failed")
}
