---
type: C4 Component
title: Pull-request commands
status: stable
groma:
  id: pull-request-commands
  parent: gha-cli
  code:
    - scanner: go
      file: internal/commands/review.go
      symbol: newReviewCmd
    - scanner: go
      file: internal/commands/prs.go
    - scanner: go
      file: internal/commands/pr.go
  group: Pull requests and reviews
  technology: Go; Cobra
description: Selects and presents PR listings, review summaries, and creation plans.
---

prs performs bounded listings with filters and authenticated-user shortcuts; review inspects one PR for readiness, outstanding reviewers, risks, and recommended actions. pr prepare selects the provider target and local head, combines provider preflight with a bounded local commit/file draft, and renders a proposal. pr create performs the same preparation and creates through the service unless --dry-run is set.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [internal/commands/review.go](../../../../../../internal/commands/review.go) | [internal/git/repository.go](../../../../../../internal/git/repository.go) | Resolves PR repository | Go call |
| [internal/commands/review.go](../../../../../../internal/commands/review.go) | [internal/review/service.go](../../../../../../internal/review/service.go) | Requests review summary | Go service call |
| [internal/commands/pr.go](../../../../../../internal/commands/pr.go) | [internal/git/repository.go](../../../../../../internal/git/repository.go) | Drafts local changes | Go call; bounded Git reads |
| [internal/commands/pr.go](../../../../../../internal/commands/pr.go) | [internal/review/service.go](../../../../../../internal/review/service.go) | Prepares PR creation | Go service calls |
| [internal/commands/review.go](../../../../../../internal/commands/review.go) | [internal/output/review.go](../../../../../../internal/output/review.go) | Renders workflow result | Go output functions |
| [internal/commands/pr.go](../../../../../../internal/commands/pr.go) | [internal/output/review.go](../../../../../../internal/output/review.go) | Renders workflow result | Go output functions |
