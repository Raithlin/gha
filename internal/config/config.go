// Package config loads application configuration at startup.
package config

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Config contains the configuration required by the application.
type Config struct {
	GitHubToken string
	Repository  string
}

// Load reads supported configuration sources, preferring an explicit token
// before falling back to the GitHub CLI's authenticated token.
func Load() Config {
	return load(os.Getenv, githubCLIToken)
}

func load(getenv func(string) string, cliToken func() (string, error)) Config {
	token := strings.TrimSpace(getenv("GHA_GITHUB_TOKEN"))
	if token == "" {
		if candidate, err := cliToken(); err == nil {
			token = strings.TrimSpace(candidate)
		}
	}
	return Config{
		GitHubToken: token,
		Repository:  getenv("GHA_REPOSITORY"),
	}
}

func githubCLIToken() (string, error) {
	return githubCLITokenWith(exec.LookPath, func(path string, args ...string) ([]byte, error) {
		return exec.Command(path, args...).Output()
	})
}

func githubCLITokenWith(lookPath func(string) (string, error), run func(string, ...string) ([]byte, error)) (string, error) {
	path, err := lookPath("gh")
	if err != nil {
		return "", fmt.Errorf("find GitHub CLI: %w", err)
	}
	output, err := run(path, "auth", "token")
	if err != nil {
		return "", fmt.Errorf("read GitHub CLI token: %w", err)
	}
	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", fmt.Errorf("GitHub CLI returned an empty token")
	}
	return token, nil
}
