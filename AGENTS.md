# Agent Development Guidelines for GHA

This document outlines the practices and guidelines for AI agents working on the
GHA (GitHub Assistant) project.

## Product Direction

GHA is an **agent-first developer tool**. Coding agents are the primary
consumer of its command contracts; developers must be able to use those same
commands comfortably in a terminal. GHA complements `git` and `gh` rather than
reimplementing their raw command surfaces.

## GHA skill

When working on GHA and a local GHA command can inform implementation,
acceptance criteria, or validation, use the installed `gha` skill. Build the
local binary with the platform-appropriate project command, establish its
current surface with `capabilities --format json`, and prefer its structured
output to raw provider output or ad-hoc Git parsing. The skill's guidance on
cached and unavailable signals and branch-write dry-runs remains in force.

Add a command only when it makes a workflow easier or safer: by combining
signals, exposing a stable model, making uncertainty explicit, or guiding a
safe next step. A thin alias or reformatted copy of an existing `git` or `gh`
command is not sufficient value.

## Development Process

### Test-Driven Development (TDD)
All development on this project should follow the TDD approach:
1. **RED**: Write a failing test that defines the desired behavior
2. **GREEN**: Write the minimum code to make the test pass
3. **REFACTOR**: Improve the code while keeping tests passing

### Testing Framework
- Primary: Go's built-in `testing` package
- Assertions: `github.com/stretchr/testify/assert` and `github.com/stretchr/testify/mock`
- Test files should be named `*_test.go` and placed in the same package as the code they test
- Test coverage should remain a minimum of 95%

### Running Tests
```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/commands/

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...
```

## Project Structure Guidelines

### Package Organization
- `cmd/`: Main application entry points
- `internal/`: Application-specific code (not meant for reuse)
  - `commands/`: CLI command implementations
  - `config/`: Configuration handling
  - `github/`: GitHub API client
  - `git/`: Git operations
  - `output/`: Rendering and output formatting
- `pkg/`: Libraries that could be reused by other projects
  - `model/`: Shared data models
- `docs/`: Documentation including Architectural Decision Records (ADRs)

### Coding Standards
- Follow Go formatting standards (`gofmt`)
- Use meaningful names for variables, functions, and types
- Keep functions focused and small (aim for < 50 lines)
- Handle errors explicitly - don't ignore them
- Use context.Context for cancellation and timeouts
- Dependencies should point inward (clean architecture principles)

## Git Workflow

### Branching
- `main`: Production-ready code
- Feature branches: `feature/description` or `fix/description`
- Use descriptive branch names

### Commits
- Write clear, descriptive commit messages
- Follow the format: `type: description`
  - Types: feat, fix, docs, style, refactor, test, chore
- Example: `feat: add PR review command with flags`

### Pull Requests
- Keep PRs focused on a single concern
- Include tests for new functionality
- Update documentation as needed
- Request review from at least one team member

## Code Review Checklist

### For Code Changes
- [ ] Does the code follow Go formatting standards?
- [ ] Are there tests for new functionality?
- [ ] Do existing tests still pass?
- [ ] Is error handling appropriate?
- [ ] Is the code readable and maintainable?
- [ ] Are comments useful and up-to-date?

### For Documentation
- [ ] Is the README updated if needed?
- [ ] Are ADRs created for significant architectural decisions?
- [ ] Are inline comments clear and helpful?

## Specific to GHA Project

### Command Implementation
When implementing new commands:
1. Add the command through the explicit command-tree constructor in `internal/commands/root.go`
2. Create a dedicated file for the command in `internal/commands/`
3. Use Cobra for command structure and flag handling
4. Implement the actual logic in separate functions (not directly in the RunE function)
5. Add comprehensive tests
6. Verify both the structured contract and terminal-readable output when the
   command returns data
7. Ask questions rather than assume

### Agent-First CLI Contract
GHA is a machine interface for coding agents and a human CLI built from the
same domain results. Treat its automation behaviour as a public contract and
do not make text parsing necessary for correct use.

- Every data command must support `--format json`; agents must not need to
  parse human-readable text. Text should render the same result clearly for a
  developer.
- JSON output should be structured around GHA domain models, documented with
  examples, and changed additively where possible. Renaming or removing a field
  requires an explicit compatibility decision.
- Write the requested data only to stdout. Diagnostics, progress, and errors
  belong on stderr so JSON output remains parseable.
- A successful command, including one with an empty result, exits with code 0.
  Validation and operational failures must exit non-zero and provide an
  actionable error message. When structured errors are introduced, preserve a
  stable error code as well as the human-readable message.
- Listing commands should provide bounded, incremental queries such as
  `--limit`, filters, and time-based selection where the provider supports it.
  Include whether the result was truncated; do not require agents to retrieve
  or locally parse an unbounded result set.
- Explicitly model freshness and uncertainty. For example, cached local Git
  data must say that it is cached; unsupported or failed optional signals must
  be `unavailable`, never inferred.
- Agent conveniences such as `@me` must resolve through the authenticated
  provider identity, not assumptions about local Git configuration.
- Apply target selection consistently: `--repo` is an explicit provider target;
  `--path` is an explicit local checkout. Document precedence and never clone,
  fetch, or change state merely to resolve a read-only target.
- New mutating commands must support `--dry-run` and require an explicit
  confirmation flag and target before making a local or external change.
  Read-only inspection commands must remain safe by default.
- Keep commands discoverable through accurate Cobra help. When a machine-readable
  capability inventory is added, keep `gha capabilities --format json` complete
  and backward-compatible.

### Review Workflow Output
`gha review <number>` is the single-PR inspection workflow; `gha prs` owns PR
listings. As review assistance grows, its structured result should expose
decision-ready information rather than a raw provider response:

- merge and CI readiness
- required or requested reviewers that are still missing
- unresolved review threads and review state
- risk signals with severity and supporting detail
- recommended next actions an agent can take or propose

Keep raw source data available where useful, but make the stable summary the
primary automation surface.

### Error Handling
- Always check and handle errors from function calls
- Use `fmt.Errorf()` to wrap errors with context when appropriate
- Return errors from functions rather than panicking (except for truly unrecoverable situations)

### Configuration
- Configuration should be loaded once at startup
- Use the `config` package for managing configuration
- Support configuration via flags and environment variables; add configuration
  files only when the use case justifies the extra complexity
- Defaults should be sensible and secure

## AI Agent Specific Guidelines

### Context Awareness
- Always read relevant files before making changes
- Understand the existing architecture before implementing new features
- Check for similar patterns in the codebase to maintain consistency

### Communication
- When uncertain about requirements, ask for clarification
- Provide clear explanations of changes made
- Reference related issues, ADRs, or documentation when appropriate

### Quality Focus
- Prioritize correctness over speed
- Write tests before implementing features
- Keep the codebase clean and maintainable
- Follow the existing code style and patterns

## Example Workflow

Here's an example of how to add a new feature using TDD:

1. **Understand the requirement**: Read the issue or feature request carefully
2. **Explore the codebase**: Look at similar implementations for guidance
3. **Write tests**: Create a test file that defines the expected behavior
4. **Run tests**: Watch them fail (RED)
5. **Implement**: Write the minimum code to make tests pass (GREEN)
6. **Refactor**: Improve the code structure while keeping tests passing
7. **Update documentation**: Add any necessary documentation updates
8. **Submit for review**: Create a pull request with clear description

---

*Last updated: September 13, 2026*
