---
type: C4 Component
title: Worktree commands
status: stable
groma:
  id: worktree-commands
  parent: gha-cli
  code:
    - scanner: go
      file: internal/commands/worktrees.go
  group: Branches and checkouts
  technology: Go; Cobra
description: Exposes bounded inventory and linked-worktree add/remove workflows.
---

Validates CLI options, selects the source checkout, and calls WorktreeService for listing or planned/executed mutations. Add can use an existing branch or explicitly create one; remove is limited to registered clean unlocked linked worktrees. Results use the same text/JSON/YAML rendering boundary.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/commands/worktrees.go](../../../../../../internal/commands/worktrees.go) | [internal/git/worktrees.go](../../../../../../internal/git/worktrees.go) | Coordinates worktree operations | Go service calls |
| [internal/commands/worktrees.go](../../../../../../internal/commands/worktrees.go) | [internal/output/review.go](../../../../../../internal/output/review.go) | Renders workflow result | Go output functions |
