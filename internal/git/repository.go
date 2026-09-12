// Package git provides access to local Git repository metadata.
package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/raithlin/gha/pkg/model"
)

// RepositoryResolver resolves a GitHub repository from an explicit value,
// configured default, or the current directory's origin remote.
type RepositoryResolver struct {
	defaultRepository string
}

// NewRepositoryResolver creates a resolver with an optional configured default.
func NewRepositoryResolver(defaultRepository string) *RepositoryResolver {
	return &RepositoryResolver{defaultRepository: defaultRepository}
}

// Resolve returns the repository selected by the command-line override, then
// the configured default, then the local origin remote.
func (r *RepositoryResolver) Resolve(ctx context.Context, override string) (model.RepositoryRef, error) {
	if override != "" {
		return ParseRepository(override)
	}
	if r.defaultRepository != "" {
		return ParseRepository(r.defaultRepository)
	}

	output, err := exec.CommandContext(ctx, "git", "config", "--get", "remote.origin.url").Output()
	if err != nil {
		return model.RepositoryRef{}, fmt.Errorf("could not determine the repository; pass --repo owner/repo or set GHA_REPOSITORY")
	}
	return ParseRepositoryRemote(strings.TrimSpace(string(output)))
}

// ParseRepository parses an owner/repository value.
func ParseRepository(value string) (model.RepositoryRef, error) {
	parts := strings.Split(strings.TrimSpace(value), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return model.RepositoryRef{}, fmt.Errorf("invalid repository %q; use owner/repo", value)
	}
	return model.RepositoryRef{Owner: parts[0], Name: parts[1]}, nil
}

// ParseRepositoryRemote extracts an owner and name from a GitHub origin URL.
func ParseRepositoryRemote(remote string) (model.RepositoryRef, error) {
	remote = strings.TrimSuffix(strings.TrimSpace(remote), ".git")
	remote = strings.TrimPrefix(remote, "git@github.com:")
	remote = strings.TrimPrefix(remote, "ssh://git@github.com/")
	remote = strings.TrimPrefix(remote, "https://github.com/")
	remote = strings.TrimPrefix(remote, "http://github.com/")
	if strings.Contains(remote, "://") || strings.Contains(remote, "@") {
		return model.RepositoryRef{}, fmt.Errorf("origin remote %q is not a GitHub repository; pass --repo owner/repo", remote)
	}
	return ParseRepository(remote)
}
