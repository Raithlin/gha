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
  group: Release publication
  technology: Go; Cobra
description: Lists releases, generates notes, and coordinates guarded release-tag publication.
---

releases and release create-notes use the review service for published-release discovery and merged-PR notes. release publish resolves the selected checkout/repository/workflow and delegates preflight and publication to release.Service. It returns blockers and planned effects in a dry run; successful tag push is reported separately from observed Actions activity.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/commands/release.go](../../../../../../internal/commands/release.go) | [internal/review/service.go](../../../../../../internal/review/service.go) | Generates release notes | Go service call |
| [internal/commands/releases.go](../../../../../../internal/commands/releases.go) | [internal/review/service.go](../../../../../../internal/review/service.go) | Lists published releases | Go service call |
| [internal/commands/release_publish.go](../../../../../../internal/commands/release_publish.go) | [internal/git/repository.go](../../../../../../internal/git/repository.go) | Resolves release target | Go call |
| [internal/commands/release_publish.go](../../../../../../internal/commands/release_publish.go) | [internal/release/service.go](../../../../../../internal/release/service.go) | Coordinates release publication | Go service calls |
| [internal/commands/release.go](../../../../../../internal/commands/release.go) | [internal/output/review.go](../../../../../../internal/output/review.go) | Renders workflow result | Go output functions |
| [internal/commands/release_publish.go](../../../../../../internal/commands/release_publish.go) | [internal/output/review.go](../../../../../../internal/output/review.go) | Renders workflow result | Go output functions |
