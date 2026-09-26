# DESIGN.md

# GHA — GitHub Assistant

## Vision

GHA is an agent-first developer tool written in Go.

Its primary user is a coding agent operating on behalf of a developer. Its
commands must therefore provide reliable scope, bounded results, stable
structured output, and safe next actions. Developers use the same commands at
the terminal, where concise text output should make the workflow easier to
understand and act on.

Its purpose is **not** to wrap or replace `git` or `gh`.

Its purpose is to make engineering workflows easier and safer when combining
information from GitHub, Git, CI systems, issue trackers, and local repositories
produces more useful context than a raw tool invocation.

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

It should be agent-first without becoming agent-only:

* structured output is the primary public interface
* text is a first-class, readable rendering of the same result
* machine contracts are explicit about scope, freshness, truncation, and
  unavailable signals
* safe inspection is the default; mutations require deliberate intent

The application should prefer:

* versioned structured JSON for command data
* simple, terminal-readable text from the same domain model
* minimal configuration
* sensible defaults

It should avoid unnecessary complexity.

## Command Value Test

GHA should not duplicate a raw `git` or `gh` command merely to change its
spelling. A command belongs in GHA when it does at least one of the following:

* combines local, provider, or CI signals into one decision-ready result
* replaces fragile parsing with a stable, documented machine contract
* makes scope, freshness, risk, or an unavailable signal explicit
* guides a safe workflow that would otherwise require several coordinated steps

When a raw command is clearer or cheaper, GHA should say so in its help or
documentation rather than pretending to be a replacement.

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

Release discovery, note generation, and publication have distinct command
intentions. The current and proposed surfaces are:

```text
gha releases                         list published GitHub releases (available)
gha release create-notes --since ... generate notes from merged pull requests (available)
gha release publish <version>        safely publish a release tag (available)
gha tag publish <name>               create and publish one exact tag (available)
gha release show <tag>               inspect one published release (deferred)
```

The plural `releases` command is for listings. `create-notes` produces content
and does not create a GitHub Release. `publish` checks the commit, tags, CI,
and release workflow before pushing an annotated tag. When a workflow opts in,
it also checks that reviewed release notes are committed. CI and the workflow
decide whether a release is created. `show` remains deferred until it adds
value beyond `gh release view`. `gha release --since ...` is not supported; use
`gha release create-notes`.

Generated notes should include:

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

## Branch Lifecycle Management

Branch management is a developer-assistance workflow with both read and write
operations. Its common foundation is Git: GHA should understand and operate on
local branches and the configured `origin` remote regardless of whether the
remote is GitHub, GitLab, or another future provider.

Provider integrations enrich that Git view with remote safety signals such as
open pull or merge requests, branch protection, permissions, and default-branch
status. They must not define the core workflow. A signal unavailable from the
current provider is reported as `unavailable`, not inferred.

The current branch command surface is:

```text
gha branches                         inspect local and origin branches
gha branch show <name>               inspect one branch and its safety signals
gha branch create <name> [--from ...] create locally, with an explicit publish option
gha branch publish <name>            publish an existing local branch through a guarded preflight
gha branch rename <old> <new>        rename locally, with an explicit origin option
gha branch delete <name>             plan or delete an explicitly selected target
gha branches cleanup                 review bounded local cleanup candidates without deleting
```

The first five stages have been delivered in this order:

1. Inventory local and origin branches, including tracking and divergence.
2. Inspect a branch with merge, request, protection, and permission signals
   where the provider supports them.
3. Create and publish branches with explicit local and origin targets.
4. Rename and delete branches only through an explicit, reviewable plan.
5. Offer cleanup candidates using documented rules, never an unexplained
   "stale" classification.

A cleanup action is a separate future workflow. It must consume explicitly
reviewed candidates and preserve the existing confirmation, provider-safety,
and checkout-transition guardrails.

Inspection is read-only. Every mutation must state its target and support
`--dry-run`; the operation executes by default and is preview-only when that
flag is present. Explicit target selection, preflight checks, and safety
guardrails such as `--force` remain in force.
Destructive operations must protect default and protected branches unless the
user deliberately overrides a documented guardrail.

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

The current GitHub adapter implements the provider-neutral `CodeHostProvider`
capabilities used by application workflows. Future adapters translate their
native APIs into the same GHA domain model; commands and services must not
depend on GitHub client types or response payloads.

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

# Agent-First CLI

GHA is built first for coding agents and secondarily for developers using a
terminal. The command line is an automation interface, not only a presentation
layer; the terminal experience renders the same trustworthy result for people.

## Structured Data

Commands that return data should offer JSON output. Text remains the default
for interactive use, but agents should not need to scrape it.

JSON is a compatibility contract:

* model results around GHA concepts, rather than provider-specific payloads
* document representative outputs
* prefer additive schema changes
* make a deliberate compatibility decision before removing or renaming fields

Command data belongs on stdout. Diagnostics, progress, and errors belong on
stderr so structured output remains parseable.

## Predictable Automation

Commands should have stable, unsurprising control flow.

* exit code `0` means the requested operation completed, including an empty result
* validation and operational failures use non-zero exit codes and actionable errors
* listing commands provide bounded results, filters, and incremental selection
  such as `--limit` and `--since` where supported
* authenticated-user shortcuts, such as `@me`, resolve from provider identity
* command help stays accurate; `gha capabilities --format json` exposes a
  complete versioned machine-readable capability inventory

Mutating commands provide `--dry-run`; they execute by default and preview only
when the flag is present. Branch mutations identify whether they affect the
local repository, `origin`, or both. Inspection commands remain read-only by
default.

## Agent-Ready Review Results

`gha prs` is the discovery and listing interface. `gha review <number>` is the
single-PR inspection interface.

As review assistance develops, structured review results should present a stable
summary of merge and CI readiness, missing reviewers, unresolved threads, risk
signals, and recommended next actions. This lets an agent move from discovery to
inspection to a safe next step without reconstructing that context from raw
GitHub responses.

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

# Delivery status and future direction

The CLI, PR discovery and review, release discovery and publication, branch
lifecycle workflows, local analysis, and Codex and Claude Code skill setup are
delivered. The installer writes a marked GHA guidance block and bundled skill
to the selected agent's global configuration unless `--dry-run` is set; uninstall
removes the managed content while preserving unrelated instructions and files.
See `gha capabilities --format json` for the command surface in a particular
build and [the roadmap](docs/roadmap.md) for prioritized work.

Agent guidance installation records configured harnesses and exact managed
destinations in a versioned manifest under the user's GHA config directory.
Uninstall uses those recorded paths and retains destinations still shared by
another configured harness. `gha agent list` reports the recorded harnesses,
destinations, and current file presence without claiming to detect agents GHA
has not configured. Codex, Claude Code, Pi, OpenCode, GitHub Copilot, and Gemini
CLI are supported; the latter four share a globally discoverable skills path.
Cursor remains deferred until a safe global discovery path exists. Installation
detects configured supported harnesses from their setup directories, offers
`--binary-only`, and reports when none are found; detection does not depend on
harness executables. Skill discovery paths have been checked against current
harness documentation. `gha update` now
fetches the newest published release (including prereleases) and refreshes only
the skill and managed guidance for harnesses recorded by `gha agent install`.
It does not update the GHA executable. PR coverage reporting and useful README badges follow. A separately
specified branch-cleanup action remains later because it adds destructive
behavior that needs explicit candidate selection and safety rules.
General-purpose tag publication is delivered.

Longer-term repository management goals include listing, searching, and
cloning repositories. These are not commands in the current build.

## Phase 3 goals

* engineering metrics
* hotspot analysis
* risk scoring
* ownership analysis

## Phase 4 goals

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
