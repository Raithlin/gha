# ADR-002: Use Cobra for CLI Framework

## Status
Accepted

## Context
We need to choose a library for building the command-line interface (CLI) for GHA.
The CLI should support subcommands, flags, and automatic help generation.
We want to avoid building a custom CLI parser from scratch to save development time and leverage a well-maintained library.

## Decision
We will use [Cobra](https://github.com/spf13/cobra) for the CLI framework.

## Consequences
### Positive
- Mature and widely used library in the Go ecosystem.
- Supports nested commands, flags, and automatic help generation.
- Integrates well with Viper for configuration (if needed in the future).
- Active community and good documentation.

### Negative
- Adds an external dependency (mitigated by its popularity and stability).
- Slight learning curve for team members unfamiliar with Cobra.

### Neutral
- Cobra is the de facto standard for CLIs in Go, so it aligns with community expectations.

## Related Documents
- ARCHITECTURE.md
- DESIGN.md
- ADR-0001 (Project Structure Decision)

## Status
Accepted
