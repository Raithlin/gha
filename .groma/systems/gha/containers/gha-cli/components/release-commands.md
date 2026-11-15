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
    - scanner: go
      file: internal/commands/release_assets.go
  group: Release publication
  technology: Go; Cobra
description: Lists releases, generates notes, publishes guarded tags, follows publication, and verifies hosted assets.
---

releases and release create-notes use the review service for published-release discovery and merged-PR notes. release publish resolves the checkout, repository, and workflow, then delegates preflight and publication to release.Service. release status observes a matching workflow attempt and hosted release without changing state. release assets lists bounded provider metadata. release verify requires an explicit output directory; it previews destinations with --dry-run, then delegates download and digest checks to AssetService and renders per-asset evidence.

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
| [internal/commands/release_assets.go](../../../../../../internal/commands/release_assets.go) | [internal/release/assets.go](../../../../../../internal/release/assets.go) | Requests asset verification | Go service call |
| [internal/commands/release_assets.go](../../../../../../internal/commands/release_assets.go) | [internal/output/release_assets.go](../../../../../../internal/output/release_assets.go) | Renders asset evidence | Go output functions |
