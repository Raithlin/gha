# GHA - GitHub Assistant

![GitHub](https://img.shields.io/badge/go-1.26.5-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

GHA is an agent-first developer tool written in Go. It gives coding agents
bounded, versioned, decision-ready views of GitHub, Git, CI, and local
repositories; developers use the same commands through a clear terminal UI.

## Table of Contents
- [Overview](#overview)
- [Features](#features)
- [Getting Started](#getting-started)
- [Usage](#usage)
- [Architecture](#architecture)
- [Development](#development)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)

## Overview

GHA helps an agent move from discovery to a safe, useful next step without
scraping terminal text or reconstructing context from several tools. Its
structured output is the primary contract; its text output makes that same
information practical for a developer at a terminal.

GHA does not aim to replace `git` or `gh`. It earns a command when it makes a
workflow easier or safer by combining local and provider signals, normalizing
them into a stable model, or exposing an actionable summary. Raw Git and GitHub
operations remain the right tool when they are the clearer or more efficient
choice.

The project follows a phased approach to deliver these workflows incrementally
while maintaining a stable automation contract.

## Features

### Phase 1: CLI Foundation ✓
- [x] Basic CLI structure with Cobra
- [x] Command framework (dashboard, review, release, prs)
- [x] Go module setup
- [x] Build system (`makefile`)
- [x] Versioned, machine-readable command capability inventory
- [x] GitHub-backed PR listings and single-PR review with decision-ready structured output

### Phase 2: Core Functionality (In Progress)
- [x] PR listing and filtering command
- [x] Repository-scoped PR review workflow
- [x] Initial review summary with review state, risk signals, and recommended actions
- [x] CI check-run review signal
- [x] Unresolved-thread review signal
- [x] Read-only release notes and contributor summaries from merged pull requests
- [ ] Branch lifecycle management
  - [x] Read-only local and `origin` branch inventory with tracking, divergence, and explicit freshness
  - [ ] Provider-enriched safety signals and explicit write operations
    - [x] Read-only `branch show` safety inspection
- [ ] Local git repository analysis

### Phase 3: Analytics (Planned)
- [ ] Engineering metrics (cycle time, throughput, etc.)
- [ ] Hotspot identification (frequently changed files)
- [ ] Risk assessment for PRs/branches
- [ ] Ownership and knowledge distribution analysis

### Phase 4: Enhanced Experience (Planned)
- [ ] TUI dashboard with real-time updates
- [ ] Plugin architecture for extensibility
- [ ] Multiple provider support (GitLab, Azure DevOps, etc.)
- [ ] Offline caching and background synchronization

## Getting Started

### Prerequisites
- Go 1.26.5 or later
- Git
- GitHub account (for full functionality)

### Installation

```bash
# Clone the repository
git clone https://github.com/raithlin/gha.git
cd gha

# Build the binary locally
make build

# Build, then install gha into GOBIN or GOPATH/bin
make install
```

### Quick Start

```bash
# See available commands
gha --help

# See what this build can safely do, including unavailable features.
gha capabilities --format json

# List open pull requests in the current repository
gha prs

# Inspect local branches and the remote-tracking branches for origin
gha branches

# Inspect a different local checkout without changing the current directory
gha branches --path ../other-checkout

# List pull requests in the current repository or inspect one for review.
# A token is needed for private repositories and avoids API rate limits.
export GHA_GITHUB_TOKEN=your-token
gha review 123
gha prs --assigned
gha prs --queue

# Current release-note generator (command naming will change; see below)
gha release --since 2026-09-01
```

## Usage

```bash
# Display help
gha --help

# Command-specific help
gha prs --help
gha review --help
gha release --help
gha branches --help
gha dashboard --help
gha capabilities --help
```

### Capability Inventory for Automation

`gha capabilities --format json` is the versioned `Capabilities` v1 inventory
of every installed command. It reports `available` commands separately from
features that are intentionally `unavailable`; agents should use it before
planning work from this CLI. The current `dashboard` command is unavailable
and exits non-zero rather than pretending to launch a TUI.

### Agent-First Contract

Every available command is safe to inspect by default and exposes accurate
Cobra help. Data commands provide a versioned JSON contract on stdout; text is
the human rendering of the same underlying result. Diagnostics and structured
errors are written to stderr, so an agent never has to parse a mixed stream.

Listings are bounded and say when results were truncated. Signals that GHA
cannot establish are `unavailable`, not guesses. Future commands that mutate
local or remote state will require an explicit target and confirmation, and
will support `--dry-run`.

### Review Command Examples

Set `GHA_GITHUB_TOKEN` to a GitHub token that can read private repositories (or
to avoid unauthenticated GitHub API limits).
Repository-aware commands use `--repo owner/repo`, the `origin` remote in an
explicit `--path /path/to/checkout`, `GHA_REPOSITORY`, or the current Git
repository's `origin` remote to select a repository, in that order.

```bash
# Show help for review command
gha review --help

# Review a specific PR
gha review 123

# Inspect a repository outside the current directory
gha review 123 --repo owner/repo

# Resolve the repository from another local checkout's origin remote
gha review 123 --path ../other-checkout

# Change output format
gha review 123 --format json
```

### Branch Inventory for Automation

`gha branches --format json` returns the versioned `BranchInventory` v1 schema.
It reads only local Git state: `origin_branches` are cached remote-tracking refs
and GHA never fetches implicitly. `origin_state` is `cached`, `absent`, or
`unconfigured_cached`; agents must treat `cached` data as potentially stale.

Pass `--path /path/to/checkout` to inspect another local checkout. This is a
local path, not an `owner/repo` identifier; GHA does not clone or fetch it.

`limit`, `local_truncated`, and `origin_truncated` make bounded results
explicit. A local branch has `divergence_state` of `available`, `not_tracked`,
or `unavailable`; an origin branch is `not_applicable`. Ahead/behind counts
exist only when divergence is available.
Structured failures are written to stderr as `CommandError` v1 with a stable
code and message, leaving stdout reserved for successful data.

### Branch Safety Inspection

`gha branch show <name> --format json` returns the versioned
`BranchInspection` v1 schema. It reads the selected local and cached-origin
branch without fetching or changing state. When GHA can resolve a GitHub
repository, it also reports open pull requests, protection, caller push
permission, default-branch status, and mergeability. Each provider signal is
independently marked `available`, `unavailable`, or `not_applicable`; missing
data must never be interpreted as a negative safety result.

Use `--repo owner/repo` when the checkout's origin is not a GitHub remote, and
`--path /path/to/checkout` to inspect another local checkout.

### Review Summary JSON

`gha review <number> --format json` returns the versioned `ReviewSummary` v1
schema. It is the supported automation surface for single-PR inspection; text
output is intended for people.

```json
{
  "schema_version": "v1",
  "pull_request": { "number": 123, "title": "Improve automation" },
  "reviews": [],
  "readiness": {
    "mergeable": true,
    "ci_status": "success",
    "review_threads_state": "none",
    "approved_by": [],
    "changes_requested_by": [],
    "pending_reviewers": []
  },
  "risk_signals": [],
  "recommended_actions": []
}
```

`ci_status` is derived from GitHub check runs for the PR head commit and is one
of `success`, `pending`, `failure`, or `none`. `review_threads_state` is one of
`none`, `resolved`, or `unresolved`. Either signal is `unavailable` when GHA
cannot retrieve it. Fields in this schema will be changed additively within v1.

### Pull Request Listing Examples

```bash
# List open pull requests
gha prs

# List work relevant to you
gha prs --queue
gha prs --assigned
gha prs --mine

# Filter repository pull requests
gha prs --state all --author octocat --base main
gha prs --reviewer @me --format json

# Restrict a query to PRs updated at or after this RFC 3339 instant.
gha prs --since 2026-09-01T00:00:00Z --limit 20 --format json

# Resolve the target repository from another local checkout.
gha prs --path ../other-checkout --format json
```

`gha prs --format json` returns the versioned `PullRequestList` v1 schema.
It includes the selected `repository`, requested `limit`, and `truncated` so
agents never have to infer whether a bounded result may omit matching PRs.
All GitHub-backed commands emit `CommandError` v1 diagnostics to stderr for
JSON and YAML requests; successful data remains exclusively on stdout.

### Release Command Naming

The current `gha release --since ...` command generates release notes. Its
singular name is misleading because it neither creates nor displays a GitHub
Release. The intended command contract is:

```bash
gha releases                         # List published GitHub releases
gha release show v0.1-alpha          # Inspect one published release
gha release create-notes --since ... # Generate notes from merged pull requests
```

`gha releases` and `gha release show` will use the GitHub Releases API. The
existing `gha release --since ...` spelling remains available only until the
command tree is migrated to `gha release create-notes`.

### Current Release Notes Examples

The current `gha release --since ...` command is read-only: it does not create
a GitHub release or change a repository. Give it the inclusive start of the
release window; it derives notes from merged pull requests and writes them to
standard output.

```bash
# Generate terminal-readable release notes for the current repository.
# A date or timezone-less ISO datetime means midnight/local time on this machine.
gha release --since 2026-09-01

# An explicit offset remains authoritative.
gha release --repo owner/repo --since 2026-09-01T00:00:00Z --format json

# Use another checkout's origin remote to select the repository.
gha release --path ../other-checkout --since 2026-09-01
```

`gha release --format json` returns the versioned `ReleaseNotes` v1 schema.
Its `limit` and `truncated` fields make bounded release windows explicit.
`--since` accepts an RFC 3339 timestamp, an ISO datetime without a timezone, or
an ISO date. When no timezone is supplied, GHA uses the current machine
timezone; for example, in UTC+2, `2025-09-01` means
`2025-09-01T00:00:00+02:00` (the equivalent API instant is
`2025-08-31T22:00:00Z`).

## Architecture

GHA follows a clean, responsibility-based architecture:

```
cmd/
    gha/
        main.go              # Application entry point
internal/
    commands/                # CLI command implementations
    branch/                  # Read-only branch inventory workflow
    config/                  # Configuration management
    git/                     # Local Git repository resolution and inspection
    github/                  # GitHub API client
    interfaces/              # Provider boundaries
    output/                  # Rendering/output formatting
    review/                  # Pull request review workflows
pkg/
    model/                   # Shared data models
```

### Key Principles
- **Keep main() minimal** - Only responsible for bootstrapping
- **Dependency Inversion** - Dependencies point inward toward business logic
- **Explicit Dependencies** - Constructor injection, no globals
- **Context Everywhere** - All blocking operations accept context.Context
- **Error Handling** - Return and wrap errors with context
- **Testability** - Focus on behavior over implementation

See [ARCHITECTURE.md](ARCHITECTURE.md) and [DESIGN.md](DESIGN.md) for detailed documentation.

## Development

### Prerequisites
- Go 1.26.5+
- Make (optional, for using `makefile`)
- Git

### Commands

```bash
# Build the application
make build          # Creates bin/gha
make run            # Runs with go run
go build ./...      # Alternative build command

# Testing
make test           # Run all tests
go test ./...       # Alternative test command

# Code quality
make fmt            # Format code and tidy dependencies
make lint           # Run staticcheck
go fmt ./...        # Alternative formatting
go vet ./...        # Alternative vet

# Cleanup
make clean          # Remove bin/ directory
```

### Project Structure

```
.
├── .gitignore          # Git ignore rules
├── README.md           # This file
├── ARCHITECTURE.md     # Current architecture documentation
├── DESIGN.md           # Long-term vision and design principles
├── docs/
│   └── adr/            # Architectural Decision Records
│       └── ADR-001-Project-Layout.md
├── go.mod              # Go module definition
├── go.sum              # Go module checksums
├── makefile            # Build automation
├── PROJECT_SUMMARY.md  # Project overview and status
└── cmd/
    └── gha/
        └── main.go     # Application entry point
└── internal/
    ├── commands/       # CLI implementations
    ├── branch/         # Read-only branch inventory workflow
    │   ├── dashboard.go
    │   ├── release.go
    │   ├── review.go   # Review command
    │   ├── prs.go
    │   └── root.go
    ├── config/         # Startup configuration
    ├── git/            # Local Git repository resolution and inspection
    ├── github/         # GitHub provider
    ├── interfaces/     # Provider boundary
    ├── output/         # Text, JSON, and YAML rendering
    └── review/         # Pull request review workflows
└── pkg/
    └── model/          # Shared data models
```

## Roadmap

See [DESIGN.md](DESIGN.md) for detailed roadmap and feature breakdown by phase.

### Phase 1: CLI Foundation ✓
- Basic CLI executable
- Go module and build system
- Command framework setup
- GitHub-backed PR listings and versioned review summaries with JSON/YAML output

### Phase 2: Core Functionality (In Progress)
- PR listing and management
- Repository-scoped review workflows
- Release command migration: release listing, inspection, and explicit note generation
- Branch lifecycle management: provider-neutral local and `origin` workflows, enriched by provider safety signals
- Local repository analysis
- Agent adoption: a distributable GHA skill plus dedicated `gha agent install` and `gha agent uninstall` workflows that safely add or remove a GHA-managed instruction in agent guidance files, directing agents to GHA's bounded, structured workflows before generic GitHub tooling

The planned branch command surface is `gha branches` for inventory and
`gha branch` subcommands for inspection and write operations. Mutations will
declare whether they affect the local repository, `origin`, or both; they will
support `--dry-run` and require confirmation before remote changes. See
[DESIGN.md](DESIGN.md#branch-lifecycle-management) for the staged roadmap and
safety rules.

### Phase 3: Analytics
- Engineering metrics dashboard
- Hotspot and churn analysis
- Risk scoring for PRs
- Ownership and knowledge distribution

### Phase 4: Enhanced Experience
- TUI dashboard with real-time data
- Plugin architecture
- Multiple Git provider support
- Offline capabilities and sync

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please ensure your code follows:
- Go formatting standards (`gofmt`)
- Project coding standards (see [DESIGN.md](DESIGN.md))
- Include tests for new functionality
- Update documentation as needed

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by the need for better engineering tooling in AI-augmented development workflows
- Built with Go and Cobra for robust CLI experience
- Influenced by modern developer productivity tools and practices

---

**GHA - Helping developers make better engineering decisions, one commit at a time.**
