package git

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raithlin/gha/pkg/model"
)

func TestParseRepository(t *testing.T) {
	target, err := ParseRepository("Raithlin/gha")
	require.NoError(t, err)
	assert.Equal(t, model.RepositoryRef{Owner: "Raithlin", Name: "gha"}, target)

	_, err = ParseRepository("Raithlin/gha/extra")
	assert.Error(t, err)
}

func TestRepositoryResolverUsesExplicitPathBeforeConfiguredDefault(t *testing.T) {
	checkout := t.TempDir()
	require.NoError(t, exec.Command("git", "init", "--quiet", checkout).Run())
	require.NoError(t, exec.Command("git", "-C", checkout, "remote", "add", "origin", "git@github.com:path-owner/path-repo.git").Run())

	resolver := NewRepositoryResolver("configured-owner/configured-repo")
	target, err := resolver.ResolveAtPath(context.Background(), "", checkout)

	require.NoError(t, err)
	assert.Equal(t, model.RepositoryRef{Owner: "path-owner", Name: "path-repo"}, target)
}

func TestRepositoryResolverPrefersExplicitRepositoryOverPath(t *testing.T) {
	resolver := NewRepositoryResolver("")
	target, err := resolver.ResolveAtPath(context.Background(), "flag-owner/flag-repo", "/does/not/exist")

	require.NoError(t, err)
	assert.Equal(t, model.RepositoryRef{Owner: "flag-owner", Name: "flag-repo"}, target)
}

func TestParseRepositoryRemote(t *testing.T) {
	for _, remote := range []string{
		"git@github.com:Raithlin/gha.git",
		"https://github.com/Raithlin/gha.git",
		"ssh://git@github.com/Raithlin/gha.git",
	} {
		target, err := ParseRepositoryRemote(remote)
		require.NoError(t, err, remote)
		assert.Equal(t, model.RepositoryRef{Owner: "Raithlin", Name: "gha"}, target)
	}
}

func TestResolveOriginWriteAtPathRejectsMismatchedSelections(t *testing.T) {
	checkout := t.TempDir()
	require.NoError(t, exec.Command("git", "init", "--quiet", checkout).Run())
	require.NoError(t, exec.Command("git", "-C", checkout, "remote", "add", "origin", "git@github.com:Acme/project.git").Run())

	for _, test := range []struct {
		name       string
		configured string
		override   string
		wantError  bool
	}{
		{"origin selected", "", "", false},
		{"matching configured", "acme/PROJECT", "", false},
		{"matching override", "other/repo", "acme/project", false},
		{"mismatched override", "", "other/repo", true},
		{"mismatched configured", "other/repo", "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			selected, err := NewRepositoryResolver(test.configured).ResolveOriginWriteAtPath(context.Background(), test.override, checkout)
			if test.wantError {
				assert.ErrorContains(t, err, "does not match origin")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, model.RepositoryRef{Owner: "Acme", Name: "project"}, selected)
		})
	}
}

func TestValidateOriginWriteAtPathKeepsProviderNeutralWritesAndChecksSelections(t *testing.T) {
	checkout := t.TempDir()
	require.NoError(t, exec.Command("git", "init", "--quiet", checkout).Run())
	require.NoError(t, exec.Command("git", "-C", checkout, "remote", "add", "origin", "/tmp/local-origin.git").Run())
	ctx := context.Background()
	require.NoError(t, NewRepositoryResolver("").ValidateOriginWriteAtPath(ctx, "", checkout))

	require.NoError(t, exec.Command("git", "-C", checkout, "remote", "set-url", "origin", "https://github.com/acme/project.git").Run())
	require.NoError(t, NewRepositoryResolver("acme/project").ValidateOriginWriteAtPath(ctx, "", checkout))
	assert.ErrorContains(t, NewRepositoryResolver("other/repo").ValidateOriginWriteAtPath(ctx, "", checkout), "does not match origin")
	assert.ErrorContains(t, NewRepositoryResolver("").ValidateOriginWriteAtPath(ctx, "other/repo", checkout), "does not match origin")
}

func TestResolveOriginWriteAtPathRejectsMissingOriginAndInvalidSelection(t *testing.T) {
	checkout := t.TempDir()
	require.NoError(t, exec.Command("git", "init", "--quiet", checkout).Run())
	resolver := NewRepositoryResolver("acme/project")
	_, err := resolver.ResolveOriginWriteAtPath(context.Background(), "", checkout)
	assert.ErrorContains(t, err, "resolve origin write target")
	require.NoError(t, exec.Command("git", "-C", checkout, "remote", "add", "origin", "https://github.com/acme/project.git").Run())
	_, err = resolver.ResolveOriginWriteAtPath(context.Background(), "invalid", checkout)
	assert.ErrorContains(t, err, "invalid repository")
}

func TestResolveOriginWriteAtPathUsesConfiguredPushURL(t *testing.T) {
	checkout := t.TempDir()
	require.NoError(t, exec.Command("git", "init", "--quiet", checkout).Run())
	require.NoError(t, exec.Command("git", "-C", checkout, "remote", "add", "origin", "https://github.com/acme/read.git").Run())
	require.NoError(t, exec.Command("git", "-C", checkout, "config", "remote.origin.pushurl", "git@github.com:acme/write.git").Run())
	ctx := context.Background()
	selected, err := NewRepositoryResolver("acme/write").ResolveOriginWriteAtPath(ctx, "", checkout)
	require.NoError(t, err)
	assert.Equal(t, model.RepositoryRef{Owner: "acme", Name: "write"}, selected)
	_, err = NewRepositoryResolver("acme/read").ResolveOriginWriteAtPath(ctx, "", checkout)
	assert.ErrorContains(t, err, "does not match origin")

	require.NoError(t, exec.Command("git", "-C", checkout, "config", "--add", "remote.origin.pushurl", "git@github.com:acme/other.git").Run())
	_, err = NewRepositoryResolver("acme/write").ResolveOriginWriteAtPath(ctx, "", checkout)
	assert.ErrorContains(t, err, "2 push URLs")
}
