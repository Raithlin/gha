---
type: C4 Component
title: Compatible guidance refresh
status: stable
groma:
  id: guidance-update
  parent: gha-cli
  code:
    - scanner: go
      file: internal/commands/update.go
    - scanner: go
      file: internal/commands/skill_compatibility.go
  group: Agent guidance
  technology: Go net/http; YAML frontmatter; local files
description: Refreshes recorded harness guidance from a published GHA release.
---

Queries published GHA releases including prereleases, retrieves the selected release skill and instruction files from GitHub, and checks the skill requirements against this executable capability inventory. It updates only recorded managed targets through installation helpers and digest checks, or reports incompatibility/unavailability. --dry-run avoids writes. It does not replace the GHA executable.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [guidance-update](guidance-update.md) | [coding-agent-harnesses](../../../../../externals/coding-agent-harnesses.md) | Refreshes managed guidance | Local Markdown files |
| [guidance-update](guidance-update.md) | [github](../../../../../externals/github.md) | Downloads released guidance | HTTPS REST and raw content |
| [internal/commands/update.go](../../../../../../internal/commands/update.go) | [internal/commands/agent.go](../../../../../../internal/commands/agent.go) | Refreshes owned installations | Go helper calls |
| [internal/commands/update.go](../../../../../../internal/commands/update.go) | [internal/commands/capabilities.go](../../../../../../internal/commands/capabilities.go) | Checks skill compatibility | Go capability inventory |
| [internal/commands/update.go](../../../../../../internal/commands/update.go) | [internal/buildinfo/buildinfo.go](../../../../../../internal/buildinfo/buildinfo.go) | Reports binary identity | Go build variable |
| [internal/commands/update.go](../../../../../../internal/commands/update.go) | [internal/output/review.go](../../../../../../internal/output/review.go) | Renders workflow result | Go output functions |
