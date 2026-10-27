---
type: C4 Component
title: GitHub adapter
status: stable
groma:
  id: github-provider
  parent: gha-cli
  code:
    - scanner: go
      file: internal/github/client.go
    - scanner: go
      file: internal/github/release.go
    - scanner: go
      file: internal/github/release_ci.go
  technology: Go net/http; HTTPS REST and GraphQL
  group: Provider integration
description: Maps GitHub APIs to workflow facts and writes.
---

GitHubClient implements CodeHostProvider plus branch SafetyProvider and release Provider. It makes authenticated REST requests for identity, repositories, PRs, reviews, checks, releases, branch policy and permissions, and uses GraphQL for review-thread resolution. Its release adapter supplies CI/tag/permission facts and observes Actions for the pushed tag and commit once. It has a 30-second HTTP client timeout and passes context through requests. Git authentication is separate.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [github-provider](github-provider.md) | [github](../../../../../externals/github.md) | Queries workflow facts | HTTPS REST and GraphQL |
| [internal/github/client.go](../../../../../../internal/github/client.go) | [internal/interfaces/code_host_provider.go](../../../../../../internal/interfaces/code_host_provider.go) | Implements code-host operations | Go interface implementation |
