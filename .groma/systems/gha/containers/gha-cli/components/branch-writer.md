---
type: C4 Component
title: Branch mutation adapter
status: stable
groma:
  id: branch-writer
  parent: gha-cli
  code:
    - scanner: go
      file: internal/git/branch_writer.go
  group: Branches and checkouts
  technology: Git subprocesses
description: Validates refs and executes selected local or origin branch changes.
---

Checks branch names, ref collisions, starting commits, and selected origin refs before creation, rename, publication, or deletion. It provides local merge/deletion checks and primitives for checkout switching. Commands add provider and cross-worktree policy. Remote changes use Git push and remote reads rather than implicitly refreshing cached tracking refs.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [branch-writer](branch-writer.md) | [git-and-repository-storage](../../../../../externals/git-and-repository-storage.md) | Mutates branch refs | Git subprocesses and origin transport |
