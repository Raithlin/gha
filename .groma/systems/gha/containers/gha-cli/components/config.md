---
type: C4 Component
title: Startup configuration
status: stable
groma:
  id: config
  parent: gha-cli
  code:
    - scanner: go
      file: internal/config/config.go
  group: CLI and contracts
  technology: Environment variables; gh subprocess
description: Resolves API credentials and the default repository once at startup.
---

Reads GHA_GITHUB_TOKEN and GHA_REPOSITORY. If the explicit token is empty, it attempts gh auth token and tolerates unavailable CLI authentication. It supplies configuration to bootstrap; it does not maintain a GHA configuration file. Command target flags apply their own documented precedence.

## Relationships

| Source | Target | Description | Technology |
| --- | --- | --- | --- |
| [config](config.md) | [github-cli-authentication](../../../../../externals/github-cli-authentication.md) | Obtains API credential | gh auth token |
