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
make check          # Mirror CI: dependencies, tidy check, lint, race tests, and coverage

# Code quality
make fmt            # Format code and tidy dependencies
make lint           # Run golangci-lint with .golangci.yml
go fmt ./...        # Alternative formatting
go vet ./...        # Alternative vet

# Cleanup
make clean          # Remove bin/ directory

# Build the app and choose one or more supported coding-agent harnesses.
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

## Preparing a release

The tag workflow publishes a reviewed Markdown file from the tagged commit.
Copy [the template](releases/TEMPLATE.md) to `docs/releases/<tag>.md`, using the
exact tag name (for example, `docs/releases/v0.4.0-alpha.md`).

1. Run `gha release create-notes --since <previous-release-time> --format json`
   to inventory merged pull requests. Review commits in the release range too:
   the GHA command reports merged PRs, not every direct commit.
2. Write the release summary and link changes that matter to users. Exclude
   administrative PRs from the notes, and mention human contributors with
   their GitHub `@` handles. The [v0.3.0-alpha notes](releases/v0.3.0-alpha.md)
   show the agreed format.
3. Replace every `REPLACE_ME` token, review the file, and merge it into the
   default branch before tagging. The release workflow fails on a missing,
   empty, or unfinished notes file.
4. From the current default-branch commit, run
   `gha release publish <tag> --dry-run --format json`. Verify that
   `release_notes.state` is `available` and its path is the reviewed file.
   After reviewing the plan, run the command without `--dry-run` to publish the tag. Wait for
   the Actions release run, then verify the GitHub Release, notes, prerelease
   status, and downloads. A tag push alone does not prove publication.

The workflow passes this versioned file to GoReleaser with `--release-notes`.
The bundled GHA preflight reads the committed workflow's
`GHA_RELEASE_NOTES_DIR` setting and checks the matching file at `HEAD`, so a
missing file blocks the tag before the workflow runs. Other repositories
without that setting retain the existing release-publication behavior.
