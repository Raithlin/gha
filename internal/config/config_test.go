package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadReadsSupportedEnvironmentVariables(t *testing.T) {
	t.Setenv("GHA_GITHUB_TOKEN", "token")
	t.Setenv("GHA_REPOSITORY", "acme/project")

	assert.Equal(t, Config{GitHubToken: "token", Repository: "acme/project"}, Load())
}

func TestLoadUsesEmptyValuesWhenConfigurationIsUnset(t *testing.T) {
	t.Setenv("GHA_GITHUB_TOKEN", "")
	t.Setenv("GHA_REPOSITORY", "")

	// Inject an unavailable CLI credential so this test never inspects or
	// formats a developer's actual gh token.
	configuration := load(func(string) string { return "" }, func() (string, error) {
		return "", errors.New("gh is unavailable")
	})
	assert.Equal(t, Config{}, configuration)
}

func TestLoadUsesGitHubCLITokenWhenEnvironmentTokenIsUnset(t *testing.T) {
	called := false
	configuration := load(func(name string) string {
		if name == "GHA_REPOSITORY" {
			return "acme/project"
		}
		return ""
	}, func() (string, error) {
		called = true
		return " gh-token\n", nil
	})

	assert.True(t, called)
	assert.Equal(t, Config{GitHubToken: "gh-token", Repository: "acme/project"}, configuration)
}

func TestLoadPrefersEnvironmentTokenOverGitHubCLI(t *testing.T) {
	called := false
	configuration := load(func(name string) string {
		if name == "GHA_GITHUB_TOKEN" {
			return " env-token\n"
		}
		return ""
	}, func() (string, error) {
		called = true
		return "gh-token", nil
	})

	assert.False(t, called)
	assert.Equal(t, "env-token", configuration.GitHubToken)
}

func TestLoadKeepsUnauthenticatedModeWhenGitHubCLIIsUnavailable(t *testing.T) {
	configuration := load(func(string) string { return "" }, func() (string, error) {
		return "", errors.New("gh is not authenticated")
	})

	assert.Equal(t, Config{}, configuration)
}

func TestGitHubCLITokenUsesGhAuthToken(t *testing.T) {
	var lookedUp, commandPath string
	var commandArgs []string
	token, err := githubCLITokenWith(func(name string) (string, error) {
		lookedUp = name
		return "/usr/bin/gh", nil
	}, func(path string, args ...string) ([]byte, error) {
		commandPath = path
		commandArgs = args
		return []byte(" gh-token\n"), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "gh", lookedUp)
	assert.Equal(t, "/usr/bin/gh", commandPath)
	assert.Equal(t, []string{"auth", "token"}, commandArgs)
	assert.Equal(t, "gh-token", token)
}

func TestGitHubCLITokenReportsMissingCLIAndAuthFailures(t *testing.T) {
	_, err := githubCLITokenWith(func(string) (string, error) {
		return "", errors.New("not found")
	}, func(string, ...string) ([]byte, error) {
		t.Fatal("must not run without gh")
		return nil, nil
	})
	assert.ErrorContains(t, err, "find GitHub CLI")

	_, err = githubCLITokenWith(func(string) (string, error) { return "/usr/bin/gh", nil }, func(string, ...string) ([]byte, error) {
		return nil, errors.New("not logged in")
	})
	assert.ErrorContains(t, err, "read GitHub CLI token")

	_, err = githubCLITokenWith(func(string) (string, error) { return "/usr/bin/gh", nil }, func(string, ...string) ([]byte, error) {
		return []byte(" \n "), nil
	})
	assert.ErrorContains(t, err, "empty token")
}
