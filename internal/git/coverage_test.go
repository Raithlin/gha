package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchListerParserFailuresRemainActionable(t *testing.T) {
	for _, test := range []struct {
		name string
		mode string
		call func(*BranchLister) error
		want string
	}{
		{"local list", "local-malformed", func(l *BranchLister) error { _, err := l.listLocal(context.Background(), "", 1); return err }, "parse local branch"},
		{"origin list", "origin-malformed", func(l *BranchLister) error { _, _, err := l.listOrigin(context.Background(), 1); return err }, "parse origin branch"},
		{"local inspection", "inspect-local-malformed", func(l *BranchLister) error { _, err := l.inspectLocal(context.Background(), "feature", ""); return err }, "parse local branch"},
		{"origin inspection", "inspect-origin-malformed", func(l *BranchLister) error { _, err := l.inspectOrigin(context.Background(), "feature"); return err }, "parse origin branch"},
		{"divergence shape", "divergence-malformed", func(l *BranchLister) error {
			_, _, err := l.divergence(context.Background(), "feature", "origin/feature")
			return err
		}, "parse divergence"},
		{"divergence ahead", "divergence-ahead", func(l *BranchLister) error {
			_, _, err := l.divergence(context.Background(), "feature", "origin/feature")
			return err
		}, "parse ahead"},
		{"divergence behind", "divergence-behind", func(l *BranchLister) error {
			_, _, err := l.divergence(context.Background(), "feature", "origin/feature")
			return err
		}, "parse behind"},
	} {
		t.Run(test.name, func(t *testing.T) {
			installFakeGit(t, test.mode)
			err := test.call(NewBranchLister(t.TempDir()))
			assert.ErrorContains(t, err, test.want)
		})
	}
}

func TestAnalyzerParserFailuresRemainActionable(t *testing.T) {
	for _, test := range []struct {
		name string
		mode string
		call func(*Analyzer) error
		want string
	}{
		{"head count", "head-count-malformed", func(a *Analyzer) error { _, err := a.head(context.Background()); return err }, "parse commit count"},
		{"worktree entry", "worktree-malformed", func(a *Analyzer) error { _, err := a.worktree(context.Background(), 1); return err }, "parse worktree"},
		{"worktree rename", "worktree-rename-malformed", func(a *Analyzer) error { _, err := a.worktree(context.Background(), 1); return err }, "parse renamed"},
		{"storage entry", "storage-malformed", func(a *Analyzer) error { _, err := a.storage(context.Background()); return err }, "parse object storage entry"},
		{"storage value", "storage-value-malformed", func(a *Analyzer) error { _, err := a.storage(context.Background()); return err }, "parse object storage value"},
		{"recent commits", "recent-malformed", func(a *Analyzer) error { _, _, err := a.recentCommits(context.Background(), 1); return err }, "parse recent commit"},
		{"largest entry", "largest-malformed", func(a *Analyzer) error { _, _, err := a.largestFiles(context.Background(), 1); return err }, "parse tracked file"},
		{"largest size", "largest-size-malformed", func(a *Analyzer) error { _, _, err := a.largestFiles(context.Background(), 1); return err }, "parse tracked file size"},
	} {
		t.Run(test.name, func(t *testing.T) {
			installFakeGit(t, test.mode)
			err := test.call(NewAnalyzer(t.TempDir()))
			assert.ErrorContains(t, err, test.want)
		})
	}
}

func TestGitCommandDiagnosticsAndHelpers(t *testing.T) {
	installFakeGit(t, "diagnostic")
	_, err := NewBranchLister(t.TempDir()).run(context.Background(), "anything")
	assert.ErrorContains(t, err, "fake diagnostic")
	assert.Equal(t, []string(nil), lines(""))
	assert.Equal(t, []string{"one", "two"}, lines("one\ntwo\n"))
	assert.False(t, conflicted("MM"))
	assert.True(t, conflicted("UU"))
}

func TestGitWorkflowsWrapCommandFailuresAtEachDecisionPoint(t *testing.T) {
	for _, test := range []struct {
		name string
		mode string
		call func(*BranchLister) error
		want string
	}{
		{"list current", "list-current-error", func(l *BranchLister) error { _, err := l.List(context.Background(), 1); return err }, "read current branch"},
		{"list local", "list-local-error", func(l *BranchLister) error { _, err := l.List(context.Background(), 1); return err }, "list local branches"},
		{"list origin refs", "list-origin-refs-error", func(l *BranchLister) error { _, err := l.List(context.Background(), 1); return err }, "list origin branches"},
		{"refresh fetch", "refresh-fetch-error", func(l *BranchLister) error { _, err := l.RefreshOrigin(context.Background(), 1, false); return err }, "refresh origin"},
		{"inspect current", "inspect-current-error", func(l *BranchLister) error { _, err := l.Inspect(context.Background(), "feature"); return err }, "read current branch"},
		{"inspect local", "inspect-local-error", func(l *BranchLister) error { _, err := l.Inspect(context.Background(), "feature"); return err }, "inspect local branch"},
		{"inspect origin", "inspect-origin-ref-error", func(l *BranchLister) error { _, err := l.Inspect(context.Background(), "feature"); return err }, "inspect origin branch"},
	} {
		t.Run(test.name, func(t *testing.T) {
			installFakeGit(t, test.mode)
			assert.ErrorContains(t, test.call(NewBranchLister(t.TempDir())), test.want)
		})
	}
}

func TestAnalyzerAndWriterWrapGitFailures(t *testing.T) {
	for _, test := range []struct {
		name string
		mode string
		call func() error
	}{
		{"analyzer root", "analyzer-root-error", func() error { _, err := NewAnalyzer(t.TempDir()).Analyze(context.Background(), 1); return err }},
		{"analyzer worktree", "analyzer-worktree-error", func() error { _, err := NewAnalyzer(t.TempDir()).Analyze(context.Background(), 1); return err }},
		{"analyzer storage", "analyzer-storage-error", func() error { _, err := NewAnalyzer(t.TempDir()).Analyze(context.Background(), 1); return err }},
		{"writer current", "writer-error", func() error { _, err := NewBranchWriter(t.TempDir()).CurrentBranch(context.Background()); return err }},
		{"writer default", "writer-error", func() error { _, err := NewBranchWriter(t.TempDir()).DefaultBranch(context.Background()); return err }},
		{"writer delete", "writer-error", func() error { return NewBranchWriter(t.TempDir()).DeleteLocal(context.Background(), "feature", true) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			installFakeGit(t, test.mode)
			assert.Error(t, test.call())
		})
	}
}

func TestGitHelpersCoverDetachedAndMergeBaseFailureStates(t *testing.T) {
	installFakeGit(t, "empty-success")
	_, err := CurrentBranch(context.Background(), t.TempDir())
	assert.ErrorContains(t, err, "detached")

	installFakeGit(t, "diagnostic")
	_, err = NewBranchLister(t.TempDir()).isAncestor(context.Background(), "feature", "main")
	assert.ErrorContains(t, err, "fake diagnostic")

	installFakeGit(t, "silent-error")
	_, err = NewBranchWriter(t.TempDir()).output(context.Background(), "branch", "feature")
	assert.Error(t, err)
}

func TestAnalyzerAndBranchListerKeepOptionalFailuresExplicit(t *testing.T) {
	for _, test := range []struct {
		name string
		mode string
		call func() error
		want string
	}{
		{"analyzer commits", "analyzer-commits-error", func() error { _, err := NewAnalyzer(t.TempDir()).Analyze(context.Background(), 1); return err }, "read recent commits"},
		{"analyzer files", "analyzer-files-error", func() error { _, err := NewAnalyzer(t.TempDir()).Analyze(context.Background(), 1); return err }, "read tracked files"},
	} {
		t.Run(test.name, func(t *testing.T) {
			installFakeGit(t, test.mode)
			assert.ErrorContains(t, test.call(), test.want)
		})
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewAnalyzer(t.TempDir()).head(cancelled)
	assert.ErrorContains(t, err, "read HEAD")
	_, _, err = NewBranchLister(t.TempDir()).originURL(cancelled)
	assert.ErrorContains(t, err, "read origin URL")

	installFakeGit(t, "inspect-divergence-error")
	inspection, err := NewBranchLister(t.TempDir()).Inspect(context.Background(), "feature")
	require.NoError(t, err)
	require.NotNil(t, inspection.Local)
	assert.Equal(t, "unavailable", inspection.Local.DivergenceState)
	assert.Contains(t, inspection.Local.DivergenceMessage, "compare feature")
}

func TestAnalyzerAndBranchListerPreserveDetailedBoundedStates(t *testing.T) {
	installFakeGit(t, "analyzer-details")
	analyzer := NewAnalyzer(t.TempDir())
	worktree, err := analyzer.worktree(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "dirty", worktree.State)
	assert.Equal(t, 1, worktree.Conflicted)
	assert.True(t, worktree.ChangesTruncated)
	commits, truncated, err := analyzer.recentCommits(context.Background(), 1)
	require.NoError(t, err)
	assert.True(t, truncated)
	require.Len(t, commits, 1)
	files, truncated, err := analyzer.largestFiles(context.Background(), 1)
	require.NoError(t, err)
	assert.True(t, truncated)
	require.Len(t, files, 1)
	assert.Equal(t, "a", files[0].Path)

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = NewBranchLister(t.TempDir()).RefreshOrigin(cancelled, 1, false)
	assert.ErrorContains(t, err, "read origin URL")

	installFakeGit(t, "inspect-cached-origin")
	inspection, err := NewBranchLister(t.TempDir()).Inspect(context.Background(), "feature")
	require.NoError(t, err)
	assert.Equal(t, "unconfigured_cached", inspection.OriginState)
	assert.Nil(t, inspection.Local)
	require.NotNil(t, inspection.OriginBranch)
}

func TestGitAnalysisRetainsEmptyAndIntermediateFailureStates(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewAnalyzer(t.TempDir()).storage(cancelled)
	assert.ErrorContains(t, err, "read object storage")

	installFakeGit(t, "head-count-error")
	_, err = NewAnalyzer(t.TempDir()).head(context.Background())
	assert.ErrorContains(t, err, "count commits")

	installFakeGit(t, "empty-success")
	files, truncated, err := NewAnalyzer(t.TempDir()).largestFiles(context.Background(), 1)
	require.NoError(t, err)
	assert.Empty(t, files)
	assert.False(t, truncated)
}

func TestCleanupReportsEachUnavailableLocalGitFact(t *testing.T) {
	for _, test := range []struct {
		name string
		mode string
		base string
		want string
	}{
		{"empty cached base", "cleanup-empty", "", "cleanup base is empty"},
		{"current branch", "cleanup-current-error", "main", "read current branch"},
		{"local branches", "cleanup-local-error", "main", "list local branches"},
		{"reachability", "cleanup-ancestor-error", "main", "compare branch"},
	} {
		t.Run(test.name, func(t *testing.T) {
			installFakeGit(t, test.mode)
			_, err := NewBranchLister(t.TempDir()).Cleanup(context.Background(), test.base, 10)
			assert.ErrorContains(t, err, test.want)
		})
	}
}

func installFakeGit(t *testing.T, mode string) {
	t.Helper()
	directory := t.TempDir()
	script := `#!/bin/sh
case "$GHA_FAKE_GIT_MODE:$*" in
  local-malformed:*) echo 'broken' ;;
  origin-malformed:*) echo 'broken' ;;
  inspect-local-malformed:*) echo 'broken' ;;
  inspect-origin-malformed:*) echo 'broken' ;;
  divergence-malformed:*) echo 'one' ;;
  divergence-ahead:*) echo 'not-a-number 1' ;;
  divergence-behind:*) echo '1 not-a-number' ;;
  head-count-malformed:*rev-parse*) echo 'abc' ;;
  head-count-malformed:*rev-list*) echo 'many' ;;
  worktree-malformed:*) printf 'bad\000' ;;
  worktree-rename-malformed:*) printf 'R  renamed\000' ;;
  storage-malformed:*) echo 'bad' ;;
  storage-value-malformed:*) echo 'count no' ;;
  recent-malformed:*) printf 'sha\000subject' ;;
  largest-malformed:*) printf 'broken\000' ;;
  largest-size-malformed:*) printf '100644 blob sha nope\tfile\000' ;;
  diagnostic:*) echo 'fake diagnostic' >&2; exit 2 ;;
  list-current-error:*) echo 'fake diagnostic' >&2; exit 2 ;;
  list-local-error:*branch\ --show-current*) echo 'feature' ;;
  list-local-error:*) echo 'fake diagnostic' >&2; exit 2 ;;
  list-origin-error:*branch\ --show-current*) echo 'feature' ;;
  list-origin-error:*for-each-ref*) echo '' ;;
  list-origin-error:*) echo 'fake diagnostic' >&2; exit 2 ;;
  list-origin-refs-error:*branch\ --show-current*) echo 'feature' ;;
  list-origin-refs-error:*for-each-ref*refs/heads*) echo '' ;;
  list-origin-refs-error:*remote\ get-url\ origin*) echo 'https://example.test/repo' ;;
  list-origin-refs-error:*) echo 'fake diagnostic' >&2; exit 2 ;;
  refresh-fetch-error:*remote\ get-url\ origin*) echo 'https://example.test/repo' ;;
  refresh-fetch-error:*) echo 'fake diagnostic' >&2; exit 2 ;;
  inspect-current-error:*) echo 'fake diagnostic' >&2; exit 2 ;;
  inspect-url-error:*branch\ --show-current*) echo 'feature' ;;
  inspect-url-error:*) echo 'fake diagnostic' >&2; exit 2 ;;
  inspect-local-error:*branch\ --show-current*) echo 'feature' ;;
  inspect-local-error:*remote\ get-url\ origin*) echo 'https://example.test/repo' ;;
  inspect-local-error:*) echo 'fake diagnostic' >&2; exit 2 ;;
  inspect-origin-ref-error:*branch\ --show-current*) echo 'feature' ;;
  inspect-origin-ref-error:*remote\ get-url\ origin*) echo 'https://example.test/repo' ;;
  inspect-origin-ref-error:*for-each-ref*refs/heads*) echo '' ;;
  inspect-origin-ref-error:*) echo 'fake diagnostic' >&2; exit 2 ;;
  analyzer-root-error:*) echo 'fake diagnostic' >&2; exit 2 ;;
  analyzer-worktree-error:*rev-parse*) echo '/repo' ;;
  analyzer-worktree-error:*) echo 'fake diagnostic' >&2; exit 2 ;;
  analyzer-storage-error:*rev-parse*) echo '/repo' ;;
  analyzer-storage-error:*status*) echo '' ;;
  analyzer-storage-error:*) echo 'fake diagnostic' >&2; exit 2 ;;
  analyzer-head-error:*rev-parse\ --show-toplevel*) echo '/repo' ;;
  analyzer-head-error:*status*) printf '' ;;
  analyzer-head-error:*count-objects*) echo 'count: 1' ;;
  analyzer-head-error:*) echo 'fake diagnostic' >&2; exit 2 ;;
  analyzer-commits-error:*rev-parse\ --show-toplevel*) echo '/repo' ;;
  analyzer-commits-error:*status*) printf '' ;;
  analyzer-commits-error:*count-objects*) echo 'count: 1' ;;
  analyzer-commits-error:*rev-parse\ --verify\ HEAD*) echo 'abc' ;;
  analyzer-commits-error:*rev-list\ --count*) echo '1' ;;
  analyzer-commits-error:*branch\ --show-current*) echo 'main' ;;
  analyzer-commits-error:*) echo 'fake diagnostic' >&2; exit 2 ;;
  analyzer-files-error:*rev-parse\ --show-toplevel*) echo '/repo' ;;
  analyzer-files-error:*status*) printf '' ;;
  analyzer-files-error:*count-objects*) echo 'count: 1' ;;
  analyzer-files-error:*rev-parse\ --verify\ HEAD*) echo 'abc' ;;
  analyzer-files-error:*rev-list\ --count*) echo '1' ;;
  analyzer-files-error:*branch\ --show-current*) echo 'main' ;;
  analyzer-files-error:*log*) printf 'abc\000subject\0002026-01-01T00:00:00Z\000' ;;
  analyzer-files-error:*) echo 'fake diagnostic' >&2; exit 2 ;;
  inspect-divergence-error:*branch\ --show-current*) echo 'feature' ;;
  inspect-divergence-error:*remote\ get-url\ origin*) echo 'https://example.test/repo' ;;
  inspect-divergence-error:*for-each-ref*refs/heads/feature*) printf 'feature\tabc\torigin/feature\n' ;;
  inspect-divergence-error:*for-each-ref*refs/remotes/origin/feature*) echo '' ;;
  inspect-divergence-error:*) echo 'fake diagnostic' >&2; exit 2 ;;
  analyzer-details:*status*) printf 'UU conflicted\000R  renamed\000original-name\000?? untracked\000' ;;
  analyzer-details:*log*) printf '\000sha-one\000first\0002026-01-01T00:00:00Z\000sha-two\000second\0002026-01-02T00:00:00Z\000' ;;
  analyzer-details:*ls-tree*) printf '100644 tree ignored 5\tignored\000100644 blob sha 5\tb\000100644 blob sha 5\ta\000' ;;
  analyzer-details:*) exit 0 ;;
  inspect-cached-origin:*branch\ --show-current*) printf '' ;;
  inspect-cached-origin:*remote\ get-url\ origin*) exit 2 ;;
  inspect-cached-origin:*for-each-ref*refs/heads/feature*) printf '' ;;
  inspect-cached-origin:*for-each-ref*refs/remotes/origin/feature*) printf 'feature\tabc\n' ;;
  inspect-cached-origin:*) exit 0 ;;
  head-count-error:*rev-parse\ --verify\ HEAD*) echo 'abc' ;;
  head-count-error:*rev-list\ --count*) echo 'fake diagnostic' >&2; exit 2 ;;
  head-count-error:*) exit 0 ;;
  cleanup-empty:*symbolic-ref*) printf '' ;;
  cleanup-empty:*) exit 0 ;;
  cleanup-current-error:*show-ref*) exit 0 ;;
  cleanup-current-error:*branch\ --show-current*) echo 'fake diagnostic' >&2; exit 2 ;;
  cleanup-current-error:*) exit 0 ;;
  cleanup-local-error:*show-ref*) exit 0 ;;
  cleanup-local-error:*branch\ --show-current*) echo 'main' ;;
  cleanup-local-error:*for-each-ref*) echo 'fake diagnostic' >&2; exit 2 ;;
  cleanup-local-error:*) exit 0 ;;
  cleanup-ancestor-error:*show-ref*) exit 0 ;;
  cleanup-ancestor-error:*branch\ --show-current*) echo 'main' ;;
  cleanup-ancestor-error:*for-each-ref*) printf 'main\tabc\t\nfeature\tdef\t\n' ;;
  cleanup-ancestor-error:*merge-base*) echo 'fake diagnostic' >&2; exit 2 ;;
  cleanup-ancestor-error:*) exit 0 ;;
  writer-error:*) echo 'fake diagnostic' >&2; exit 2 ;;
  empty-success:*) exit 0 ;;
  silent-error:*) exit 2 ;;
  *) exit 0 ;;
esac
`
	path := filepath.Join(directory, "git")
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	t.Setenv("GHA_FAKE_GIT_MODE", mode)
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
}
