---
type: C4 Component
title: Worktree operations
status: stable
groma:
  id: worktree-operations
  parent: gha-cli
  code:
    - scanner: go
      file: internal/git/worktrees.go
  group: Branches and checkouts
  technology: Git subprocesses; filesystem path checks
description: Inspects registered checkouts and guards linked-worktree changes.
---

Parses Git worktree porcelain records, reads status per checkout, resolves canonical paths, and reports bounded inventory with unavailable status details where needed. Add/remove have planning methods and execution methods with Git safety checks. BranchPath and OtherBranchPath are also used by branch deletion to prevent deleting a branch checked out elsewhere.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [worktree-operations](worktree-operations.md) | [git-and-repository-storage](../../../../../externals/git-and-repository-storage.md) | Manages linked checkouts | Git subprocesses and filesystem |
