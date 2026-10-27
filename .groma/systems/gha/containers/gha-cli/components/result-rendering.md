---
type: C4 Component
title: Result rendering
status: stable
groma:
  id: result-rendering
  parent: gha-cli
  code:
    - scanner: go
      file: internal/output/review.go
    - scanner: go
      file: internal/output/color.go
    - scanner: go
      file: internal/output/agent.go
    - scanner: go
      file: internal/output/worktrees.go
    - scanner: go
      file: internal/output/release_ci.go
      symbol: releaseCIText
  group: CLI and contracts
  technology: encoding/json; yaml.v3; terminal ANSI styling
description: Renders domain results as text, JSON, or YAML.
---

Commands supply workflow results and an io.Writer; shared format parsing and serialization serve machine consumers, while domain-specific terminal renderers expose readiness, risks, actions, and effects. Terminal strings are sanitized and styling is conditional on the output environment. The renderers also accept the release service Plan; they do not perform provider or Git operations.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/output/review.go](../../../../../../internal/output/review.go) | [pkg/model/model.go](../../../../../../pkg/model/model.go) | Serializes workflow contracts | Go structs; JSON and YAML |
| [internal/output/review.go](../../../../../../internal/output/review.go) | [internal/release/service.go](../../../../../../internal/release/service.go) | Renders release plan | Go release.Plan contract |
