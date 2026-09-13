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

Set `GHA_GITHUB_TOKEN` to a GitHub token that can read private repositories, or
to avoid unauthenticated GitHub API limits.

Repository-aware commands select a repository in this order:

1. `--repo owner/repo`
2. `GHA_REPOSITORY`
3. the `origin` remote in an explicit `--path /path/to/checkout`
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

# Inspect local branches and cached origin tracking branches
gha branches

# Inspect a different local checkout without changing directory
gha branches --path ../other-checkout

# Generate read-only release notes from merged pull requests
gha release --since 2026-09-01
```

For command-specific examples and automation contracts, see the
[command guide](command-guide.md).
