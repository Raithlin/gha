---
type: C4 Component
title: Release asset inspection and verification
status: stable
groma:
  id: release-assets
  parent: gha-cli
  code:
    - scanner: go
      file: internal/release/assets.go
  group: Release publication
  technology: Go; SHA-256; local files
description: Lists bounded hosted-release assets and verifies downloaded bytes against provider and manifest digests.
---

AssetService obtains a complete or explicitly truncated inventory from AssetProvider. Verification requires an existing destination directory, refuses unsafe or occupied names, stages authenticated downloads, checks size and SHA-256 evidence, and places final files only after every check passes. Missing, unsupported, unavailable, and mismatched evidence cannot produce a verified result.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/release/assets.go](../../../../../../internal/release/assets.go) | [internal/github/release_assets.go](../../../../../../internal/github/release_assets.go) | Reads and downloads assets | Injected provider interface |
