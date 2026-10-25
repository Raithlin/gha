---
type: Groma Flow
title: Plan a branch deletion
groma:
  id: plan-a-branch-deletion
---

The deletion command owns the preflight and checks local Git, provider policy where required, and every registered worktree. A dry run renders the plan before the command would perform a mutation.

## Steps

| From | To | Action |
| --- | --- | --- |
| [Developer](../actors/developer.md) | [GHA CLI](../systems/gha/containers/gha-cli/container.md) | Run branch delete with an explicit local/origin target and --dry-run |
| [cli-runtime](../systems/gha/containers/gha-cli/components/cli-runtime.md) | [branch-commands](../systems/gha/containers/gha-cli/components/branch-commands.md) | Dispatch deletion preflight |
| [branch-commands](../systems/gha/containers/gha-cli/components/branch-commands.md) | [repository-context](../systems/gha/containers/gha-cli/components/repository-context.md) | Validate the selected origin target when needed |
| [branch-commands](../systems/gha/containers/gha-cli/components/branch-commands.md) | [branch-service](../systems/gha/containers/gha-cli/components/branch-service.md) | Request provider safety for origin deletion unless overridden |
| [branch-service](../systems/gha/containers/gha-cli/components/branch-service.md) | [github-provider](../systems/gha/containers/gha-cli/components/github-provider.md) | Read safety signals through SafetyProvider when needed |
| [branch-commands](../systems/gha/containers/gha-cli/components/branch-commands.md) | [worktree-operations](../systems/gha/containers/gha-cli/components/worktree-operations.md) | Reject a branch checked out in another worktree |
| [branch-commands](../systems/gha/containers/gha-cli/components/branch-commands.md) | [branch-writer](../systems/gha/containers/gha-cli/components/branch-writer.md) | Validate selected refs, merge safety, and checkout transition |
| [branch-commands](../systems/gha/containers/gha-cli/components/branch-commands.md) | [result-rendering](../systems/gha/containers/gha-cli/components/result-rendering.md) | Emit the plan without changing refs in dry-run mode |
