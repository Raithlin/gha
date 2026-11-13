---
type: C4 Component
title: Release status coordination
status: stable
groma:
  id: release-status-service
  parent: gha-cli
  code:
    - scanner: go
      file: internal/release/status.go
  group: Release publication
description: Polls read-only release evidence until publication, failure, cancellation, or timeout.
---

StatusService requests a provider observation for an exact repository and tag. A one-shot check returns the current remote facts. Optional waiting repeats bounded reads and preserves the latest observation on timeout or caller cancellation. The provider supplies tag identity, workflow run and attempt, and hosted release state; the service never creates a tag or reruns a workflow.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/release/status.go](../../../../../../internal/release/status.go) | [internal/github/release_status.go](../../../../../../internal/github/release_status.go) | Reads release evidence | Injected Go provider interface |
