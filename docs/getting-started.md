# Getting started

GHA is an agent-first developer tool written in Go. It gives coding agents
bounded, versioned, decision-ready views of GitHub, Git, CI, and local
repositories; developers use the same commands through a clear terminal UI.

GHA does not aim to replace `git` or `gh`. It earns a command when it makes a
workflow easier or safer by combining local and provider signals, normalizing
them into a stable model, or exposing an actionable summary. Raw Git and GitHub
operations remain the right tool when they are the clearer or more efficient
choice.

## Prerequisites

- Go 1.26.5 or later
- Git
- A GitHub account for GitHub-backed workflows

## Installation

```bash
# Clone the repository
git clone https://github.com/raithlin/gha.git
cd gha

# Build the binary locally
make build

# Build, then install gha into GOBIN or GOPATH/bin
make install
```

## Authentication and repository selection

GHA uses `GHA_GITHUB_TOKEN` when set. Otherwise, if `gh` is installed and
authenticated, it uses `gh auth token`. With neither credential, public API
requests remain unauthenticated; private-repository access and higher API rate
limits require authentication. A fine-grained token can be supplied through
`GHA_GITHUB_TOKEN`, restricted to the repositories GHA will access. For the
full current command surface, grant these repository permissions:

- `Contents: read` for branch metadata, comparisons, and releases
- `Pull requests: write` for inspection and `gha pr create`
- `Checks: read` for CI status in `gha review`
- `Actions: read` to observe the tag-triggered release workflow
- `Issues: read` for `gha prs --assigned`

`Pull requests: write` includes read access. Branch and release-tag pushes use
the checkout remote's authentication, not the GitHub API token. A classic
token needs the `repo` scope for private repositories.

Repository-aware commands select a repository in this order:

1. `--repo owner/repo`
2. the `origin` remote in an explicit `--path /path/to/checkout`
3. `GHA_REPOSITORY`
4. the current Git repository's `origin` remote

`--path` identifies a local checkout; it does not clone or fetch it.

## First commands

```bash
# See available commands and their help
gha --help

# See what this build can safely do, including unavailable features
gha capabilities --format json

# List pull requests in the current repository
gha prs

# Inspect one pull request
gha review 123

# Prepare a provider-safe pull request preview before its remote write
gha pr prepare --title "Improve reviews" --head feature/reviews

# Create only after reviewing the plan
gha pr create --title "Improve reviews" --head feature/reviews

# Inspect local branches and cached origin tracking branches
gha branches

# Review then explicitly refresh origin tracking refs when freshness matters
gha branches --refresh-origin --dry-run
gha branches --refresh-origin

# Analyze local worktree, history, storage, and largest tracked files offline
gha analyze

# Inspect a different local checkout without changing directory
gha branches --path ../other-checkout

# Generate read-only release notes from merged pull requests
gha release create-notes --since 2026-09-01

# List published releases with a bounded, versioned result.
gha releases --limit 10

# Review the tag, commit, CI, and workflow plan before publishing.
gha release publish 1.2.3 --dry-run --format json
gha release publish 1.2.3
```

For command-specific examples and automation contracts, see the
[command guide](command-guide.md).
