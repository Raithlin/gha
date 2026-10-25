---
type: C4 System
title: GHA — GitHub Assistant
status: stable
groma:
  id: gha
description: Combines GitHub and local Git signals into agent-first developer workflows.
---

Coding agents are the primary automation consumers; developers use the same commands from a terminal. The current product is one locally executed Go CLI covering pull-request inspection and creation, branch and linked-worktree operations, local analysis, release discovery and tag publication, and agent guidance management. GitHub is the only implemented code-host adapter. There is no running server, dashboard, or product database. Git repositories and harness configuration remain local or external resources, not additional deployed GHA applications.
