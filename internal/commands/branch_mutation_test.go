package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/internal/branch"
	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/pkg/model"
)

func TestBranchCreateDryRunReportsBothTargetsWithoutWriting(t *testing.T) {
	checkout, _ := mutationRepository(t)
	command := newBranchCreateCmd(nil)
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"feature", "--publish", "--dry-run", "--path", checkout, "--format", "json"})

	require.NoError(t, command.Execute())
	var result model.BranchMutation
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	assert.Equal(t, model.BranchMutationSchemaVersion, result.SchemaVersion)
	assert.True(t, result.DryRun)
	assert.Equal(t, "planned", result.Local)
	assert.Equal(t, "planned", result.Origin)
	assertBranchMissing(t, checkout, "feature")
}

func TestBranchCreateDryRunRejectsInvalidRefsAndCollisions(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(*testing.T, string, string)
		args         []string
		want         string
		branchExists bool
	}{
		{
			name: "missing start ref",
			args: []string{"feature", "--from", "missing", "--dry-run", "--format", "json"},
			want: "start point",
		},
		{
			name: "local name collision",
			setup: func(t *testing.T, checkout, _ string) {
				runMutationGit(t, checkout, "branch", "feature")
			},
			args:         []string{"feature", "--dry-run", "--format", "json"},
			want:         "already exists locally",
			branchExists: true,
		},
		{
			name: "published origin collision",
			setup: func(t *testing.T, checkout, _ string) {
				runMutationGit(t, checkout, "branch", "feature")
				runMutationGit(t, checkout, "push", "origin", "feature")
				runMutationGit(t, checkout, "branch", "-D", "feature")
			},
			args: []string{"feature", "--publish", "--dry-run", "--format", "json"},
			want: "already exists on origin",
		},
		{
			name: "publish without origin",
			setup: func(t *testing.T, checkout, _ string) {
				runMutationGit(t, checkout, "remote", "remove", "origin")
			},
			args: []string{"feature", "--publish", "--dry-run", "--format", "json"},
			want: "read origin branch refs",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checkout, _ := mutationRepository(t)
			if test.setup != nil {
				test.setup(t, checkout, "")
			}
			command := newBranchCreateCmd(nil)
			var output bytes.Buffer
			command.SetOut(&output)
			command.SetArgs(append(test.args, "--path", checkout))

			err := command.Execute()

			require.Error(t, err)
			assert.ErrorContains(t, err, test.want)
			assert.Empty(t, output.String(), "invalid dry runs must not return an executable plan")
			if test.branchExists {
				assertBranchExists(t, checkout, "feature")
			} else {
				assertBranchMissing(t, checkout, "feature")
			}
		})
	}
}

func TestBranchCreatePublishesByDefaultWhenRequested(t *testing.T) {
	checkout, remote := mutationRepository(t)
	command := newBranchCreateCmd(nil)
	command.SetArgs([]string{"feature", "--publish", "--path", checkout})
	require.NoError(t, command.Execute())
	assertBranchExists(t, checkout, "feature")
	assertBranchExists(t, remote, "feature")
}

func TestBranchCreatePublishesToProviderNeutralOrigin(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "remote", "set-url", "origin", remote)
	command := newBranchCreateCmd(nil)
	command.SetArgs([]string{"feature", "--publish", "--path", checkout})
	require.NoError(t, command.Execute())
	assertBranchExists(t, remote, "feature")
}

func TestBranchPublishPreflightsAnUnpublishedLocalBranchWithoutWriting(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")

	safety := model.BranchSafety{
		Permissions: model.ProviderSignal{State: "available"},
		CanPush:     boolPointer(true),
	}
	command := newBranchPublishCmd(branch.NewService(git.NewBranchLister(checkout), mutationSafetyProvider{safety: safety}), git.NewRepositoryResolver("acme/project"))
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"feature", "--dry-run", "--repo", "acme/project", "--path", checkout, "--format", "json"})

	require.NoError(t, command.Execute())
	var result model.BranchPublication
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	assert.Equal(t, model.BranchPublicationSchemaVersion, result.SchemaVersion)
	assert.Equal(t, "origin/feature", result.Target)
	assert.Equal(t, "planned", result.Publication)
	assert.True(t, result.DryRun)
	require.NotNil(t, result.Local)
	assert.Empty(t, result.Local.Upstream)
	assert.Equal(t, "not_tracked", result.Local.DivergenceState)
	assert.Equal(t, "available", result.Permissions.State)
	assert.True(t, *result.CanPush)
	assertBranchMissing(t, remote, "feature")
}

func TestBranchPublishRunsByDefault(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	safety := model.BranchSafety{Permissions: model.ProviderSignal{State: "available"}, CanPush: boolPointer(true)}
	command := newBranchPublishCmd(branch.NewService(git.NewBranchLister(checkout), mutationSafetyProvider{safety: safety}), git.NewRepositoryResolver("acme/project"))
	command.SetArgs([]string{"feature", "--repo", "acme/project", "--path", checkout})

	require.NoError(t, command.Execute())
	assertBranchExists(t, remote, "feature")
}

func TestBranchPublishSetsUpstreamAfterSafePreflight(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	safety := model.BranchSafety{Permissions: model.ProviderSignal{State: "available"}, CanPush: boolPointer(true)}
	command := newBranchPublishCmd(branch.NewService(git.NewBranchLister(checkout), mutationSafetyProvider{safety: safety}), git.NewRepositoryResolver("acme/project"))
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"feature", "--repo", "acme/project", "--path", checkout, "--format", "json"})

	require.NoError(t, command.Execute())
	var result model.BranchPublication
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	assert.Equal(t, "completed", result.Publication)
	assertBranchExists(t, remote, "feature")
	assert.Equal(t, "origin/feature", upstreamMutationBranch(t, checkout, "feature"))
}

func TestBranchPublishUsesGitPushWhenProviderPermissionIsUnavailable(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	safety := model.BranchSafety{Permissions: model.ProviderSignal{State: "unavailable", Message: "token rejected"}}
	command := newBranchPublishCmd(branch.NewService(git.NewBranchLister(checkout), mutationSafetyProvider{safety: safety}), git.NewRepositoryResolver("acme/project"))
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"feature", "--repo", "acme/project", "--path", checkout, "--format", "json"})

	require.NoError(t, command.Execute())

	var result model.BranchPublication
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	assert.Equal(t, "completed", result.Publication)
	assert.Equal(t, "unavailable", result.Permissions.State)
	assertBranchExists(t, remote, "feature")
}

func TestBranchPublishRejectsAnAlreadyTrackedBranch(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	runMutationGit(t, checkout, "push", "-u", "origin", "feature")
	safety := model.BranchSafety{Permissions: model.ProviderSignal{State: "available"}, CanPush: boolPointer(true)}
	command := newBranchPublishCmd(branch.NewService(git.NewBranchLister(checkout), mutationSafetyProvider{safety: safety}), git.NewRepositoryResolver("acme/project"))
	command.SetArgs([]string{"feature", "--dry-run", "--repo", "acme/project", "--path", checkout})

	err := command.Execute()

	require.Error(t, err)
	assert.ErrorContains(t, err, "already tracks origin/feature")
	assertBranchExists(t, remote, "feature")
}

func TestOriginBranchWritesRejectRepositoryMismatchBeforeMutation(t *testing.T) {
	for _, test := range []struct {
		name    string
		args    []string
		command func(*branch.Service, *git.RepositoryResolver) interface {
			SetArgs([]string)
			Execute() error
		}
	}{
		{"create publish configured", []string{"new", "--publish"}, func(_ *branch.Service, r *git.RepositoryResolver) interface {
			SetArgs([]string)
			Execute() error
		} {
			return newBranchCreateCmd(r)
		}},
		{"publish explicit", []string{"feature", "--repo", "other/repo"}, func(s *branch.Service, r *git.RepositoryResolver) interface {
			SetArgs([]string)
			Execute() error
		} {
			return newBranchPublishCmd(s, r)
		}},
		{"rename explicit", []string{"feature", "renamed", "--origin", "--repo", "other/repo"}, func(s *branch.Service, r *git.RepositoryResolver) interface {
			SetArgs([]string)
			Execute() error
		} {
			return newBranchRenameCmd(s, r)
		}},
		{"rename forced configured", []string{"feature", "renamed", "--origin", "--force"}, func(s *branch.Service, r *git.RepositoryResolver) interface {
			SetArgs([]string)
			Execute() error
		} {
			return newBranchRenameCmd(s, r)
		}},
		{"delete explicit", []string{"feature", "--local", "--origin", "--repo", "other/repo"}, func(s *branch.Service, r *git.RepositoryResolver) interface {
			SetArgs([]string)
			Execute() error
		} {
			return newBranchDeleteCmd(s, r)
		}},
		{"delete forced configured", []string{"feature", "--local", "--origin", "--force"}, func(s *branch.Service, r *git.RepositoryResolver) interface {
			SetArgs([]string)
			Execute() error
		} {
			return newBranchDeleteCmd(s, r)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			checkout, remote := mutationRepository(t)
			runMutationGit(t, checkout, "branch", "feature")
			runMutationGit(t, checkout, "push", "origin", "feature")
			service := branch.NewService(git.NewBranchLister(checkout), mutationSafetyProvider{})
			resolver := git.NewRepositoryResolver("other/repo")
			command := test.command(service, resolver)
			command.SetArgs(append(test.args, "--path", checkout))
			err := command.Execute()
			require.ErrorContains(t, err, "does not match origin")
			assertBranchExists(t, checkout, "feature")
			assertBranchExists(t, remote, "feature")
			assertBranchMissing(t, checkout, "new")
			assertBranchMissing(t, checkout, "renamed")
			assertBranchMissing(t, remote, "renamed")
		})
	}
}

func TestBranchPublishRejectsMismatchedOriginPushURL(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	runMutationGit(t, checkout, "config", "remote.origin.pushurl", "git@github.com:other/project.git")
	service := branch.NewService(git.NewBranchLister(checkout), mutationSafetyProvider{})
	command := newBranchPublishCmd(service, git.NewRepositoryResolver("acme/project"))
	command.SetArgs([]string{"feature", "--path", checkout, "--dry-run"})
	assert.ErrorContains(t, command.Execute(), "does not match origin")
	assertBranchMissing(t, remote, "feature")
}

func TestBranchRenameOriginSafetyCanBeExplicitlyForced(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	runMutationGit(t, checkout, "push", "-u", "origin", "feature")

	command := newBranchRenameCmd(nil, nil)
	command.SetArgs([]string{"feature", "better", "--origin", "--path", checkout})
	err := command.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "safety")
	assertBranchExists(t, checkout, "feature")

	command = newBranchRenameCmd(nil, nil)
	command.SetArgs([]string{"feature", "better", "--origin", "--force", "--path", checkout})
	require.NoError(t, command.Execute())
	assertBranchMissing(t, checkout, "feature")
	assertBranchExists(t, checkout, "better")
	assertBranchMissing(t, remote, "feature")
	assertBranchExists(t, remote, "better")
}

func TestBranchRenameDryRunValidatesRefsOriginAndProviderSafety(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*testing.T, string, string)
		service *branch.Service
		args    []string
		want    string
	}{
		{
			name: "missing local source",
			args: []string{"missing", "renamed", "--dry-run"},
			want: "not found locally",
		},
		{
			name: "local destination collision",
			setup: func(t *testing.T, checkout, _ string) {
				runMutationGit(t, checkout, "branch", "feature")
				runMutationGit(t, checkout, "branch", "renamed")
			},
			args: []string{"feature", "renamed", "--dry-run"},
			want: "already exists locally",
		},
		{
			name: "missing origin source",
			setup: func(t *testing.T, checkout, _ string) {
				runMutationGit(t, checkout, "branch", "feature")
			},
			args: []string{"feature", "renamed", "--origin", "--dry-run", "--force"},
			want: "does not exist on origin",
		},
		{
			name: "origin destination collision",
			setup: func(t *testing.T, checkout, _ string) {
				runMutationGit(t, checkout, "branch", "feature")
				runMutationGit(t, checkout, "push", "origin", "feature")
				runMutationGit(t, checkout, "branch", "renamed")
				runMutationGit(t, checkout, "push", "origin", "renamed")
				runMutationGit(t, checkout, "branch", "-D", "renamed")
			},
			args: []string{"feature", "renamed", "--origin", "--dry-run", "--force"},
			want: "already exists on origin",
		},
		{
			name: "provider safety unavailable",
			setup: func(t *testing.T, checkout, _ string) {
				runMutationGit(t, checkout, "branch", "feature")
				runMutationGit(t, checkout, "push", "origin", "feature")
			},
			service: branch.NewService(nil, mutationSafetyProvider{}),
			args:    []string{"feature", "renamed", "--origin", "--dry-run", "--repo", "acme/project"},
			want:    "permission is unavailable or denied",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checkout, remote := mutationRepository(t)
			if test.setup != nil {
				test.setup(t, checkout, remote)
			}
			command := newBranchRenameCmd(test.service, git.NewRepositoryResolver(""))
			var output bytes.Buffer
			command.SetOut(&output)
			command.SetArgs(append(test.args, "--path", checkout, "--format", "json"))

			err := command.Execute()

			require.Error(t, err)
			assert.ErrorContains(t, err, test.want)
			assert.Empty(t, output.String(), "invalid dry runs must not return an executable plan")
			if test.name == "provider safety unavailable" {
				assertBranchExists(t, checkout, "feature")
				assertBranchExists(t, remote, "feature")
				assertBranchMissing(t, checkout, "renamed")
				assertBranchMissing(t, remote, "renamed")
			}
		})
	}
}

func TestBranchRenameDryRunPlansOnlyAfterLocalOriginAndProviderChecks(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	runMutationGit(t, checkout, "push", "origin", "feature")
	safety := model.BranchSafety{
		Permissions:       model.ProviderSignal{State: "available"},
		CanPush:           boolPointer(true),
		DefaultBranch:     model.ProviderSignal{State: "available"},
		DefaultBranchName: "main",
		IsDefault:         boolPointer(false),
		Protection:        model.ProviderSignal{State: "available"},
		Protected:         boolPointer(false),
	}
	service := branch.NewService(git.NewBranchLister(checkout), mutationSafetyProvider{safety: safety})
	command := newBranchRenameCmd(service, git.NewRepositoryResolver(""))
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"feature", "renamed", "--origin", "--repo", "acme/project", "--dry-run", "--path", checkout, "--format", "json"})

	require.NoError(t, command.Execute())
	var result model.BranchMutation
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	assert.True(t, result.DryRun)
	assert.Equal(t, "planned", result.Local)
	assert.Equal(t, "planned", result.Origin)
	assertBranchExists(t, checkout, "feature")
	assertBranchExists(t, remote, "feature")
	assertBranchMissing(t, checkout, "renamed")
	assertBranchMissing(t, remote, "renamed")
}

func TestBranchOriginDryRunsRejectRepositoryMismatchWithoutPlans(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	runMutationGit(t, checkout, "push", "origin", "feature")
	service := branch.NewService(git.NewBranchLister(checkout), mutationSafetyProvider{})
	command := newBranchRenameCmd(service, git.NewRepositoryResolver(""))
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"feature", "renamed", "--origin", "--repo", "other/repo", "--dry-run", "--path", checkout, "--format", "json"})

	err := command.Execute()

	require.ErrorContains(t, err, "does not match origin")
	assert.Empty(t, output.String())
	assertBranchExists(t, checkout, "feature")
	assertBranchExists(t, remote, "feature")
}

func TestBranchDeleteRequiresAnExplicitTargetAndSupportsBothTargets(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	runMutationGit(t, checkout, "push", "-u", "origin", "feature")

	command := newBranchDeleteCmd(nil, nil)
	command.SetArgs([]string{"feature", "--path", checkout})
	err := command.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "--local and/or --origin")
	assertBranchExists(t, checkout, "feature")

	command = newBranchDeleteCmd(nil, nil)
	command.SetArgs([]string{"feature", "--local", "--origin", "--force", "--path", checkout})
	require.NoError(t, command.Execute())
	assertBranchMissing(t, checkout, "feature")
	assertBranchMissing(t, remote, "feature")
}

func TestBranchDeleteOriginGuardsTheDefaultBranchBeforeWriting(t *testing.T) {
	checkout, remote := mutationRepository(t)
	safety := model.BranchSafety{
		Permissions:   model.ProviderSignal{State: "available"},
		CanPush:       boolPointer(true),
		DefaultBranch: model.ProviderSignal{State: "available"},
		IsDefault:     boolPointer(true),
		Protection:    model.ProviderSignal{State: "available"},
		Protected:     boolPointer(false),
	}
	service := branch.NewService(git.NewBranchLister(checkout), mutationSafetyProvider{safety: safety})
	command := newBranchDeleteCmd(service, git.NewRepositoryResolver("acme/project"))
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"main", "--origin", "--repo", "acme/project", "--dry-run", "--path", checkout})

	err := command.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "default branch")
	assert.Empty(t, output.String())
	assertBranchExists(t, remote, "main")
}

func TestBranchDeleteCurrentMergedBranchSwitchesToDefaultBeforeDeleting(t *testing.T) {
	checkout, _ := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	runMutationGit(t, checkout, "switch", "feature")
	runMutationGit(t, checkout, "commit", "--allow-empty", "-m", "feature")
	runMutationGit(t, checkout, "switch", "main")
	runMutationGit(t, checkout, "merge", "--ff-only", "feature")
	runMutationGit(t, checkout, "switch", "feature")

	command := newBranchDeleteCmd(nil, nil)
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"feature", "--local", "--path", checkout, "--format", "json"})

	require.NoError(t, command.Execute())
	var result model.BranchMutation
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	assert.Equal(t, "main", result.CheckedOut)
	assert.Equal(t, "completed", result.Local)
	assertBranchMissing(t, checkout, "feature")
	assert.Equal(t, "main", currentMutationBranch(t, checkout))
}

func TestBranchDeleteCurrentDryRunReportsCheckoutWithoutChangingBranches(t *testing.T) {
	checkout, _ := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	runMutationGit(t, checkout, "switch", "feature")

	command := newBranchDeleteCmd(nil, nil)
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"feature", "--local", "--dry-run", "--path", checkout, "--format", "json"})

	require.NoError(t, command.Execute())
	var result model.BranchMutation
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	assert.True(t, result.DryRun)
	assert.Equal(t, "planned", result.Local)
	assert.Equal(t, "main", result.CheckedOut)
	assertBranchExists(t, checkout, "feature")
	assert.Equal(t, "feature", currentMutationBranch(t, checkout))
}

func TestBranchDeleteDryRunBlocksBranchCheckedOutInAnotherWorktree(t *testing.T) {
	checkout, _ := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	linked := filepath.Join(t.TempDir(), "linked")
	runMutationGit(t, checkout, "worktree", "add", "--quiet", linked, "feature")
	command := newBranchDeleteCmd(nil, nil)
	command.SetArgs([]string{"feature", "--local", "--dry-run", "--path", checkout, "--format", "json"})

	err := command.Execute()

	require.Error(t, err)
	assert.ErrorContains(t, err, "checked out")
	assert.ErrorContains(t, err, linked)
	assertBranchExists(t, checkout, "feature")
	assert.Equal(t, "main", currentMutationBranch(t, checkout))

	command = newBranchDeleteCmd(nil, nil)
	command.SetArgs([]string{"feature", "--local", "--path", checkout})
	err = command.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "checked out")
	assert.ErrorContains(t, err, linked)
	assertBranchExists(t, checkout, "feature")
}

func TestBranchDeleteDryRunValidatesRefsAndProviderSafety(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*testing.T, string, string)
		service *branch.Service
		args    []string
		want    string
	}{
		{
			name: "missing local branch",
			args: []string{"missing", "--local", "--dry-run"},
			want: "not found locally",
		},
		{
			name: "local branch is not merged",
			setup: func(t *testing.T, checkout, _ string) {
				runMutationGit(t, checkout, "branch", "feature")
				runMutationGit(t, checkout, "switch", "feature")
				runMutationGit(t, checkout, "commit", "--quiet", "--allow-empty", "-m", "unmerged")
				runMutationGit(t, checkout, "switch", "main")
			},
			args: []string{"feature", "--local", "--dry-run"},
			want: "not fully merged",
		},
		{
			name: "missing origin branch",
			args: []string{"missing", "--origin", "--dry-run", "--force"},
			want: "does not exist on origin",
		},
		{
			name: "provider safety unavailable",
			setup: func(t *testing.T, checkout, _ string) {
				runMutationGit(t, checkout, "branch", "feature")
				runMutationGit(t, checkout, "push", "origin", "feature")
			},
			service: branch.NewService(nil, mutationSafetyProvider{}),
			args:    []string{"feature", "--origin", "--dry-run", "--repo", "acme/project"},
			want:    "permission is unavailable or denied",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checkout, remote := mutationRepository(t)
			if test.setup != nil {
				test.setup(t, checkout, remote)
			}
			command := newBranchDeleteCmd(test.service, git.NewRepositoryResolver(""))
			var output bytes.Buffer
			command.SetOut(&output)
			command.SetArgs(append(test.args, "--path", checkout, "--format", "json"))

			err := command.Execute()

			require.Error(t, err)
			assert.ErrorContains(t, err, test.want)
			assert.Empty(t, output.String(), "invalid dry runs must not return an executable plan")
			if test.name == "provider safety unavailable" {
				assertBranchExists(t, checkout, "feature")
				assertBranchExists(t, remote, "feature")
			}
		})
	}
}

func TestBranchDeleteFailsClosedWhenWorktreeInventoryIsUnavailable(t *testing.T) {
	command := newBranchDeleteCmd(nil, nil)
	command.SetArgs([]string{"feature", "--local", "--dry-run", "--path", t.TempDir()})

	err := command.Execute()

	require.Error(t, err)
	assert.ErrorContains(t, err, "inspect local branches")
}

func TestBranchDeleteRefusesTheCurrentDefaultBranch(t *testing.T) {
	checkout, _ := mutationRepository(t)
	command := newBranchDeleteCmd(nil, nil)
	command.SetArgs([]string{"main", "--local", "--path", checkout})

	err := command.Execute()

	require.Error(t, err)
	assert.ErrorContains(t, err, "current default branch")
	assertBranchExists(t, checkout, "main")
}

type mutationSafetyProvider struct {
	safety model.BranchSafety
}

func (p mutationSafetyProvider) InspectBranchSafety(_ context.Context, _ model.RepositoryRef, _ string) (model.BranchSafety, error) {
	return p.safety, nil
}

func boolPointer(value bool) *bool { return &value }

func mutationRepository(t *testing.T) (string, string) {
	t.Helper()
	checkout := filepath.Join(t.TempDir(), "checkout")
	remote := filepath.Join(t.TempDir(), "origin.git")
	runMutationGit(t, "", "init", "--quiet", "--bare", remote)
	runMutationGit(t, "", "init", "--quiet", "-b", "main", checkout)
	runMutationGit(t, checkout, "config", "user.email", "test@example.com")
	runMutationGit(t, checkout, "config", "user.name", "Test User")
	runMutationGit(t, checkout, "commit", "--quiet", "--allow-empty", "-m", "initial")
	runMutationGit(t, checkout, "remote", "add", "origin", remote)
	runMutationGit(t, checkout, "push", "-u", "origin", "main")
	runMutationGit(t, checkout, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	// Keep provider identity realistic while routing writes to the local bare fixture.
	runMutationGit(t, checkout, "config", "url."+remote+".insteadOf", "git@github.com:acme/project.git")
	runMutationGit(t, checkout, "remote", "set-url", "origin", "git@github.com:acme/project.git")
	return checkout, remote
}

func assertBranchExists(t *testing.T, directory, name string) {
	t.Helper()
	command := exec.Command("git", "-C", directory, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	if filepath.Ext(directory) == ".git" {
		command = exec.Command("git", "--git-dir", directory, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	}
	require.NoError(t, command.Run())
}

func assertBranchMissing(t *testing.T, directory, name string) {
	t.Helper()
	command := exec.Command("git", "-C", directory, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	if filepath.Ext(directory) == ".git" {
		command = exec.Command("git", "--git-dir", directory, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	}
	require.Error(t, command.Run())
}

func runMutationGit(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, output)
}

func currentMutationBranch(t *testing.T, directory string) string {
	t.Helper()
	command := exec.Command("git", "branch", "--show-current")
	command.Dir = directory
	output, err := command.Output()
	require.NoError(t, err)
	return strings.TrimSpace(string(output))
}

func upstreamMutationBranch(t *testing.T, directory, name string) string {
	t.Helper()
	command := exec.Command("git", "for-each-ref", "--format=%(upstream:short)", "refs/heads/"+name)
	command.Dir = directory
	output, err := command.Output()
	require.NoError(t, err)
	return strings.TrimSpace(string(output))
}
