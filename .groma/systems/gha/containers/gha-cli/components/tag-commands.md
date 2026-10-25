---
type: C4 Component
title: Exact tag publication command
status: stable
groma:
  id: tag-commands
  parent: gha-cli
  code:
    - scanner: go
      file: internal/commands/tag.go
  group: Release publication
  technology: Go; Cobra
description: Plans and executes one selected tag and origin ref.
---

tag publish validates the selected name and commit and checks local/origin collisions through TagWriter. Dry run returns planned effects; execution creates a lightweight tag and pushes only that tag. This direct workflow is distinct from the annotated SemVer release preflight and reports partial completion when push fails.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/commands/tag.go](../../../../../../internal/commands/tag.go) | [internal/git/tag.go](../../../../../../internal/git/tag.go) | Publishes selected tag | Go adapter calls |
| [internal/commands/tag.go](../../../../../../internal/commands/tag.go) | [internal/output/review.go](../../../../../../internal/output/review.go) | Renders workflow result | Go output functions |
