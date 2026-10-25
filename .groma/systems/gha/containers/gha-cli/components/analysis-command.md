---
type: C4 Component
title: Repository analysis command
status: stable
groma:
  id: analysis-command
  parent: gha-cli
  code:
    - scanner: go
      file: internal/commands/analyze.go
  group: Branches and checkouts
  technology: Go; Cobra
description: Validates and renders an offline repository snapshot.
---

Selects a checkout and per-section limit, delegates through a small analyzer interface, and emits the RepositoryAnalysis contract. It calls the local analyzer directly rather than a provider-backed workflow service.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/commands/analyze.go](../../../../../../internal/commands/analyze.go) | [internal/git/analyzer.go](../../../../../../internal/git/analyzer.go) | Requests offline snapshot | Analyzer interface dispatch |
| [internal/commands/analyze.go](../../../../../../internal/commands/analyze.go) | [internal/output/review.go](../../../../../../internal/output/review.go) | Renders workflow result | Go output functions |
