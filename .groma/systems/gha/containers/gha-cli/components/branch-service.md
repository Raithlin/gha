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
    - scanner: go
      file: internal/branch/cleanup.go
  group: Branches and checkouts
  technology: Go interfaces
description: Combines local branch facts with optional code-host safety signals.
---

Coordinates local branch inventory and optional code-host safety signals through interfaces. Inventory stays local unless origin refresh is explicitly selected. Show preserves local facts when provider signals fail. Cleanup preserves offline reachability inspection or requires fresh merged-PR evidence bound to the checkout repository, integration base and exact local tip. Multiple PR identities, unavailable evidence and post-merge commits retain branches. Linked worktrees remain excluded; matching PR heads can include a read-only retirement assessment from the Git adapter. Commands own branch writes; this service never executes cleanup mutations.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/branch/service.go](../../../../../../internal/branch/service.go) | [internal/git/branches.go](../../../../../../internal/git/branches.go) | Reads branch inventory | Lister and Inspector interface dispatch |
| [internal/branch/service.go](../../../../../../internal/branch/service.go) | [internal/github/client.go](../../../../../../internal/github/client.go) | Enriches branch safety | SafetyProvider interface dispatch |
| [internal/branch/cleanup.go](../../../../../../internal/branch/cleanup.go) | [internal/git/cleanup.go](../../../../../../internal/git/cleanup.go) | Reads cleanup candidates | Cleanup and worktree-planner interfaces |
| [internal/branch/cleanup.go](../../../../../../internal/branch/cleanup.go) | [internal/github/client.go](../../../../../../internal/github/client.go) | Verifies merged PR heads | Bounded provider reads |
