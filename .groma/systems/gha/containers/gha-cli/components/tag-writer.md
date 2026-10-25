---
type: C4 Component
title: Tag mutation adapter
status: stable
groma:
  id: tag-writer
  parent: gha-cli
  code:
    - scanner: go
      file: internal/git/tag.go
  group: Release publication
  technology: Git subprocesses
description: Inspects exact tags and creates or pushes their refs.
---

Validates tag names, resolves a commit, and checks local and live-origin tag existence. Create makes a lightweight tag without a message or an annotated tag with a message; Push specifies the exact source and destination refs. Both direct tag publication and the release checkout adapter reuse this responsibility.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [tag-writer](tag-writer.md) | [git-and-repository-storage](../../../../../externals/git-and-repository-storage.md) | Publishes exact tags | Git subprocesses and origin transport |
