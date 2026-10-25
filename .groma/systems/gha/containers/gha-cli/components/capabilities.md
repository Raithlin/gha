---
type: C4 Component
title: Capability inventory
status: stable
groma:
  id: capabilities
  parent: gha-cli
  code:
    - scanner: go
      file: internal/commands/capabilities.go
  group: CLI and contracts
  technology: Cobra; model.Capabilities
description: Publishes the current machine-readable command contract.
---

Builds the versioned command inventory with availability, supported formats, read-only status, and workflow notes. Both the capabilities command and guidance compatibility checking consume this inventory. Dashboard is explicitly unavailable. The inventory describes the executable rather than planned roadmap commands.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/commands/capabilities.go](../../../../../../internal/commands/capabilities.go) | [internal/output/review.go](../../../../../../internal/output/review.go) | Renders workflow result | Go output functions |
