---
type: C4 Component
title: Agent guidance installation and ownership
status: stable
groma:
  id: agent-integration
  parent: gha-cli
  code:
    - scanner: go
      file: internal/commands/agent.go
    - scanner: go
      file: internal/commands/agent_list.go
    - scanner: go
      file: skills/gha/embed.go
  group: Agent guidance
  technology: Go embed; local Markdown and JSON files; SHA-256
description: Manages bundled skills and instruction blocks for supported coding agents.
---

Embeds the shipped SKILL.md and AGENT-GUIDANCE.md in the executable, detects or explicitly selects supported harnesses, installs/uninstalls managed blocks and skill files, and lists recorded installations. A versioned ownership JSON file under os.UserConfigDir()/gha tracks destinations and skill digests. Shared destinations and user-modified or unowned skills are preserved by ownership checks; writes use temporary-file replacement. This is local file integration, not a running agent service.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [agent-integration](agent-integration.md) | [coding-agent-harnesses](../../../../../externals/coding-agent-harnesses.md) | Installs managed guidance | Local Markdown files |
| [internal/commands/agent_list.go](../../../../../../internal/commands/agent_list.go) | [internal/output/review.go](../../../../../../internal/output/review.go) | Renders workflow result | Go output functions |
