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

BranchLister reads refs, upstream divergence, current checkout, and sanitized origin metadata using Git. Origin remote-tracking information is reported as cached until an explicit refresh fetches and prunes origin; a refresh dry run does not contact origin. Cleanup classifies local branches by reachability from a selected local base and never deletes or infers provider safety.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [branch-inspection](branch-inspection.md) | [git-and-repository-storage](../../../../../externals/git-and-repository-storage.md) | Inspects branch refs | Git subprocesses; explicit fetch |
