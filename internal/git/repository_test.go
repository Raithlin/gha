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
