---
type: C4 Component
title: Branch inspection and cleanup candidates
status: stable
groma:
  id: branch-inspection
  parent: gha-cli
  code:
    - scanner: go
      file: internal/git/branches.go
    - scanner: go
      file: internal/git/cleanup.go
  group: Branches and checkouts
  technology: Git subprocesses
description: Reads bounded local branch and cached-origin state.
---

BranchLister reads refs, upstream divergence, current checkout and sanitized origin metadata using Git. It checks exact configured upstream refs before comparison, reporting upstream_gone with unavailable counts when a ref is absent locally. Origin remote-tracking data remains cached until an explicit refresh fetches and prunes origin; refresh dry-run does not contact origin. Local cleanup classifies branches by reachability from a selected local base. When the branch service has verified merged-PR evidence, the adapter can preview linked-worktree retirement through existing PRWorktree safety checks. Both cleanup paths remain read-only.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [branch-inspection](branch-inspection.md) | [git-and-repository-storage](../../../../../externals/git-and-repository-storage.md) | Inspects branch refs | Git subprocesses; explicit fetch |
| [internal/git/cleanup.go](../../../../../../internal/git/cleanup.go) | [internal/git/pr_worktree.go](../../../../../../internal/git/pr_worktree.go) | Previews worktree retirement | PRWorktree dry-run |
