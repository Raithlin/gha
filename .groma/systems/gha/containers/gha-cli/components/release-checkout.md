---
type: C4 Component
title: Release checkout adapter
status: stable
groma:
  id: release-checkout
  parent: gha-cli
  code:
    - scanner: go
      file: internal/release/checkout.go
  group: Release publication
  technology: Git subprocesses; yaml.v3
description: Reads committed release inputs and performs the planned tag effects.
---

Implements the release Checkout boundary using Git and YAML. It checks HEAD, cleanliness, the live origin branch tip, local tag existence, committed workflow tag patterns, and any workflow-required reviewed release notes. CreateTag and PushTag reuse the shared TagWriter; this adapter never creates a GitHub Release itself.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [release-checkout](release-checkout.md) | [git-and-repository-storage](../../../../../externals/git-and-repository-storage.md) | Checks release inputs | Git subprocesses; ls-remote |
| [internal/release/checkout.go](../../../../../../internal/release/checkout.go) | [internal/git/repository.go](../../../../../../internal/git/repository.go) | Parses origin identity | Go call |
| [internal/release/checkout.go](../../../../../../internal/release/checkout.go) | [internal/git/tag.go](../../../../../../internal/git/tag.go) | Creates and pushes tag | Go adapter calls |
