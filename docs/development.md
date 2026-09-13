# Development

## Prerequisites

- Go 1.26.5 or later
- Make (optional, for using `makefile`)
- Git
- [golangci-lint](https://golangci-lint.run/docs/welcome/install/) for local linting

## Commands

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
make lint           # Run golangci-lint with .golangci.yml
go fmt ./...        # Alternative formatting
go vet ./...        # Alternative vet

# Cleanup
make clean          # Remove bin/ directory

# Build the app and install its bundled GHA skill for Codex, Claude Code, or both.
make skill-install

# Verify the product installer is repeatable.
make test-skill-install
```

## Repository layout

```text
.
├── README.md           # Project entry point
├── ARCHITECTURE.md     # Current architecture documentation
├── DESIGN.md           # Long-term vision and design principles
├── docs/               # Guides and architecture decision records
├── cmd/gha/            # Application entry point
├── internal/
│   ├── commands/       # CLI command implementations
│   ├── branch/         # Branch inventory and safety inspection workflows
│   ├── config/         # Startup configuration
│   ├── git/            # Local Git repository resolution and inspection
│   ├── github/         # GitHub provider
│   ├── interfaces/     # Provider boundary
│   ├── output/         # Text, JSON, and YAML rendering
│   └── review/         # Pull-request review workflows
├── pkg/model/          # Shared data models
└── makefile            # Build automation
```

See [Architecture](../ARCHITECTURE.md) for the current implementation and
[Design](../DESIGN.md) for long-term principles.

## Contributing

Contributions are welcome.

1. Fork the repository.
2. Create a feature branch: `git checkout -b feature/amazing-feature`.
3. Commit your changes: `git commit -m 'Add amazing feature'`.
4. Push the branch: `git push origin feature/amazing-feature`.
5. Open a pull request.

Before opening a pull request, format Go code, include relevant tests, and
update documentation when a user-facing workflow or contract changes.
