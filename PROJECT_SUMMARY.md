# GHA (GitHub Assistant) - Project Summary

## Overview
GHA is a developer productivity tool written in Go designed to help software developers make better engineering decisions by combining information from GitHub, Git, CI systems, issue trackers, and local repositories into a single cohesive experience.

## Current Status
**Phase 1 – CLI Foundation** (as defined in ARCHITECTURE.md)
- Basic CLI executable structure
- Go module initialized
- Command framework in place
- Configuration and GitHub integration planned

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
│   │   ├── dashboard.go     # Placeholder
│   │   ├── release.go       # Placeholder
│   │   ├── review.go        # Placeholder
│   │   ├── prs.go           # Placeholder
│   │   └── root.go          # Root command (empty)
│   ├── auth/                # Authentication (planned)
│   ├── cache/               # Caching (planned)
│   ├── config/              # Configuration (planned)
│   ├── git/                 # Git provider (planned)
│   ├── github/              # GitHub provider (planned)
│   └── output/              # Output rendering (planned)
├── pkg/
│   └── model/               # Shared data models (planned)
├── docs/
│   └── adr/
│       └── ADR-001.md       # Architectural Decision Record: Project Structure
├── .vscode/                 # VS Code configuration
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

## Current Commands (Placeholders)
- `gha dashboard` - Future TUI dashboard
- `gha release` - Release management
- `gha review` - Pull request review assistance
- `gha prs` - Pull request listing and management

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
- **Phase 2**: Review helper, release generation, branch management, local git analysis
- **Phase 3**: Engineering metrics, hotspot analysis, risk scoring, ownership analysis
- **Phase 4**: TUI dashboard, plugins, multiple providers, offline cache, background refresh

## Related Documentation
- [ARCHITECTURE.md](ARCHITECTURE.md) - Current implementation details
- [DESIGN.md](DESIGN.md) - Long-term vision and design principles
- [docs/adr/ADR-001.md](docs/adr/ADR-001.md) - Architectural Decision Record for project structure