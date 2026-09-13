# Roadmap

GHA follows a phased approach to deliver workflows incrementally while
maintaining a stable automation contract.

## Delivered foundation

- Basic CLI structure with Cobra and a Go module
- Build system (`makefile`)
- Versioned, machine-readable command capability inventory
- GitHub-backed pull-request listings and single-PR review with decision-ready structured output

## Current work: core functionality

- Pull-request listing and filtering
- Repository-scoped review workflow with review state, risk signals, recommended actions, CI check-run, and unresolved-thread signals
- Read-only release notes and contributor summaries from merged pull requests
- Read-only local and `origin` branch inventory with tracking, divergence, and explicit freshness
- Read-only `branch show` safety inspection

Still planned in this phase:

- Local Git repository analysis
- Agent adoption workflows, including managed guidance installation and removal

### Planned branch lifecycle

The branch command surface is `gha branches` for inventory and `gha branch`
subcommands for inspection and write operations. Mutations declare
whether they affect the local repository, `origin`, or both; support `--dry-run`;
and require confirmation before remote changes. Default and protected branches
are guarded from destructive operations unless `--force` deliberately overrides
the safety guardrail.

### Planned release command migration

The current `gha release --since ...` command generates release notes. Its
singular name is misleading because it neither creates nor displays a GitHub
Release. The intended command contract is:

```bash
gha releases                         # List published GitHub releases
gha release show v0.1-alpha          # Inspect one published release
gha release create-notes --since ... # Generate notes from merged pull requests
```

`gha releases` and `gha release show` will use the GitHub Releases API. The
existing `gha release --since ...` spelling remains available only until the
command tree is migrated to `gha release create-notes`.

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
