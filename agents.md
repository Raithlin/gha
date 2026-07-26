# Agent Development Guidelines for GHA

This document outlines the practices and guidelines for AI agents working on the GHA (GitHub Assistant) project.

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
1. Add the command to `internal/commands/root.go` in the `init()` function
2. Create a dedicated file for the command in `internal/commands/`
3. Use Cobra for command structure and flag handling
4. Implement the actual logic in separate functions (not directly in the RunE function)
5. Add comprehensive tests

### Error Handling
- Always check and handle errors from function calls
- Use `fmt.Errorf()` to wrap errors with context when appropriate
- Return errors from functions rather than panicking (except for truly unrecoverable situations)

### Configuration
- Configuration should be loaded once at startup
- Use the `config` package for managing configuration
- Support configuration via flags, environment variables, and config files
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

*Last updated: July 26, 2026*