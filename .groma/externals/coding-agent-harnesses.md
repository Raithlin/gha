---
type: C4 System
title: Coding agent harnesses
status: stable
groma:
  id: coding-agent-harnesses
  technology: CLI subprocesses and local Markdown files
---

External coding-agent applications invoke the GHA executable and consume its structured command results. GHA can install bundled skills and managed instruction blocks into supported harness configuration locations; the harness runtimes remain outside this repository.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [coding-agent-harnesses](coding-agent-harnesses.md) | [gha-cli](../systems/gha/containers/gha-cli/container.md) | Invokes workflows | Subprocess; JSON stdout |
