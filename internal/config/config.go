// Package config loads application configuration at startup.
package config

import "os"

// Config contains the configuration required by the application.
type Config struct {
	GitHubToken string
	Repository  string
}

// Load reads the currently supported configuration sources.
func Load() Config {
	return Config{
		GitHubToken: os.Getenv("GHA_GITHUB_TOKEN"),
		Repository:  os.Getenv("GHA_REPOSITORY"),
	}
}
