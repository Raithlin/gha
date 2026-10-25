package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/review"
	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type lifecycleProvider struct {
	prProvider
	pr        *model.PullRequest
	getErr    error
	reviewErr error
	calls     int
	onGet     func(int)
}

func (p *lifecycleProvider) GetPullRequest(context.Context, string, string, int) (*model.PullRequest, error) {
	p.calls++
	if p.onGet != nil {
		p.onGet(p.calls)
	}
	if p.pr == nil {
		return nil, p.getErr
	}
	value := *p.pr
	return &value, p.getErr
}
func (p *lifecycleProvider) ListReviews(context.Context, string, string, int) ([]*model.Review, error) {
	return nil, p.reviewErr
}

func lifecycleCommandFixture(t *testing.T) (string, *lifecycleProvider) {
	t.Helper()
	root := commandWorktreeRepository(t)
	runCommandGit(t, root, "remote", "add", "origin", "https://github.com/acme/project.git")
	sha := strings.TrimSpace(runCommandGit(t, root, "rev-parse", "HEAD"))
	return root, &lifecycleProvider{pr: &model.PullRequest{Number: 42, State: "open", Head: model.BranchRef{SHA: sha, Ref: "feature", Label: "acme:feature", Repo: &model.Repository{Name: "project", Owner: model.User{Login: "acme"}}}}}
}

func TestPRCheckoutCommandContract(t *testing.T) {
	root, provider := lifecycleCommandFixture(t)
	path := filepath.Join(t.TempDir(), "work")
	cmd := newPRCmd(review.NewService(provider), git.NewRepositoryResolver("wrong/ambient"))
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"checkout", "42", "--worktree", path, "--path", root, "--dry-run", "--format", "json"})
	require.NoError(t, cmd.Execute())
	var result model.PRWorktree
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "acme/project", result.Repository.String())
	assert.Equal(t, "create", result.Decision)
	assert.Equal(t, "planned", result.State)
	assert.Equal(t, "available", result.ReviewState.State)
	require.NotNil(t, result.Review)
	assert.Empty(t, stderr.String())
	assert.NoDirExists(t, path)
}

func TestPRCheckoutResumesAndRetiresCreatedPRBranch(t *testing.T) {
	root, provider := lifecycleCommandFixture(t)
	path := filepath.Join(t.TempDir(), "new-work")
	// New development uses existing worktree add and PR creation rather than checkout.
	add := newWorktreeAddCmd()
	add.SetOut(&bytes.Buffer{})
	add.SetArgs([]string{path, "--branch", "feature", "--new-branch", "--path", root})
	require.NoError(t, add.Execute())
	createProvider := &prProvider{repository: &model.Repository{DefaultBranch: "main", Permissions: &model.RepositoryPermissions{Push: true}}, comparison: &model.BranchComparison{State: "ahead", AheadBy: 1}}
	create := newPRCmd(review.NewService(createProvider), git.NewRepositoryResolver(""))
	create.SetOut(&bytes.Buffer{})
	create.SetArgs([]string{"create", "--path", path, "--title", "New work"})
	require.NoError(t, create.Execute())
	require.Len(t, createProvider.created, 1)
	assert.Equal(t, "feature", createProvider.created[0].Head)
	for _, op := range []string{"checkout", "retire"} {
		if op == "retire" {
			provider.pr.State = "closed"
		}
		cmd := newPRCmd(review.NewService(provider), git.NewRepositoryResolver(""))
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{op, "42", "--worktree", path, "--path", root, "--format", "json"})
		require.NoError(t, cmd.Execute())
		var result model.PRWorktree
		require.NoError(t, json.Unmarshal(out.Bytes(), &result))
		if op == "checkout" {
			assert.Equal(t, "resumed", result.State)
		} else {
			assert.Equal(t, "completed", result.State)
			assert.NoDirExists(t, path)
		}
	}
	assert.Equal(t, "main", strings.TrimSpace(runCommandGit(t, root, "branch", "--show-current")))
}

func TestPRLifecycleCommandErrors(t *testing.T) {
	for _, scenario := range []string{"missing path", "invalid number", "repo mismatch", "provider error", "nil PR", "head race", "invalid format", "arguments", "nil service", "nil resolver", "confirm rejected"} {
		t.Run(scenario, func(t *testing.T) {
			root, p := lifecycleCommandFixture(t)
			path := filepath.Join(t.TempDir(), "work")
			args := []string{"checkout", "42", "--worktree", path, "--path", root, "--dry-run", "--format", "json"}
			service := review.NewService(p)
			resolver := git.NewRepositoryResolver("")
			switch scenario {
			case "missing path":
				args = []string{"checkout", "42", "--path", root, "--dry-run", "--format", "json"}
			case "invalid number":
				args[1] = "nope"
			case "repo mismatch":
				args = append(args, "--repo", "other/project")
			case "provider error":
				p.getErr = errors.New("offline")
			case "nil PR":
				p.pr = nil
			case "head race":
				args = []string{"checkout", "42", "--worktree", path, "--path", root, "--format", "json"}
				p.onGet = func(n int) {
					if n == 2 {
						p.pr.State = "closed"
					}
				}
			case "invalid format":
				args[len(args)-1] = "xml"
			case "arguments":
				args = append(args, "extra")
			case "nil service":
				service = nil
			case "nil resolver":
				resolver = nil
			case "confirm rejected":
				args = append(args, "--confirm")
			}
			cmd := newPRCmd(service, resolver)
			var out, diagnostics bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&diagnostics)
			cmd.SetArgs(args)
			require.Error(t, cmd.Execute())
			assert.Empty(t, out.String())
			assert.NoDirExists(t, path)
		})
	}
}

func TestPRLifecycleUnavailableReviewStillReturnsLocalPlan(t *testing.T) {
	root, p := lifecycleCommandFixture(t)
	p.reviewErr = errors.New("reviews offline")
	cmd := newPRCmd(review.NewService(p), git.NewRepositoryResolver(""))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"checkout", "42", "--worktree", filepath.Join(t.TempDir(), "work"), "--path", root, "--dry-run", "--format", "json"})
	require.NoError(t, cmd.Execute())
	var result model.PRWorktree
	require.NoError(t, json.Unmarshal(out.Bytes(), &result))
	assert.Equal(t, "unavailable", result.ReviewState.State)
	assert.Contains(t, result.ReviewState.Message, "offline")
}

func TestPRWorktreeCapabilitiesAndHelp(t *testing.T) {
	capabilities := ghaCapabilities()
	for _, name := range []string{"pr checkout <number>", "pr retire <number>"} {
		found := false
		for _, entry := range capabilities.Commands {
			if entry.Command == name {
				found = true
				assert.False(t, entry.ReadOnly)
				assert.Equal(t, model.PRWorktreeSchemaVersion, entry.SchemaVersion)
				assert.Contains(t, entry.Formats, "json")
			}
		}
		assert.True(t, found, "missing capability %s", name)
	}
	for _, op := range []string{"checkout", "retire"} {
		cmd := newPRCmd(nil, nil)
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{op, "--help"})
		require.NoError(t, cmd.Execute())
		assert.Contains(t, out.String(), "--worktree")
		assert.Contains(t, out.String(), "--dry-run")
		assert.NotContains(t, out.String(), "--confirm")
	}
}

func TestPRCheckoutRevalidatesFetchOrigin(t *testing.T) {
	root, p := lifecycleCommandFixture(t)
	p.onGet = func(n int) {
		if n == 2 {
			runCommandGit(t, root, "remote", "set-url", "origin", "https://github.com/other/project.git")
		}
	}
	cmd := newPRCmd(review.NewService(p), git.NewRepositoryResolver(""))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"checkout", "42", "--worktree", filepath.Join(t.TempDir(), "work"), "--path", root, "--format", "json"})
	require.ErrorContains(t, cmd.Execute(), "does not match")
	assert.Empty(t, out.String())
}

func lifecycleFetchFixture(t *testing.T, root string, fail bool) {
	t.Helper()
	gitExecutable, err := exec.LookPath("git")
	require.NoError(t, err)
	origin := filepath.Join(t.TempDir(), "origin.git")
	runCommandGit(t, root, "init", "--bare", origin)
	runCommandGit(t, root, "push", origin, "HEAD:refs/pull/42/head")
	directory := t.TempDir()
	script := `#!/bin/sh
case "$*" in
 *fetch\ *)
  if [ "$GHA_TEST_FETCH_FAIL" = yes ]; then echo fetch-offline >&2; exit 2; fi
  exec "$GHA_TEST_REAL_GIT" -c "url.$GHA_TEST_ORIGIN.insteadOf=https://github.com/acme/project.git" "$@";;
esac
exec "$GHA_TEST_REAL_GIT" "$@"
`
	require.NoError(t, os.WriteFile(filepath.Join(directory, "git"), []byte(script), 0700))
	t.Setenv("GHA_TEST_REAL_GIT", gitExecutable)
	t.Setenv("GHA_TEST_ORIGIN", origin)
	mode := "no"
	if fail {
		mode = "yes"
	}
	t.Setenv("GHA_TEST_FETCH_FAIL", mode)
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestPRCheckoutExecutionAndFailureStreams(t *testing.T) {
	for _, format := range []string{"json", "yaml", "text"} {
		for _, fail := range []bool{false, true} {
			t.Run(format+strconv.FormatBool(fail), func(t *testing.T) {
				root, p := lifecycleCommandFixture(t)
				lifecycleFetchFixture(t, root, fail)
				path := filepath.Join(t.TempDir(), "PR work")
				cmd := newPRCmd(review.NewService(p), git.NewRepositoryResolver(""))
				var out, diagnostics bytes.Buffer
				cmd.SetOut(&out)
				cmd.SetErr(&diagnostics)
				cmd.SetArgs([]string{"checkout", "42", "--worktree", path, "--path", root, "--format", format})
				err := cmd.Execute()
				if fail {
					require.ErrorContains(t, err, "fetch-offline")
					assert.NoDirExists(t, path)
					if format != "text" {
						assert.Contains(t, diagnostics.String(), "pr_worktree_failed")
					}
				} else {
					require.NoError(t, err)
					assert.DirExists(t, path)
				}
				assert.NotEmpty(t, out.String())
				if format == "json" {
					var result model.PRWorktree
					require.NoError(t, json.Unmarshal(out.Bytes(), &result))
					if fail {
						assert.Equal(t, "failed", result.Fetch)
					} else {
						assert.Equal(t, "completed", result.State)
					}
				}
			})
		}
	}
}

func TestPRCheckoutOutputFailurePreservesExecutionError(t *testing.T) {
	root, p := lifecycleCommandFixture(t)
	lifecycleFetchFixture(t, root, true)
	cmd := newPRCmd(review.NewService(p), git.NewRepositoryResolver(""))
	cmd.SetOut(commandFailingWriter{})
	cmd.SetArgs([]string{"checkout", "42", "--worktree", filepath.Join(t.TempDir(), "work"), "--path", root, "--format", "json"})
	err := cmd.Execute()
	require.ErrorContains(t, err, "fetch-offline")
	require.ErrorContains(t, err, "writer failed")
}

func TestPRCheckoutUnknownPushTargetAndFailedRecheck(t *testing.T) {
	for _, mode := range []string{"missing repository", "provider recheck failed", "local preflight failed", "wrong PR identity"} {
		t.Run(mode, func(t *testing.T) {
			root, p := lifecycleCommandFixture(t)
			path := filepath.Join(t.TempDir(), "work")
			dryRun := true
			switch mode {
			case "missing repository":
				p.pr.Head.Repo = nil
			case "provider recheck failed":
				dryRun = false
				p.onGet = func(n int) {
					if n == 2 {
						p.getErr = errors.New("recheck offline")
					}
				}
			case "local preflight failed":
				require.NoError(t, os.Mkdir(path, 0700))
			case "wrong PR identity":
				p.pr.Number = 99
			}
			cmd := newPRCmd(review.NewService(p), git.NewRepositoryResolver(""))
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&bytes.Buffer{})
			args := []string{"checkout", "42", "--worktree", path, "--path", root, "--format", "json"}
			if dryRun {
				args = append(args, "--dry-run")
			}
			cmd.SetArgs(args)
			err := cmd.Execute()
			if mode == "missing repository" {
				require.NoError(t, err)
				assert.Contains(t, out.String(), "inspect_push_target")
			} else {
				require.Error(t, err)
				assert.Empty(t, out.String())
			}
		})
	}
}
