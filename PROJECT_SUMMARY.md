# GHA (GitHub Assistant) - Project Summary

## Overview
GHA is a developer productivity tool written in Go designed to help software developers make better engineering decisions by combining information from GitHub, Git, CI systems, issue trackers, and local repositories into a single cohesive experience.

## Current Status
**Phase 1 complete; Phase 2 workflows in progress** (as defined in ARCHITECTURE.md)
- Basic CLI executable and Cobra command framework
- Startup configuration from environment variables
- GitHub provider behind an interface boundary
- Repository resolution from flags, configuration, or local Git remotes
- Repository-scoped pull request listings and single-PR review summaries with text, JSON, and YAML output

## Project Structure
```
.
├── ARCHITECTURE.md          # Current architecture documentation
├── DESIGN.md                # Long-term vision and design principles
├── cmd/
│   └── gha/
│       └── main.go          # Application entry point
├── internal/
│   ├── commands/            # CLI command implementations
│   ├── branch/              # Read-only branch inventory workflow
│   │   ├── dashboard.go     # Dashboard placeholder
│   │   ├── release.go       # Release placeholder
│   │   ├── review.go        # Review command
│   │   ├── prs.go           # PR listing command
│   │   └── root.go          # Dependency-wired command tree
│   ├── config/              # Startup configuration
│   ├── git/                 # Local Git repository resolution and inspection
│   ├── github/              # GitHub provider
│   ├── interfaces/          # Provider boundary
│   ├── output/              # Text, JSON, and YAML rendering
│   └── review/              # Review workflows
├── pkg/
│   └── model/               # Shared data models
├── docs/
│   └── adr/
│       └── ADR-001-Project-Layout.md
├── go.mod                   # Go module definition
└── makefile                 # Build automation
```

## Key Architectural Principles
1. **Keep main() small** - Bootstrap only
2. **Responsibility-based packages** - Organize by what things do, not technical layers
3. **Inward dependencies** - Dependencies point toward core business logic
4. **Explicit dependencies** - Use constructor injection
5. **Avoid global state** - Pass dependencies explicitly
6. **Context everywhere** - Pass context.Context to blocking operations
7. **Errors as values** - Return and wrap errors with context
8. **Prefer standard library** - Minimize third-party dependencies

## Current Commands
- `gha capabilities` - Versioned inventory of available and unavailable commands
- `gha dashboard` - Explicitly unavailable until the future TUI dashboard is implemented
- `gha release --since <timestamp>` - Temporary spelling for the read-only release-note generator; timezone-less values use the current timezone
- `gha review <number>` - Review a single pull request
- `gha prs` - Bounded, versioned repository-scoped pull request listings and filters
- `gha branches` - Read-only local and cached `origin` branch inventory with explicit tracking, divergence, and truncation state

## Planned Branch Lifecycle Management

Branch management is a Phase 2 read/write workflow for developers, not merely
a remote-branch listing. Git provides the common capability for local and
`origin` branches so the workflow remains useful across GitHub, GitLab, and
future providers. Provider integrations add safety signals such as open pull or
merge requests, branch protection, permissions, and default-branch status.

The planned progression is inventory, single-branch inspection, create and
publish, then explicitly planned rename, deletion, and cleanup. Mutations will
state whether they target the local repository, `origin`, or both; support
`--dry-run`; and require confirmation before remote changes. Default and
protected branches are guarded from destructive operations.

## Release Command Naming Decision

Release discovery and release-note generation are different operations. The
planned command contract makes that explicit:

```text
gha releases                         List published GitHub releases
gha release show <tag>               Inspect one published release
gha release create-notes --since ... Generate notes from merged pull requests
```

Until that command tree is implemented, `gha release --since <timestamp>`
continues to generate notes only; it does not list, show, or create releases.

## Build & Development
```bash
make build     # Build binary to bin/gha
make run       # Run with go run
make test      # Run tests
make fmt       # Format code and tidy modules
make lint      # Run staticcheck
make clean     # Remove bin/
```

## Future Phases
- **Phase 2**: Complete review assistance, release generation, provider-neutral branch lifecycle management for local and origin branches, local git analysis
- **Phase 3**: Engineering metrics, hotspot analysis, risk scoring, ownership analysis
- **Phase 4**: TUI dashboard, plugins, multiple providers, offline cache, background refresh

## Related Documentation
- [ARCHITECTURE.md](ARCHITECTURE.md) - Current implementation details
- [DESIGN.md](DESIGN.md) - Long-term vision and design principles
- [docs/adr/ADR-001-Project-Layout.md](docs/adr/ADR-001-Project-Layout.md) - Architectural Decision Record for project structure
