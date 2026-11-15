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
    - scanner: go
      file: internal/github/release_status.go
    - scanner: go
      file: internal/github/release_assets.go
  technology: Go net/http; HTTPS REST and GraphQL
  group: Provider integration
description: Maps GitHub APIs to workflow facts and writes.
---

GitHubClient implements CodeHostProvider plus branch SafetyProvider and release Provider. It makes authenticated REST requests for identity, repositories, PRs, reviews, checks, releases, branch policy and permissions, and uses GraphQL for review-thread resolution. Release publication reads CI, tag, and permission facts. Release status resolves remote tags and exact workflow attempts. AssetProvider reads bounded hosted-release assets and streams authenticated asset bytes. HTTP requests carry context and use a 30-second client timeout. Git authentication is separate.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [github-provider](github-provider.md) | [github](../../../../../externals/github.md) | Queries workflow facts | HTTPS REST and GraphQL |
| [internal/github/client.go](../../../../../../internal/github/client.go) | [internal/interfaces/code_host_provider.go](../../../../../../internal/interfaces/code_host_provider.go) | Implements code-host operations | Go interface implementation |
