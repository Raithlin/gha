package config

import (
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

	assert.Equal(t, Config{}, Load())
}
