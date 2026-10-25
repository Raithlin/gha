---
type: C4 Component
title: Code-host provider contract
status: stable
groma:
  id: code-host-provider
  parent: gha-cli
  code:
    - scanner: go
      file: internal/interfaces/code_host_provider.go
  group: Provider integration
  technology: Go interfaces
description: Defines the provider operations used by review workflows.
---

CodeHostProvider abstracts repository, identity, pull-request, review, issue, check-run, release-list, and review-thread access using domain models and context cancellation. GitHubClient is the concrete implementation today. Branch safety and release publication use separate narrower interfaces declared alongside their services; this contract is not the sole adapter boundary for every workflow.
