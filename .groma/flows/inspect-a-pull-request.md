---
type: Groma Flow
title: Inspect a pull request
groma:
  id: inspect-a-pull-request
---

A developer asks for one PR summary. The service gathers PR, review, and thread data, then head check runs. Rendering occurs after the service result returns; this flow is a call sequence, not a claim that all calls are serial.

## Steps

| From | To | Action |
| --- | --- | --- |
| [Developer](../actors/developer.md) | [GHA CLI](../systems/gha/containers/gha-cli/container.md) | Run gha review <number> --format json |
| [cli-runtime](../systems/gha/containers/gha-cli/components/cli-runtime.md) | [pull-request-commands](../systems/gha/containers/gha-cli/components/pull-request-commands.md) | Dispatch the selected review command |
| [pull-request-commands](../systems/gha/containers/gha-cli/components/pull-request-commands.md) | [repository-context](../systems/gha/containers/gha-cli/components/repository-context.md) | Resolve --repo, --path, configured default, or origin |
| [pull-request-commands](../systems/gha/containers/gha-cli/components/pull-request-commands.md) | [review-service](../systems/gha/containers/gha-cli/components/review-service.md) | Inspect the selected PR |
| [review-service](../systems/gha/containers/gha-cli/components/review-service.md) | [github-provider](../systems/gha/containers/gha-cli/components/github-provider.md) | Retrieve PR, reviews, threads, and head checks through the injected provider |
| [github-provider](../systems/gha/containers/gha-cli/components/github-provider.md) | [github](../externals/github.md) | Read REST and GraphQL facts; optional failures remain unavailable |
| [pull-request-commands](../systems/gha/containers/gha-cli/components/pull-request-commands.md) | [result-rendering](../systems/gha/containers/gha-cli/components/result-rendering.md) | Render readiness, risks, outstanding reviewers, and recommended actions |
