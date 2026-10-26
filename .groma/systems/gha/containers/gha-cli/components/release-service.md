---
type: C4 Component
title: Release publication coordination
status: stable
groma:
  id: release-service
  parent: gha-cli
  code:
    - scanner: go
      file: internal/release/service.go
    - scanner: go
      file: internal/release/ci.go
  group: Release publication
  technology: Go; release.Plan v1
description: Plans and rechecks annotated SemVer release-tag publication.
---

Combines a Checkout and Provider to require a clean checkout on the provider default branch at the current origin tip, no tag collision, provider push permission, successful CI, and a matching committed tag workflow. Opted-in reviewed notes must be present. Publish repeats preflight, creates an annotated tag, pushes its exact ref, then records one workflow observation. Partial effects and unavailable observation remain explicit; triggered does not mean a completed GitHub Release.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/release/service.go](../../../../../../internal/release/service.go) | [internal/release/checkout.go](../../../../../../internal/release/checkout.go) | Checks and writes release | Checkout interface dispatch |
| [internal/release/service.go](../../../../../../internal/release/service.go) | [internal/github/release.go](../../../../../../internal/github/release.go) | Checks publication readiness | Provider interface dispatch |
