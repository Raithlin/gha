// Package git provides access to local Git repository metadata.
package git

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/raithlin/gha/pkg/model"
)

// RepositoryResolver resolves a GitHub repository from an explicit value,
// local checkout, configured default, or the current directory's origin remote.
type RepositoryResolver struct {
	defaultRepository string
}

// DraftPullRequest summarizes committed base-to-head changes without changing
// repository state. Results are bounded to keep the proposal reviewable.
func DraftPullRequest(ctx context.Context, workdir, base, head string, limit int) ([]string, []string, bool, bool, error) {
	if limit < 1 {
		limit = 50
	}
	commits, err := gitOutput(ctx, workdir, "log", "--format=%s", "--max-count="+strconv.Itoa(limit+1), base+".."+head)
	if err != nil {
		return nil, nil, false, false, fmt.Errorf("summarize commits between %s and %s: %w", base, head, err)
	}
	files, err := gitOutput(ctx, workdir, "diff", "--name-only", base+"..."+head)
	if err != nil {
		return nil, nil, false, false, fmt.Errorf("list changed files between %s and %s: %w", base, head, err)
	}
	commitList := nonemptyLines(commits)
	fileList := nonemptyLines(files)
	commitsTruncated, filesTruncated := len(commitList) > limit, len(fileList) > limit
	if commitsTruncated {
		commitList = commitList[:limit]
	}
	if filesTruncated {
		fileList = fileList[:limit]
	}
	return commitList, fileList, commitsTruncated, filesTruncated, nil
}

func gitOutput(ctx context.Context, workdir string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = workdir
	output, err := command.Output()
	return string(output), err
}

func nonemptyLines(value string) []string {
	lines := strings.Split(strings.TrimSpace(value), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}
	}
	return lines
}

// NewRepositoryResolver creates a resolver with an optional configured default.
func NewRepositoryResolver(defaultRepository string) *RepositoryResolver {
	return &RepositoryResolver{defaultRepository: defaultRepository}
}

// Resolve returns the repository selected by the command-line override, then
// the configured default, then the current directory's origin remote.
func (r *RepositoryResolver) Resolve(ctx context.Context, override string) (model.RepositoryRef, error) {
	return r.ResolveAtPath(ctx, override, "")
}

// ResolveAtPath returns the repository selected by --repo, then the origin of
// an explicit local checkout, then the configured default, then the current
// directory's origin remote. An explicit --path takes precedence over ambient
// configuration so commands can inspect another checkout predictably.
func (r *RepositoryResolver) ResolveAtPath(ctx context.Context, override, path string) (model.RepositoryRef, error) {
	if override != "" {
		return ParseRepository(override)
	}
	if path != "" {
		return repositoryFromOrigin(ctx, path, "could not determine the repository from --path; pass --repo owner/repo or use a Git checkout with an origin remote")
	}
	if r.defaultRepository != "" {
		return ParseRepository(r.defaultRepository)
	}

	return repositoryFromOrigin(ctx, "", "could not determine the repository; pass --repo owner/repo, --path /path/to/checkout, or set GHA_REPOSITORY")
}

// ResolveOriginWriteAtPath binds provider checks to the checkout's origin.
// An explicit or configured repository may select that same repository, but
// cannot redirect safety checks away from the Git remote being changed.
func (r *RepositoryResolver) ResolveOriginWriteAtPath(ctx context.Context, override, path string) (model.RepositoryRef, error) {
	origin, err := repositoryFromOriginWriteTarget(ctx, path)
	if err != nil {
		return model.RepositoryRef{}, fmt.Errorf("resolve origin write target: %w", err)
	}
	selection := override
	if selection == "" {
		selection = r.defaultRepository
	}
	if selection != "" {
		selected, err := ParseRepository(selection)
		if err != nil {
			return model.RepositoryRef{}, err
		}
		if !strings.EqualFold(selected.Owner, origin.Owner) || !strings.EqualFold(selected.Name, origin.Name) {
			return model.RepositoryRef{}, fmt.Errorf("selected repository %s/%s does not match origin %s/%s", selected.Owner, selected.Name, origin.Owner, origin.Name)
		}
	}
	return origin, nil
}

func repositoryFromOriginWriteTarget(ctx context.Context, path string) (model.RepositoryRef, error) {
	command := exec.CommandContext(ctx, "git", "config", "--get-all", "remote.origin.pushurl")
	command.Dir = path
	output, err := command.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return model.RepositoryRef{}, fmt.Errorf("read origin push URL: %w", err)
		}
		return repositoryFromOrigin(ctx, path, "cannot determine the checkout's origin repository for a branch write")
	}
	pushURLs := nonemptyLines(string(output))
	if len(pushURLs) != 1 {
		return model.RepositoryRef{}, fmt.Errorf("origin has %d push URLs; select one write target before using provider safety checks", len(pushURLs))
	}
	return ParseRepositoryRemote(pushURLs[0])
}

// ValidateOriginWriteAtPath checks an optional provider selection before a Git
// write. Provider-neutral writes remain usable when no provider is selected.
func (r *RepositoryResolver) ValidateOriginWriteAtPath(ctx context.Context, override, path string) error {
	if override == "" && r.defaultRepository == "" {
		return nil
	}
	_, err := r.ResolveOriginWriteAtPath(ctx, override, path)
	return err
}

func repositoryFromOrigin(ctx context.Context, path, message string) (model.RepositoryRef, error) {
	command := exec.CommandContext(ctx, "git", "config", "--get", "remote.origin.url")
	command.Dir = path
	output, err := command.Output()
	if err != nil {
		return model.RepositoryRef{}, fmt.Errorf("%s", message)
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
