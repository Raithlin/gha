package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/output"
	"github.com/raithlin/gha/pkg/model"
)

func TestBranchMutationReportsPartialOriginFailure(t *testing.T) {
	for _, operation := range []string{"create", "rename", "delete"} {
		for _, format := range []string{"json", "yaml", "text"} {
			t.Run(operation+"/"+format, func(t *testing.T) {
				checkout, remote := mutationRepository(t)
				var command *cobra.Command
				var args []string
				switch operation {
				case "create":
					command = newBranchCreateCmd(nil)
					args = []string{"feature", "--publish"}
				case "rename":
					runMutationGit(t, checkout, "branch", "feature")
					runMutationGit(t, checkout, "push", "origin", "feature")
					command = newBranchRenameCmd(nil, git.NewRepositoryResolver(""))
					args = []string{"feature", "renamed", "--origin", "--force"}
				case "delete":
					runMutationGit(t, checkout, "switch", "-c", "feature")
					runMutationGit(t, checkout, "push", "origin", "feature")
					command = newBranchDeleteCmd(nil, git.NewRepositoryResolver(""))
					args = []string{"feature", "--local", "--origin", "--force"}
				}
				// Reject just one rename ref: an atomic push must preserve both remote refs.
				hook := "#!/bin/sh\nif [ \"$1\" = refs/heads/feature ]; then echo 'policy rejection' >&2; exit 1; fi\n"
				require.NoError(t, os.WriteFile(filepath.Join(remote, "hooks", "update"), []byte(hook), 0700))
				var stdout, stderr bytes.Buffer
				command.SetOut(&stdout)
				command.SetErr(&stderr)
				command.SetArgs(append(args, "--path", checkout, "--format", format))
				err := command.Execute()
				require.Error(t, err)
				assert.ErrorContains(t, err, "inspect")
				if format == "text" {
					assert.Contains(t, stdout.String(), "Local: completed")
					assert.Contains(t, stdout.String(), "Origin: failed")
					if operation == "delete" {
						assert.Contains(t, stdout.String(), "Checkout: completed")
					}
				} else {
					var result map[string]any
					if format == "json" {
						require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
					} else {
						require.NoError(t, yaml.Unmarshal(stdout.Bytes(), &result))
					}
					assert.Equal(t, "v1", result["schema_version"])
					assert.Equal(t, "completed", result["local"])
					assert.Equal(t, "failed", result["origin"])
					if operation == "delete" {
						assert.Equal(t, "main", result["checked_out"])
						assert.Equal(t, "completed", result["checkout_state"])
					}
					assert.Contains(t, stderr.String(), "branch_mutation_failed")
					assert.Contains(t, stderr.String(), "policy rejection")
				}
				switch operation {
				case "create":
					assertBranchExists(t, checkout, "feature")
					assertBranchMissing(t, remote, "feature")
				case "rename":
					assertBranchMissing(t, checkout, "feature")
					assertBranchExists(t, checkout, "renamed")
					assertBranchExists(t, remote, "feature")
					assertBranchMissing(t, remote, "renamed")
				case "delete":
					assertBranchMissing(t, checkout, "feature")
					assertBranchExists(t, remote, "feature")
					assert.Equal(t, "main", currentMutationBranch(t, checkout))
				}
			})
		}
	}
}

func TestBranchRenameReportsUnsupportedAtomicPush(t *testing.T) {
	checkout, remote := mutationRepository(t)
	runMutationGit(t, checkout, "branch", "feature")
	runMutationGit(t, checkout, "push", "origin", "feature")
	runMutationGit(t, remote, "config", "receive.advertiseAtomic", "false")
	command := newBranchRenameCmd(nil, nil)
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	command.SetArgs([]string{"feature", "renamed", "--origin", "--force", "--path", checkout, "--format", "json"})
	require.Error(t, command.Execute())
	var result model.BranchMutation
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "completed", result.Local)
	assert.Equal(t, "failed", result.Origin)
	assert.Contains(t, stderr.String(), "does not support --atomic push")
	assertBranchExists(t, checkout, "renamed")
	assertBranchMissing(t, checkout, "feature")
	assertBranchExists(t, remote, "feature")
	assertBranchMissing(t, remote, "renamed")
}

func TestBranchMutationFailurePreservesCauseWhenResultCannotBeWritten(t *testing.T) {
	command := &cobra.Command{}
	var stderr bytes.Buffer
	command.SetOut(commandFailingWriter{})
	command.SetErr(&stderr)
	cause := errors.New("origin rejected the push")
	err := renderBranchMutationFailure(command, output.JSON, &model.BranchMutation{Local: "completed", Origin: "failed"}, cause)
	require.ErrorIs(t, err, cause)
	assert.ErrorContains(t, err, "write branch mutation result: writer failed")
	assert.True(t, IsReportedError(err))
	var diagnostic model.CommandError
	require.NoError(t, json.Unmarshal(stderr.Bytes(), &diagnostic))
	assert.Equal(t, "branch_mutation_failed", diagnostic.Code)
	assert.Contains(t, diagnostic.Message, cause.Error())
	assert.Contains(t, diagnostic.Message, "writer failed")
}

func TestBranchDeleteReportsCheckoutBeforeLocalFailure(t *testing.T) {
	checkout, _ := mutationRepository(t)
	runMutationGit(t, checkout, "switch", "-c", "feature")
	// The switch completes, then a concurrent lock prevents local deletion.
	hook := "#!/bin/sh\ntouch \"$(git rev-parse --git-dir)/refs/heads/feature.lock\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(checkout, ".git", "hooks", "post-checkout"), []byte(hook), 0700))
	command := newBranchDeleteCmd(nil, nil)
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	command.SetArgs([]string{"feature", "--local", "--path", checkout, "--format", "json"})
	require.Error(t, command.Execute())
	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "main", result["checked_out"])
	assert.Equal(t, "completed", result["checkout_state"])
	assert.Equal(t, "failed", result["local"])
	assert.Equal(t, "not_requested", result["origin"])
	assert.Equal(t, "main", currentMutationBranch(t, checkout))
	assertBranchExists(t, checkout, "feature")
	assert.Contains(t, stderr.String(), "delete local branch")
}

func TestBranchMutationReportsFirstEffectFailure(t *testing.T) {
	for _, operation := range []string{"create", "rename", "delete", "switch"} {
		t.Run(operation, func(t *testing.T) {
			checkout, remote := mutationRepository(t)
			var command *cobra.Command
			var args []string
			var lock string
			switch operation {
			case "create":
				command = newBranchCreateCmd(nil)
				args = []string{"feature", "--publish"}
				lock = "refs/heads/feature.lock"
			case "rename":
				runMutationGit(t, checkout, "branch", "feature")
				runMutationGit(t, checkout, "push", "origin", "feature")
				command = newBranchRenameCmd(nil, nil)
				args = []string{"feature", "renamed", "--origin", "--force"}
				lock = "refs/heads/renamed.lock"
			default:
				runMutationGit(t, checkout, "switch", "-c", "feature")
				runMutationGit(t, checkout, "push", "origin", "feature")
				command = newBranchDeleteCmd(nil, nil)
				args = []string{"feature", "--local", "--origin", "--force"}
				if operation == "switch" {
					lock = "index.lock"
				} else {
					runMutationGit(t, checkout, "switch", "main")
					lock = "refs/heads/feature.lock"
				}
			}
			require.NoError(t, os.WriteFile(filepath.Join(checkout, ".git", lock), nil, 0600))
			var stdout, stderr bytes.Buffer
			command.SetOut(&stdout)
			command.SetErr(&stderr)
			command.SetArgs(append(args, "--path", checkout, "--format", "json"))
			require.Error(t, command.Execute())
			var result map[string]any
			require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
			assert.Equal(t, "planned", result["origin"], "origin was not attempted")
			if operation == "switch" {
				assert.Equal(t, "failed", result["checkout_state"])
				assert.Equal(t, "planned", result["local"])
				assert.Equal(t, "feature", currentMutationBranch(t, checkout))
			} else {
				assert.Equal(t, "failed", result["local"])
			}
			assert.Contains(t, stderr.String(), "branch_mutation_failed")
			if operation == "create" {
				assertBranchMissing(t, checkout, "feature")
			} else {
				assertBranchExists(t, checkout, "feature")
				assertBranchExists(t, remote, "feature")
			}
		})
	}
}
