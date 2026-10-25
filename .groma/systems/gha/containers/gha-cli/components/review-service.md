---
type: C4 Component
title: Pull-request and release discovery service
status: stable
groma:
  id: review-service
  parent: gha-cli
  code:
    - scanner: go
      file: internal/review/service.go
  group: Pull requests and reviews
  technology: Go; context; goroutines
description: Combines code-host facts into review decisions, PR preflights, and release discovery.
---

Coordinates CodeHostProvider for filtered PR selection and authenticated identity, parallel PR/review/thread inspection followed by head check runs, and derived readiness/risk/action summaries. Missing optional CI or thread information is unavailable. The same service currently owns PR creation preflight and write validation, bounded published-release listings, and release notes from merged PRs; those responsibilities have not been split into separate services.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/review/service.go](../../../../../../internal/review/service.go) | [internal/github/client.go](../../../../../../internal/github/client.go) | Retrieves PR signals | CodeHostProvider interface dispatch |
| [internal/review/service.go](../../../../../../internal/review/service.go) | [internal/interfaces/code_host_provider.go](../../../../../../internal/interfaces/code_host_provider.go) | Uses code-host contract | Go interface |
