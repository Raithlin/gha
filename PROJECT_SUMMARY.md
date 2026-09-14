# GHA (GitHub Assistant) - Project Summary

## Overview
GHA is an agent-first developer tool written in Go. It turns GitHub, Git, CI,
and local-repository signals into bounded, versioned, decision-ready workflows
for coding agents, with readable terminal output for developers.

It complements rather than replaces `git` and `gh`: a GHA command should add
context, safety, or workflow value beyond a raw provider invocation.

## Current Status
**Phase 1 complete; core Phase 2 workflows delivered** (as defined in ARCHITECTURE.md)
- Basic CLI executable and Cobra command framework
- Startup configuration from environment variables
- GitHub provider behind an interface boundary
- Repository resolution from flags, configuration, or local Git remotes
- Repository-scoped pull request listings, single-PR review summaries, and guarded PR preparation and creation with text, JSON, and YAML output
- Offline local repository analysis
- Guarded local and origin branch lifecycle operations, including publication and safe checked-out-branch deletion
- Agent guidance installation and removal for Codex and Claude Code
- Installed build identity reporting
- Versioned capability inventory and structured diagnostics for automation

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
│   ├── branch/              # Branch inventory, safety, and publication workflows
│   ├── buildinfo/           # Installed build identity
│   ├── config/              # Startup configuration
│   ├── git/                 # Local Git resolution, analysis, inspection, and writes
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
9. **Agent-first contracts** - Structured output, bounded queries, explicit uncertainty, and safe mutations
10. **Human-readable parity** - Render the same domain result clearly for terminal users

## Current Commands
- `gha agent install|uninstall` - Safely manage bundled GHA guidance for Codex and Claude Code
- `gha version` and `gha capabilities` - Identify the installed build and enumerate its versioned command surface
- `gha pr prepare|create` - Preflight and, after explicit confirmation, create a pull request
- `gha prs` and `gha review <number>` - Bounded PR discovery and decision-ready single-PR inspection
- `gha analyze` - Offline local worktree, history, storage, and largest-file analysis
- `gha branches` - Local and cached `origin` inventory; origin refresh is explicit and confirmed
- `gha branch show|create|publish|rename|delete` - Provider-enriched branch safety and guarded local/origin lifecycle operations
- `gha release --since <timestamp>` - Temporary spelling for the read-only release-note generator; timezone-less values use the current timezone
- `gha dashboard` - Explicitly unavailable until the future TUI dashboard is implemented

## Branch Lifecycle Management

Branch management is a Phase 2 read/write workflow for developers, not merely
a remote-branch listing. Git provides the common capability for local and
`origin` branches so the workflow remains useful across GitHub, GitLab, and
future providers. Provider integrations add safety signals such as open pull or
merge requests, branch protection, permissions, and default-branch status.

Inventory, single-branch inspection, creation, publication, rename, and
deletion are implemented. Mutations state whether they target the local
repository, `origin`, or both; support `--dry-run`; and require confirmation
before remote changes. Default and protected branches are guarded from
destructive operations. Deleting a checked-out non-default branch switches to
the resolved safe default branch first and reports that transition.

The next branch workflow is explainable cleanup candidates. It must show why a
branch is eligible or excluded before any existing guarded deletion workflow
can act on it.

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
make lint      # Run golangci-lint with .golangci.yml
make clean     # Remove bin/
```

## Future Phases
- **Phase 2**: Complete the release command migration and add explainable branch-cleanup candidates
- **Phase 3**: Engineering metrics, hotspot analysis, risk scoring, ownership analysis
- **Phase 4**: TUI dashboard, plugins, multiple providers, offline cache, background refresh

## Related Documentation
- [ARCHITECTURE.md](ARCHITECTURE.md) - Current implementation details
- [DESIGN.md](DESIGN.md) - Long-term vision and design principles
- [docs/adr/ADR-001-Project-Layout.md](docs/adr/ADR-001-Project-Layout.md) - Architectural Decision Record for project structure
