# ADR-003: Provider Abstraction for External Services

## Status
Accepted

## Context
GHA needs to interact with external systems such as GitHub, Git, and potentially other platforms (e.g., GitLab, CI systems).
To maintain a clean architecture and ensure testability, we must abstract these external dependencies.
This allows us to swap implementations (e.g., for testing or supporting multiple platforms) without changing the core business logic.

## Decision
We will use the provider pattern with interfaces to abstract external services.
Each external system will have:
1. An interface defining the contract (in `internal/interfaces/`).
2. One or more implementations (in `internal/<provider>/`).
3. The core application logic (in `internal/services/` or commands) will depend only on the interfaces.

## Consequences
### Positive
- **Testability**: Easy to mock dependencies in unit tests.
- **Flexibility**: Can support multiple providers (e.g., GitHub and GitLab) by implementing the same interface.
- **Separation of Concerns**: Keeps external API details out of business logic.
- **Maintainability**: Changes to an external service only affect its provider implementation.

### Negative
- Slightly more initial setup (defining interfaces and implementations).
- Indirection may add minimal complexity (mitigated by clear boundaries).

### Neutral
- This pattern is common in clean and hexagonal architectures and aligns with the dependency inversion principle.

## Related Documents
- ARCHITECTURE.md (Section: Dependency Direction)
- DESIGN.md (Section: 2. Interfaces at Boundaries)
- ADR-0001 (Project Structure Decision)

## Status
Accepted
