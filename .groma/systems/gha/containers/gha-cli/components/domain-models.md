---
type: C4 Component
title: Domain result contracts
status: stable
groma:
  id: domain-models
  parent: gha-cli
  code:
    - scanner: go
      file: pkg/model/model.go
    - scanner: go
      file: pkg/model/worktrees.go
  group: CLI and contracts
  technology: Go structs; JSON and YAML tags
description: Defines shared workflow data and explicit uncertainty states.
---

Versioned structures represent pull-request summaries and preflights, branch inventories and mutations, worktree operations, release listings and notes, repository analysis, guidance state, capabilities, and command errors. JSON/YAML tags form the public automation interface. Truncation, cached-origin state, unavailable signals, and planned/completed effects are represented as data. The detailed release publication Plan is currently defined in internal/release rather than this package.
