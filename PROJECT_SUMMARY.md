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
│   │   ├── dashboard.go     # Dashboard placeholder
│   │   ├── release.go       # Release placeholder
│   │   ├── review.go        # Review command
│   │   ├── prs.go           # PR listing command
│   │   └── root.go          # Dependency-wired command tree
│   ├── config/              # Startup configuration
│   ├── git/                 # Local Git repository resolution
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
- `gha dashboard` - Future TUI dashboard
- `gha release --since <RFC3339>` - Read-only release notes and contributor summary
- `gha review <number>` - Review a single pull request
- `gha prs` - Repository-scoped pull request listings and filters

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
- **Phase 2**: Complete review assistance, release generation, branch management, local git analysis
- **Phase 3**: Engineering metrics, hotspot analysis, risk scoring, ownership analysis
- **Phase 4**: TUI dashboard, plugins, multiple providers, offline cache, background refresh

## Related Documentation
- [ARCHITECTURE.md](ARCHITECTURE.md) - Current implementation details
- [DESIGN.md](DESIGN.md) - Long-term vision and design principles
- [docs/adr/ADR-001-Project-Layout.md](docs/adr/ADR-001-Project-Layout.md) - Architectural Decision Record for project structure
