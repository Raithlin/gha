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
    - scanner: go
      file: internal/output/cleanup.go
      symbol: CleanupExecution
    - scanner: go
      file: internal/output/release_status.go
      symbol: ReleaseStatus
    - scanner: go
      file: internal/output/release_assets.go
  group: CLI and contracts
  technology: encoding/json; yaml.v3; terminal ANSI styling
description: Renders domain results as text, JSON, or YAML.
---

Commands supply workflow results and an io.Writer. Shared format parsing and serialization serve machine consumers, while terminal renderers expose readiness, risks, actions, and effects. Terminal strings are sanitized and styling depends on the output environment. Release renderers show publication plans, status, bounded asset inventories, and per-asset verification evidence. Renderers do not perform provider or Git operations.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/output/review.go](../../../../../../internal/output/review.go) | [pkg/model/model.go](../../../../../../pkg/model/model.go) | Serializes workflow contracts | Go structs; JSON and YAML |
| [internal/output/review.go](../../../../../../internal/output/review.go) | [internal/release/service.go](../../../../../../internal/release/service.go) | Renders release plan | Go release.Plan contract |
| [internal/output/cleanup.go](../../../../../../internal/output/cleanup.go) | [pkg/model/cleanup.go](../../../../../../pkg/model/cleanup.go) | Serializes cleanup effects | Go structs; JSON and YAML |
