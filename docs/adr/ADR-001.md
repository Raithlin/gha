# ADR-001: Project Structure Decision

## Status
Accepted

## Context
We need to establish the project structure for GHA (GitHub Assistant) that aligns with the architectural principles outlined in ARCHITECTURE.md and DESIGN.md. The project is currently in Phase 1 - CLI Foundation.

## Decision
We will adopt a responsibility-based package structure as outlined in ARCHITECTURE.md:

```
cmd/
    gha/
        main.go
internal/
    auth/
    cache/
    commands/
    config/
    git/
    github/
    output/
pkg/
    model/
test/
```

This structure separates concerns by responsibility rather than technical layers, keeping the main() function small, and ensuring dependencies point inward.

## Consequences
- **Positive**: Clear separation of concerns, maintainable codebase, explicit dependencies, testability
- **Negative**: May require more upfront planning to identify proper boundaries
- **Neutral**: Follows idiomatic Go practices for larger applications

## Related Documents
- ARCHITECTURE.md
- DESIGN.md
- Makefile (build commands)

## Status
Accepted
