---
type: C4 Component
title: Branch signal coordination
status: stable
groma:
  id: branch-service
  parent: gha-cli
  code:
    - scanner: go
      file: internal/branch/service.go
  group: Branches and checkouts
  technology: Go interfaces
description: Combines local branch facts with optional code-host safety signals.
---

Wraps a Lister and optional Inspector/OriginRefresher capabilities, and a separate SafetyProvider. Inventory stays local unless refresh is explicitly selected. Show preserves local facts if repository selection or provider signals fail, marking each safety signal unavailable. Commands own branch write orchestration; this service does not execute branch mutations.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/branch/service.go](../../../../../../internal/branch/service.go) | [internal/git/branches.go](../../../../../../internal/git/branches.go) | Reads branch inventory | Lister and Inspector interface dispatch |
| [internal/branch/service.go](../../../../../../internal/branch/service.go) | [internal/github/client.go](../../../../../../internal/github/client.go) | Enriches branch safety | SafetyProvider interface dispatch |
