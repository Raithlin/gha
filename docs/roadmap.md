# Roadmap

GHA follows a phased approach to deliver workflows incrementally while
maintaining a stable automation contract.

## Command-value guardrail

GHA does not replace `git` or `gh`. Every roadmap item must pass the
[Command Value Test](../DESIGN.md#command-value-test): it must combine signals,
provide a stable machine contract, make scope or risk explicit, or guide a
safe multi-step workflow. When a direct `git` or `gh` command is clearer, GHA
should say so in its help and documentation rather than duplicate it.

### Workflow automation

When a developer or agent explicitly asks for an outcome, GHA may orchestrate
the necessary Git and provider operations as one workflow. It must first
inspect the relevant state, make the planned local and remote effects explicit,
honour dry-run and confirmation boundaries, and return the steps that actually
occurred. Git and `gh` remain the underlying tools; GHA contributes the
decision-ready plan, safety checks, and reliable result contract.

## Delivered foundation

- Basic CLI structure with Cobra and a Go module
- Build system (`makefile`)
- Versioned, machine-readable command capability inventory
- GitHub-backed pull-request listings and single-PR review with decision-ready structured output
- Agent guidance installation for Codex and Claude Code, with bundled skills and explicit write confirmation

## Current work: core functionality

- Pull-request listings and filtering with a bounded, provider-normalized contract
- Repository-scoped review workflow with review state, risk signals, recommended actions, CI check-run, and unresolved-thread signals
- Read-only release notes and contributor summaries from merged pull requests
- Read-only local and `origin` branch inventory with tracking, divergence, and explicit freshness
- Read-only `branch show` safety inspection
- Offline local Git repository analysis of worktree, history, object storage, and largest tracked files
- Pull-request preparation and creation with explicit base/head resolution, comparison, existing-PR detection, dry runs, and confirmation

Still planned in this phase:

- Publish an existing committed local branch only as a guarded workflow that makes the target, upstream, divergence, push permission, and confirmation explicit. Direct `git push -u` remains appropriate for a straightforward publish.
- Explicit origin refresh for branch inventory, preserving the current no-implicit-fetch default and reporting freshness
- A `gha version` command for installed-build identification as foundation/operability work, not a developer workflow
- Managed guidance removal

### Planned branch lifecycle

The branch command surface is `gha branches` for inventory and `gha branch`
subcommands for inspection and write operations. Mutations declare
whether they affect the local repository, `origin`, or both; support `--dry-run`;
and require confirmation before remote changes. Default and protected branches
are guarded from destructive operations unless `--force` deliberately overrides
the safety guardrail. GHA branch writes earn their place only when they provide
that reviewable plan or provider-enriched safety context; use direct Git for a
simple ref operation.

Working-tree operations such as switching branches, staging, and committing
remain intentionally outside GHA's scope: native Git commands are clearer for
those direct local operations. GHA should instead own the workflows that add
safe, decision-ready context around them.

For example, deleting a merged branch that is currently checked out is not a
raw `git branch -d` replacement: GHA verifies that it is not the default or a
protected branch, switches to the default branch when safe, deletes the chosen
local and/or origin refs after confirmation, and records the checkout
transition in its result.

### Planned release command migration

The current `gha release --since ...` command generates release notes. Its
singular name is misleading because it neither creates nor displays a GitHub
Release. The intended command contract is:

```bash
gha releases                         # List published GitHub releases
gha release show v0.1-alpha          # Inspect one published release
gha release create-notes --since ... # Generate notes from merged pull requests
```

`gha releases` and `gha release show` should be added only when they combine
release data with decision-ready local or provider context, or offer a stable
automation contract that direct `gh release` output cannot. They must not be
aliases for the corresponding `gh` commands. The existing `gha release --since
...` spelling remains available only until the command tree is migrated to
`gha release create-notes`.

## Future phases

### Analytics

- Engineering metrics, such as cycle time and throughput
- Hotspot identification
- Pull-request and branch risk assessment
- Ownership and knowledge-distribution analysis

### Enhanced experience

- TUI dashboard with real-time updates
- Plugin architecture for extensibility
- Multiple provider support, including GitLab and Azure DevOps
- Offline caching and background synchronization
