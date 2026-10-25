---
type: Groma Flow
title: Refresh coding-agent guidance
groma:
  id: refresh-coding-agent-guidance
---

The updater works only on recorded installations. It downloads released guidance, validates declared capabilities against the current binary, and preserves incompatible or locally changed skills. The executable itself is not updated.

## Steps

| From | To | Action |
| --- | --- | --- |
| [coding-agent-harnesses](../externals/coding-agent-harnesses.md) | [GHA CLI](../systems/gha/containers/gha-cli/container.md) | Invoke gha update --format json, optionally with --dry-run |
| [cli-runtime](../systems/gha/containers/gha-cli/components/cli-runtime.md) | [guidance-update](../systems/gha/containers/gha-cli/components/guidance-update.md) | Dispatch guidance refresh |
| [guidance-update](../systems/gha/containers/gha-cli/components/guidance-update.md) | [github](../externals/github.md) | Read published releases and retrieve the selected guidance |
| [guidance-update](../systems/gha/containers/gha-cli/components/guidance-update.md) | [capabilities](../systems/gha/containers/gha-cli/components/capabilities.md) | Check required commands against the local capability inventory |
| [guidance-update](../systems/gha/containers/gha-cli/components/guidance-update.md) | [agent-integration](../systems/gha/containers/gha-cli/components/agent-integration.md) | Validate ownership and digests and update only compatible recorded targets when not a dry run |
| [guidance-update](../systems/gha/containers/gha-cli/components/guidance-update.md) | [coding-agent-harnesses](../externals/coding-agent-harnesses.md) | Refresh managed skill files and instruction blocks only on execution |
| [guidance-update](../systems/gha/containers/gha-cli/components/guidance-update.md) | [result-rendering](../systems/gha/containers/gha-cli/components/result-rendering.md) | Report source, compatibility, and per-target outcomes |
