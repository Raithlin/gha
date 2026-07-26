# DESIGN.md

# GHA — GitHub Assistant

## Vision

GHA is a developer productivity tool written in Go.

Its purpose is **not** to wrap the GitHub CLI.

Its purpose is to help software developers make better engineering decisions by combining information from GitHub, Git, CI systems, issue trackers, and local repositories into a single cohesive experience.

GitHub is simply the first integration.

The project should remain useful even if GitHub were replaced with GitLab, Azure DevOps or another provider.

---

# Philosophy

GHA should feel like a native Unix tool.

It should be:

* fast
* predictable
* scriptable
* composable
* offline-friendly where practical

The application should prefer:

* simple text output
* structured JSON when requested
* minimal configuration
* sensible defaults

It should avoid unnecessary complexity.

---

# Long-Term Goals

The completed application should eventually support areas such as:

## Repository Management

* List repositories
* Search repositories
* Clone repositories
* Repository health
* Repository statistics
* Language breakdown
* Activity summaries

---

## Pull Requests

* List assigned PRs
* Review queue
* Checkout PR
* Merge readiness
* Risk analysis
* Missing reviewers
* CI status
* Branch conflicts
* Review estimates

Example:

```text
gha review 351

Risk: HIGH

Files changed: 42

Touches:
 ✓ Authentication
 ✓ Billing
 ✓ API
 ✓ Database

Missing:
 ✗ Tests
 ✗ Documentation

Estimated review time:
38 minutes
```

---

## Engineering Metrics

Generate useful engineering insights.

Examples:

* Hotspots
* Frequently modified files
* Bus factor
* Churn analysis
* Large PR detection
* Long-lived branches
* Review turnaround
* Commit frequency

These should produce actionable information rather than vanity metrics.

---

## Releases

Generate:

* release notes
* changelogs
* contributor summaries
* breaking changes
* migration notes

---

## Local Repository Analysis

Analyse local repositories without requiring network access.

Examples:

* branch cleanup
* stale branches
* uncommitted work
* repository size
* largest files
* ownership
* commit history

---

## Dashboards

Eventually provide an interactive TUI.

Examples:

* assigned work
* open PRs
* failing builds
* notifications
* review queue
* personal dashboard

This is a future milestone.

The initial application should remain a traditional CLI.

---

# Design Principles

## 1. Native Go

Avoid writing Java, Ruby or C# in Go.

Prefer:

* packages
* interfaces
* composition
* small types
* explicit dependencies

Avoid deep inheritance-like structures.

---

## 2. Interfaces at Boundaries

External systems should always be abstracted.

For example:

```go
type RepositoryService interface {
    List(ctx context.Context) ([]Repository, error)
}
```

Business logic should not depend directly on GitHub APIs.

---

## 3. Providers

GitHub is a provider.

Future providers may include:

* GitLab
* Azure DevOps
* Bitbucket

Core application logic should not care which provider supplies data.

---

## 4. Commands are Independent

Each command should be self-contained.

Commands should not depend heavily upon one another.

Example:

```
internal/
    commands/
        review/
        release/
        dashboard/
        repos/
```

---

## 5. Concurrency is a Feature

Independent work should happen concurrently.

Examples:

* fetching repositories
* CI status
* PR reviews
* notifications

These should execute in parallel where practical.

Use:

* goroutines
* channels
* errgroup
* context

Do not introduce concurrency where it does not improve performance or clarity.

---

## 6. Context Everywhere

Every public operation that may block should accept a context.

Never invent custom cancellation mechanisms.

---

## 7. Errors are Values

Return useful errors.

Wrap errors with context.

Avoid panic except for unrecoverable programmer mistakes.

---

## 8. Logging

Logging is for diagnostics.

User-facing output belongs on stdout/stderr.

Do not mix the two.

---

## 9. Testing

Prefer:

* table-driven tests
* small focused tests
* deterministic tests

Avoid brittle integration tests unless they add real value.

---

# Architecture

```
cmd/
    gha/

internal/
    auth/
    cache/
    commands/
    config/
    git/
    github/
    providers/
    output/
    metrics/
    review/
    release/

pkg/
    model/
```

Only reusable public packages belong in `pkg`.

Everything else belongs in `internal`.

---

# Configuration

Configuration should be minimal.

Potential sources:

1. flags
2. environment variables
3. configuration file

Command-line flags override everything.

Environment overrides configuration.

Configuration overrides defaults.

---

# Output Modes

Support multiple formats.

Default:

```
text
```

Optional:

```
json
yaml
markdown
```

Commands should separate data generation from rendering.

---

# Performance Goals

The application should feel instantaneous.

Targets:

Simple commands:

* under 100 ms

Network commands:

* under 1 second where practical

Dashboard:

* fetch data concurrently

Avoid unnecessary allocations.

Profile before optimising.

---

# Dependencies

Prefer the standard library.

Only introduce dependencies when they provide significant value.

Every dependency should have a clear justification.

---

# Coding Style

Prefer:

* explicit code
* readable code
* small functions
* meaningful names

Avoid:

* unnecessary abstractions
* excessive generics
* reflection
* clever code

Readability is more important than brevity.

---

# Roadmap

## Phase 1

* basic CLI
* configuration
* GitHub authentication
* repository listing
* PR listing

## Phase 2

* review helper
* release generation
* branch management
* local git analysis

## Phase 3

* engineering metrics
* hotspot analysis
* risk scoring
* ownership analysis

## Phase 4

* TUI dashboard
* plugins
* multiple providers
* offline cache
* background refresh

---

# What GHA Is Not

GHA is not:

* a GitHub replacement
* an IDE
* another git implementation
* another GitHub CLI

It should answer higher-level engineering questions rather than merely exposing API endpoints.

---

# Guidance for Coding Agents

When implementing features:

1. Preserve the overall architecture.
2. Prefer extending existing abstractions over duplicating code.
3. Keep packages cohesive and focused.
4. Introduce interfaces only when there is more than one plausible implementation or a clear testing boundary.
5. Do not optimise prematurely.
6. Keep `main()` extremely small.
7. Favour composition over inheritance-like patterns.
8. Minimise third-party dependencies.
9. Every exported symbol should have a clear purpose.
10. Ask: "Does this make the tool more useful to developers?" before adding a feature.

The primary objective is to build a fast, reliable, idiomatic Go application that assists developers in understanding and managing their software engineering work rather than simply interacting with GitHub.
