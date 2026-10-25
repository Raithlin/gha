---
type: C4 Component
title: Offline repository analysis
status: stable
groma:
  id: repository-analysis
  parent: gha-cli
  code:
    - scanner: go
      file: internal/git/analyzer.go
  group: Branches and checkouts
  technology: Git subprocesses
description: Combines checkout, history, storage, and largest-file facts locally.
---

Runs Git inspection for repository roots, branch/HEAD, dirty files, recent commits, object/storage statistics, and largest tracked files. Per-section limits and truncation make the snapshot bounded. It does not contact origin or GitHub.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [repository-analysis](repository-analysis.md) | [git-and-repository-storage](../../../../../externals/git-and-repository-storage.md) | Reads repository facts | Git subprocesses |
