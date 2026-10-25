---
type: C4 Component
title: Repository target and PR draft resolution
status: stable
groma:
  id: repository-context
  parent: gha-cli
  code:
    - scanner: go
      file: internal/git/repository.go
    - scanner: go
      file: internal/git/current_branch.go
      symbol: CurrentBranch
  group: Branches and checkouts
  technology: Git subprocesses; GitHub remote URL parsing
description: Resolves provider targets and bounded local PR drafting inputs.
---

RepositoryResolver prefers an explicit --repo, then an explicit --path origin, the configured repository, and the current checkout origin. Origin-write validation checks the actual push target against any selected provider repository so policy checks cannot be redirected. Shared helpers read the current branch and committed base-to-head subjects and file names for PR drafts. Resolution never clones or fetches.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [repository-context](repository-context.md) | [git-and-repository-storage](../../../../../externals/git-and-repository-storage.md) | Resolves checkout context | Git subprocesses |
