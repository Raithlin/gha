---
type: C4 Component
title: CLI bootstrap and command routing
status: stable
groma:
  id: cli-runtime
  parent: gha-cli
  code:
    - scanner: go
      file: internal/commands/root.go
    - scanner: go
      file: cmd/gha/main.go
    - scanner: go
      file: internal/commands/errors.go
    - scanner: go
      file: internal/commands/dashboard.go
      symbol: newDashboardCmd
  group: CLI and contracts
  technology: Go; Cobra
description: Wires concrete dependencies and dispatches the Cobra command tree.
---

The executable loads configuration and constructs GitHub, branch, review, release, and repository-resolution dependencies before executing a newly built command tree. Shared argument and error helpers keep structured diagnostics on stderr and prevent duplicate error printing in main. The registered dashboard command returns an unavailable error; it does not start a TUI.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [cmd/gha/main.go](../../../../../../cmd/gha/main.go) | [internal/config/config.go](../../../../../../internal/config/config.go) | Loads startup configuration | Go call |
| [cmd/gha/main.go](../../../../../../cmd/gha/main.go) | [internal/github/client.go](../../../../../../internal/github/client.go) | Constructs API adapter | Go constructor |
| [cmd/gha/main.go](../../../../../../cmd/gha/main.go) | [internal/branch/service.go](../../../../../../internal/branch/service.go) | Wires branch workflow | Go constructor injection |
| [cmd/gha/main.go](../../../../../../cmd/gha/main.go) | [internal/review/service.go](../../../../../../internal/review/service.go) | Wires review workflow | Go constructor injection |
| [cmd/gha/main.go](../../../../../../cmd/gha/main.go) | [internal/release/service.go](../../../../../../internal/release/service.go) | Wires release workflow | Go constructor injection |
| [internal/commands/root.go](../../../../../../internal/commands/root.go) | [internal/buildinfo/buildinfo.go](../../../../../../internal/buildinfo/buildinfo.go) | Reads build identity | Go call |
| [internal/commands/root.go](../../../../../../internal/commands/root.go) | [internal/commands/capabilities.go](../../../../../../internal/commands/capabilities.go) | Registers capability discovery | Cobra command construction |
| [internal/commands/root.go](../../../../../../internal/commands/root.go) | [internal/commands/review.go](../../../../../../internal/commands/review.go) | Registers PR workflows | Cobra command construction |
| [internal/commands/root.go](../../../../../../internal/commands/root.go) | [internal/commands/branch.go](../../../../../../internal/commands/branch.go) | Registers branch workflows | Cobra command construction |
| [internal/commands/root.go](../../../../../../internal/commands/root.go) | [internal/commands/worktrees.go](../../../../../../internal/commands/worktrees.go) | Registers worktree workflows | Cobra command construction |
| [internal/commands/root.go](../../../../../../internal/commands/root.go) | [internal/commands/analyze.go](../../../../../../internal/commands/analyze.go) | Registers repository analysis | Cobra command construction |
| [internal/commands/root.go](../../../../../../internal/commands/root.go) | [internal/commands/release.go](../../../../../../internal/commands/release.go) | Registers release workflows | Cobra command construction |
| [internal/commands/root.go](../../../../../../internal/commands/root.go) | [internal/commands/tag.go](../../../../../../internal/commands/tag.go) | Registers tag publication | Cobra command construction |
| [internal/commands/root.go](../../../../../../internal/commands/root.go) | [internal/commands/agent.go](../../../../../../internal/commands/agent.go) | Registers guidance management | Cobra command construction |
| [internal/commands/root.go](../../../../../../internal/commands/root.go) | [internal/commands/update.go](../../../../../../internal/commands/update.go) | Registers guidance refresh | Cobra command construction |
