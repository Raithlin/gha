# GHA - GitHub Assistant

![GitHub](https://img.shields.io/badge/go-1.26.5-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

GHA is a developer productivity tool written in Go designed to help software developers make better engineering decisions by combining information from GitHub, Git, CI systems, issue trackers, and local repositories into a single cohesive experience.

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

GitHub Assistant (GHA) helps development teams navigate the increasing complexity of modern software development by providing:
- Unified view of GitHub data (PRs, issues, reviews)
- Local repository analysis
- CI/CD integration insights
- Engineering metrics and reporting
- TUI dashboard for real-time monitoring

The project follows a phased approach to deliver value incrementally while maintaining architectural integrity.

## Features

### Phase 1: CLI Foundation ✓
- [x] Basic CLI structure with Cobra
- [x] Command framework (dashboard, review, release, prs)
- [x] Go module setup
- [x] Build system (Makefile)
- [x] GitHub-backed PR listings and single-PR review with structured output

### Phase 2: Core Functionality (In Progress)
- [x] PR listing and filtering command
- [x] Repository-scoped PR review workflow
- [ ] Review assistance and context gathering
- [ ] Release notes and changelog generation
- [ ] Branch management utilities
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

# Build the binary
make build

# Or install directly
go install ./cmd/gha
```

### Quick Start

```bash
# See available commands
gha --help

# List open pull requests in the current repository
gha prs

# Start the dashboard (placeholder)
gha dashboard

# List pull requests in the current repository or inspect one for review.
# A token is needed for private repositories and avoids API rate limits.
export GHA_GITHUB_TOKEN=your-token
gha review 123
gha prs --assigned
gha prs --queue

# Manage releases (placeholder)
gha release
```

## Usage

```bash
# Display help
gha --help

# Command-specific help
gha prs --help
gha review --help
gha release --help
gha dashboard --help
```

### Review Command Examples

Set `GHA_GITHUB_TOKEN` to a GitHub token that can read private repositories (or
to avoid unauthenticated GitHub API limits).
The command uses `--repo owner/repo`, `GHA_REPOSITORY`, or the current Git
repository's `origin` remote to select a repository.

```bash
# Show help for review command
gha review --help

# Review a specific PR
gha review 123

# Inspect a repository outside the current directory
gha review 123 --repo owner/repo

# Change output format
gha review 123 --format json
```

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
```

## Architecture

GHA follows a clean, responsibility-based architecture:

```
cmd/
    gha/
        main.go              # Application entry point
internal/
    commands/                # CLI command implementations
    config/                  # Configuration management
    git/                     # Local Git repository resolution
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
- Make (optional, for using Makefile)
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
├── Makefile            # Build automation
├── PROJECT_SUMMARY.md  # Project overview and status
└── cmd/
    └── gha/
        └── main.go     # Application entry point
└── internal/
    ├── commands/       # CLI implementations
    │   ├── dashboard.go
    │   ├── release.go
    │   ├── review.go   # Review command
    │   ├── prs.go
    │   └── root.go
    ├── config/         # Startup configuration
    ├── git/            # Local Git repository resolution
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
- GitHub-backed review command with filtering and JSON/YAML output

### Phase 2: Core Functionality (In Progress)
- PR listing and management
- Repository-scoped review workflows
- Release automation
- Local repository analysis
- Branch management utilities

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
