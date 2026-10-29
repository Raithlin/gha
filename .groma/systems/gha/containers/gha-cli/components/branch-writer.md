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
    - scanner: go
      file: internal/git/cleanup_delete.go
  group: Branches and checkouts
  technology: Git subprocesses
description: Validates refs and executes selected local or origin branch changes.
---

Validates ref names, start commits and local/origin collisions, and performs explicit branch creation, publication, rename and deletion. Ordinary deletion retains Git merge checks unless its documented force override is selected. Coordinated cleanup accepts a freshly verified or explicitly confirmed branch plan, rechecks default/base and worktree protection, retires only a selected eligible linked checkout, deletes origin with an exact tip lease and deletes the local ref with compare-and-delete. It removes local branch configuration after ref deletion. Forced cleanup may discard selected checkout files but never bypasses locks, primary/current checkout protection or in-progress operation checks. Partial failures preserve observed effects and original tips for recovery.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [branch-writer](branch-writer.md) | [git-and-repository-storage](../../../../../externals/git-and-repository-storage.md) | Mutates branch refs | Git subprocesses and origin transport |
| [internal/git/cleanup_delete.go](../../../../../../internal/git/cleanup_delete.go) | [internal/git/worktrees.go](../../../../../../internal/git/worktrees.go) | Checks checkout protection | Local worktree inspection |
