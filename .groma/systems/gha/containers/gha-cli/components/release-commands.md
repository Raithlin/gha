---
type: C4 Component
title: Release commands
status: stable
groma:
  id: release-commands
  parent: gha-cli
  code:
    - scanner: go
      file: internal/commands/release.go
    - scanner: go
      file: internal/commands/releases.go
    - scanner: go
      file: internal/commands/release_publish.go
      symbol: newReleasePublishCmd
    - scanner: go
      file: internal/commands/release_status.go
  group: Release publication
  technology: Go; Cobra
description: Lists releases, generates notes, publishes guarded tags, and follows publication status.
---

releases and release create-notes use the review service for published-release discovery and merged-PR notes. release publish resolves the selected checkout/repository/workflow and delegates preflight and publication to release.Service. It reports blockers, planned effects, and one post-push Actions observation. release status resolves --repo or --path without changing local state, then asks StatusService for one observation or bounded waiting and renders the workflow and hosted release evidence independently.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/commands/release.go](../../../../../../internal/commands/release.go) | [internal/review/service.go](../../../../../../internal/review/service.go) | Generates release notes | Go service call |
| [internal/commands/releases.go](../../../../../../internal/commands/releases.go) | [internal/review/service.go](../../../../../../internal/review/service.go) | Lists published releases | Go service call |
| [internal/commands/release_publish.go](../../../../../../internal/commands/release_publish.go) | [internal/git/repository.go](../../../../../../internal/git/repository.go) | Resolves release target | Go call |
| [internal/commands/release_publish.go](../../../../../../internal/commands/release_publish.go) | [internal/release/service.go](../../../../../../internal/release/service.go) | Coordinates release publication | Go service calls |
| [internal/commands/release.go](../../../../../../internal/commands/release.go) | [internal/output/review.go](../../../../../../internal/output/review.go) | Renders workflow result | Go output functions |
| [internal/commands/release_publish.go](../../../../../../internal/commands/release_publish.go) | [internal/output/review.go](../../../../../../internal/output/review.go) | Renders workflow result | Go output functions |
| [internal/commands/release_status.go](../../../../../../internal/commands/release_status.go) | [internal/release/status.go](../../../../../../internal/release/status.go) | Requests release observation | Go service call |
| [internal/commands/release_status.go](../../../../../../internal/commands/release_status.go) | [internal/output/release_status.go](../../../../../../internal/output/release_status.go) | Renders release status | Go output function |
