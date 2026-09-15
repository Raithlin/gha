package git

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryResolverPrecedenceAndPathSelection(t *testing.T) {
	workdir := t.TempDir()
	runGit(t, workdir, "init", "-b", "main")
	runGit(t, workdir, "remote", "add", "origin", "git@github.com:acme/from-path.git")
	resolver := NewRepositoryResolver("acme/from-config")

	repository, err := resolver.ResolveAtPath(context.Background(), "acme/explicit", workdir)
	require.NoError(t, err)
	assert.Equal(t, "acme/explicit", repository.String())
	repository, err = resolver.ResolveAtPath(context.Background(), "", workdir)
	require.NoError(t, err)
	assert.Equal(t, "acme/from-path", repository.String())
	repository, err = resolver.Resolve(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, "acme/from-config", repository.String())
}

func TestRepositoryParsingAcceptsGitHubTransportsAndRejectsInvalidValues(t *testing.T) {
	for _, remote := range []string{"git@github.com:acme/project.git", "ssh://git@github.com/acme/project.git", "https://github.com/acme/project.git", "http://github.com/acme/project"} {
		repository, err := ParseRepositoryRemote(remote)
		require.NoError(t, err, remote)
		assert.Equal(t, "acme/project", repository.String())
	}
	for _, value := range []string{"", "owner", "owner/", "/repo", "owner/repo/extra"} {
		_, err := ParseRepository(value)
		assert.ErrorContains(t, err, "invalid repository")
	}
	_, err := ParseRepositoryRemote("https://gitlab.com/acme/project.git")
	assert.ErrorContains(t, err, "not a GitHub repository")
}

func TestRepositoryResolverReportsMissingOrigin(t *testing.T) {
	workdir := t.TempDir()
	runGit(t, workdir, "init", "-b", "main")
	_, err := NewRepositoryResolver("").ResolveAtPath(context.Background(), "", workdir)
	assert.ErrorContains(t, err, "could not determine the repository from --path")
}
