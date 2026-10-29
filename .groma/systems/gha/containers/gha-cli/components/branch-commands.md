---
type: C4 Component
title: Branch commands
status: stable
groma:
  id: branch-commands
  parent: gha-cli
  code:
    - scanner: go
      file: internal/commands/branch.go
    - scanner: go
      file: internal/commands/branches.go
    - scanner: go
      file: internal/commands/cleanup_apply.go
  group: Branches and checkouts
  technology: Go; Cobra
description: Coordinates branch inspection, publication and explicitly selected branch cleanup.
---

Branches lists local and cached-origin refs and can explicitly refresh origin. Cleanup inspection offers offline reachability or fresh merged-PR evidence through the branch service. Cleanup apply accepts explicit names or --all for a complete snapshot of at most 100 local branches. Bulk local cleanup automatically assesses associated linked worktrees and retains unsafe entries; a named force override still requires an explicit worktree path to discard checkout files. It captures a complete dry-run plan, gathers exact branch-name confirmations for every forced branch before any writes, and revalidates each pending branch separately so earlier deletions do not invalidate the remaining batch. Historical merged-PR evidence remains usable after GitHub deletes origin branches. Optional origin cleanup verifies repository identity and the exact remote tip, reports already-deleted refs as absent, and requires permission and protection signals only for present refs. Commit-bound writes are delegated to the mutation adapter. Missing force confirmation retains the entire plan. Results preserve per-branch completed, retained and failed effects. Other single-branch commands retain their existing provider checks, checkout transitions and documented force behavior.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/commands/branch.go](../../../../../../internal/commands/branch.go) | [internal/git/repository.go](../../../../../../internal/git/repository.go) | Validates origin target | Go call |
| [internal/commands/branch.go](../../../../../../internal/commands/branch.go) | [internal/branch/service.go](../../../../../../internal/branch/service.go) | Requests branch safety | Go service calls |
| [internal/commands/branch.go](../../../../../../internal/commands/branch.go) | [internal/git/branch_writer.go](../../../../../../internal/git/branch_writer.go) | Executes branch changes | Go adapter calls |
| [internal/commands/branch.go](../../../../../../internal/commands/branch.go) | [internal/git/worktrees.go](../../../../../../internal/git/worktrees.go) | Checks checkout occupancy | Go call |
| [internal/commands/branches.go](../../../../../../internal/commands/branches.go) | [internal/git/cleanup.go](../../../../../../internal/git/cleanup.go) | Requests cleanup candidates | Go call |
| [internal/commands/branch.go](../../../../../../internal/commands/branch.go) | [internal/output/review.go](../../../../../../internal/output/review.go) | Renders workflow result | Go output functions |
| [internal/commands/branches.go](../../../../../../internal/commands/branches.go) | [internal/branch/cleanup.go](../../../../../../internal/branch/cleanup.go) | Requests cleanup evidence | Go service calls |
| [internal/commands/cleanup_apply.go](../../../../../../internal/commands/cleanup_apply.go) | [internal/branch/cleanup.go](../../../../../../internal/branch/cleanup.go) | Revalidates cleanup evidence | Go service calls |
| [internal/commands/cleanup_apply.go](../../../../../../internal/commands/cleanup_apply.go) | [internal/git/cleanup_delete.go](../../../../../../internal/git/cleanup_delete.go) | Executes selected cleanup | Exact-tip Git adapter calls |
| [internal/commands/cleanup_apply.go](../../../../../../internal/commands/cleanup_apply.go) | [internal/output/cleanup.go](../../../../../../internal/output/cleanup.go) | Renders cleanup effects | Go output functions |
