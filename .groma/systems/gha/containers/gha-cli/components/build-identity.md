---
type: C4 Component
title: Build identity
status: stable
groma:
  id: build-identity
  parent: gha-cli
  code:
    - scanner: go
      file: internal/buildinfo/buildinfo.go
    - scanner: go
      file: internal/commands/version.go
      symbol: newVersionCmd
  group: CLI and contracts
  technology: Go linker variables
description: Reports the executable version, commit, and build date.
---

Linker-populated build variables default to development values. The version command renders the versioned VersionInfo result, and command construction uses the same identity for Cobra version reporting. Guidance updates compare against capabilities rather than trusting a version string alone.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/commands/version.go](../../../../../../internal/commands/version.go) | [internal/output/review.go](../../../../../../internal/output/review.go) | Renders workflow result | Go output functions |
