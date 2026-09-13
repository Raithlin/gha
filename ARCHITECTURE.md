# ARCHITECTURE.md

# GHA Architecture

This document describes the **current** architecture of GHA.

Unlike `DESIGN.md`, which describes the long-term vision, this document should always reflect the implementation that exists today.

---

# Current Status

Project phase:

> Phase 1 complete; Phase 2 review workflows in progress

Current capabilities:

* CLI executable and Cobra command tree
* Startup configuration from environment variables
* GitHub REST client behind a provider interface
* Repository resolution from flags, configuration, or the local Git remote
* Pull request listings and single-PR review summaries with text, JSON, and YAML rendering

---

# Architectural Principles

The application follows a few simple rules.

## Keep `main()` Small

The executable should do almost nothing beyond bootstrapping the application.

Responsibilities:

* initialise configuration
* initialise logging
* create the application
* execute the requested command

Business logic does **not** belong in `main()`.

---

## Package Responsibilities

The codebase is organised around responsibilities rather than technical layers.

```text
cmd/
    gha/

internal/
    commands/
    config/
    git/
    github/
    interfaces/
    output/
    review/

pkg/
    model/
```

Each package should have one reason to change.

---

## Dependency Direction

Dependencies should always point inward.

```text
CLI
 │
 ▼
Commands
 │
 ▼
Services
 │
 ▼
Providers
```

Providers should never call commands.

Commands should not know implementation details of providers.

---

# Application Flow

A typical execution should look like:

```text
main()

    │

    ▼

Load configuration

    │

    ▼

Construct application

    │

    ▼

Resolve command

    │

    ▼

Execute command

    │

    ▼

Render output
```

Each stage should be independently testable.

---

# Commands

Commands are the entry point for user interaction.

Responsibilities:

* validate arguments
* call domain services
* render results

Commands validate arguments, select a workflow, and render results. Review
selection and filtering live in `internal/review`.

---

# Services

Services implement application behaviour.

Current service:

* `review.Service`, which coordinates PR listings, review summaries, and CI status

Services coordinate work.

They should remain independent of presentation concerns.

---

# Providers

Providers communicate with external systems.

Examples:

* GitHub
* Git
* Local filesystem

Providers should:

* make API calls
* parse responses
* convert external models into internal models

Providers should not contain application logic.

---

# Models

Shared models belong in `pkg/model`.

Examples:

```text
Repository
PullRequest
Review
Release
Branch
```

These models represent concepts understood by the application rather than external APIs.

---

# Configuration

Configuration is loaded once during startup. Currently, `internal/config` reads
environment variables. Command flags are applied by their commands and override
the corresponding configured value. A configuration file is a future extension.

Consumers receive configuration through dependency injection.

## Release Command Contract

Release listing, viewing, and note generation are separate workflows. The
planned CLI surface is:

```text
gha releases                         list published GitHub releases
gha release show <tag>               inspect one published release
gha release create-notes --since ... generate notes from merged pull requests
```

This prevents the read-only note generator from being mistaken for either a
GitHub Release lookup or a mutating release-creation operation. Until the
command tree is migrated, `gha release --since ...` remains the implemented,
temporary spelling for generating notes.

Packages should not read environment variables directly.

---

# Output

Rendering should be isolated from business logic.

Current renderers:

* text
* JSON
* YAML

Commands return structured data where practical.

`gha review <number>` returns the versioned `model.ReviewSummary` schema for
structured formats. It contains the pull request, review decisions, readiness
signals, risk signals, and recommended next actions. CI status is derived from
GitHub check runs for the PR head commit, and review-thread resolution is read
from GitHub's GraphQL API. Signals that cannot be retrieved are explicitly
marked as unavailable rather than inferred.

Renderers decide how data is displayed.

---

# Error Handling

Errors should be returned.

Wrap errors with context.

Example:

```go
return fmt.Errorf("loading repositories: %w", err)
```

Avoid panic except for unrecoverable programmer errors.

---

# Context Propagation

Operations that may block should accept a `context.Context`.

Cancellation and deadlines should flow naturally through the call stack.

---

# Concurrency

Concurrency should be introduced only when it improves responsiveness.

Likely candidates:

* loading multiple repositories
* fetching review data
* dashboard refresh

Preferred tools:

* goroutines
* channels
* `errgroup`

Avoid unnecessary synchronisation.

---

# Dependency Injection

Dependencies should be provided explicitly.

Prefer constructor injection.

Example:

```go
service := review.NewService(repoProvider)
```

Avoid global state.

---

# Testing Strategy

Focus on testing behaviour rather than implementation.

Priorities:

1. domain services
2. command behaviour
3. provider integrations

Use table-driven tests where appropriate.

---

# Directory Structure

Current layout:

```text
cmd/
    gha/
        main.go

internal/
    commands/
    config/
    git/
    github/
    interfaces/
    output/
    review/

pkg/
    model/
```

`cmd/gha` loads configuration and constructs the GitHub provider, review service,
repository resolver, and command tree explicitly. Additional packages should be
introduced only when a clear responsibility emerges.

---

# Future Evolution

The following areas are expected to grow:

* provider abstraction
* plugin architecture
* caching
* metrics
* TUI
* multiple source providers

They should not be implemented until required.

---

# Design Philosophy

The architecture should optimise for:

* clarity
* maintainability
* testability
* explicit dependencies
* idiomatic Go

When faced with a design choice, prefer the simplest solution that preserves these goals.

As the project evolves, this document should be updated to describe the architecture that **exists**, not the architecture that is merely planned.
