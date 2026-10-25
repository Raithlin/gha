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
  group: Branches and checkouts
  technology: Go; Cobra
description: Coordinates inventory, safety inspection, and branch lifecycle operations.
---

Branches lists local and cached-origin refs and can explicitly refresh origin; cleanup only proposes reachability-based candidates. Single-branch commands resolve the write target, use service safety enrichment, and orchestrate Git preflight and create/publish/rename/delete effects directly. Deletion checks registered worktrees and may plan a safe checkout transition. Mutation execution is suppressed by --dry-run; --force overrides documented guards where supported.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/commands/branch.go](../../../../../../internal/commands/branch.go) | [internal/git/repository.go](../../../../../../internal/git/repository.go) | Validates origin target | Go call |
| [internal/commands/branch.go](../../../../../../internal/commands/branch.go) | [internal/branch/service.go](../../../../../../internal/branch/service.go) | Requests branch safety | Go service calls |
| [internal/commands/branch.go](../../../../../../internal/commands/branch.go) | [internal/git/branch_writer.go](../../../../../../internal/git/branch_writer.go) | Executes branch changes | Go adapter calls |
| [internal/commands/branch.go](../../../../../../internal/commands/branch.go) | [internal/git/worktrees.go](../../../../../../internal/git/worktrees.go) | Checks checkout occupancy | Go call |
| [internal/commands/branches.go](../../../../../../internal/commands/branches.go) | [internal/git/cleanup.go](../../../../../../internal/git/cleanup.go) | Requests cleanup candidates | Go call |
| [internal/commands/branch.go](../../../../../../internal/commands/branch.go) | [internal/output/review.go](../../../../../../internal/output/review.go) | Renders workflow result | Go output functions |
