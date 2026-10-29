package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raithlin/gha/internal/branch"
	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/github"
	"github.com/raithlin/gha/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestMergedPRCleanupCommandAcrossMergeStrategies(t *testing.T) {
	for _, strategy := range []string{"merge", "squash", "rebase"} {
		t.Run(strategy, func(t *testing.T) {
			root := commandWorktreeRepository(t)
			runCommandGit(t, root, "remote", "add", "origin", "https://github.com/acme/project.git")
			runCommandGit(t, root, "checkout", "-b", "topic")
			require.NoError(t, os.WriteFile(filepath.Join(root, "feature.txt"), []byte("landed work\n"), 0600))
			runCommandGit(t, root, "add", "feature.txt")
			runCommandGit(t, root, "commit", "-m", "feature")
			sha := strings.TrimSpace(runCommandGit(t, root, "rev-parse", "HEAD"))
			runCommandGit(t, root, "checkout", "main")
			runCommandGit(t, root, "commit", "--allow-empty", "-m", "base advances")
			switch strategy {
			case "merge":
				runCommandGit(t, root, "merge", "--no-ff", "topic", "-m", "merge PR")
			case "squash":
				runCommandGit(t, root, "merge", "--squash", "topic")
				runCommandGit(t, root, "commit", "-m", "squash PR")
			case "rebase":
				runCommandGit(t, root, "cherry-pick", sha)
			}
			before := runCommandGit(t, root, "show-ref")
			repo := &model.Repository{Name: "project", Owner: model.User{Login: "acme"}, DefaultBranch: "main"}
			pr := &model.PullRequest{Number: 7, State: "closed", MergedAt: "2026-01-01T12:00:00Z", Head: model.BranchRef{Ref: "topic", SHA: sha, Repo: repo}, Base: model.BranchRef{Ref: "main", Repo: repo}}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodGet, r.Method)
				var value any
				switch r.URL.Path {
				case "/repos/acme/project":
					value = repo
				case "/repos/acme/project/pulls":
					require.Equal(t, "2", r.URL.Query().Get("per_page"))
					require.Equal(t, "all", r.URL.Query().Get("state"))
					value = []*model.PullRequest{}
					if r.URL.Query().Get("head") == "acme:topic" {
						value = []*model.PullRequest{pr}
					}
				case "/repos/acme/project/pulls/7":
					value = pr
				default:
					t.Errorf("unexpected request %s", r.URL)
					http.Error(w, "unexpected", 500)
					return
				}
				require.NoError(t, json.NewEncoder(w).Encode(value))
			}))
			defer server.Close()
			baseURL, err := url.Parse(server.URL + "/")
			require.NoError(t, err)
			provider := &github.GitHubClient{HTTPClient: server.Client(), BaseURL: baseURL}
			for _, format := range []string{"json", "text"} {
				cmd := newBranchesCmd(branch.NewService(git.NewBranchLister(root), provider))
				var out, diagnostics bytes.Buffer
				cmd.SetOut(&out)
				cmd.SetErr(&diagnostics)
				cmd.SetArgs([]string{"cleanup", "--path", root, "--base", "main", "--merged-prs", "--format", format})
				require.NoError(t, cmd.Execute())
				require.Empty(t, diagnostics.String())
				if format == "json" {
					var result model.BranchCleanup
					require.NoError(t, json.Unmarshal(out.Bytes(), &result))
					require.Len(t, result.Candidates, 1)
					require.Equal(t, "topic", result.Candidates[0].Name)
					require.Equal(t, "fresh", result.Candidates[0].MergeEvidence.State)
				} else {
					require.Contains(t, out.String(), "merged pr head matches")
					require.Contains(t, out.String(), "#7")
					require.Contains(t, out.String(), "fresh")
				}
			}
			require.Equal(t, before, runCommandGit(t, root, "show-ref"))
		})
	}
}

func TestMergedPRCleanupWithoutOriginReportsUnavailable(t *testing.T) {
	root := commandWorktreeRepository(t)
	runCommandGit(t, root, "branch", "topic")
	cmd := newBranchesCleanupCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--path", root, "--base", "main", "--merged-prs", "--format", "json"})
	require.NoError(t, cmd.Execute())
	var result model.BranchCleanup
	require.NoError(t, json.Unmarshal(out.Bytes(), &result))
	require.Empty(t, result.Candidates)
	require.Equal(t, "provider_unavailable", result.Excluded[1].Reason)
	require.Equal(t, "unavailable", result.Excluded[1].MergeEvidence.State)
}
