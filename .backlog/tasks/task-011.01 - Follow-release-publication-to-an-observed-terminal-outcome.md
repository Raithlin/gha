---
id: TASK-011.01
title: Follow release publication to an observed terminal outcome
status: Done
assignee:
  - '@codex'
created_date: '2026-10-27T19:07:00Z'
updated_date: '2026-11-13T07:57:00Z'
labels: []
dependencies: []
references:
  - docs/roadmap.md
  - https://github.com/Raithlin/plasma-shortcut-explorer/releases/tag/v0.1.4
  - internal/release/service.go
  - internal/github/release.go
  - internal/commands/release.go
  - github-provider
  - release-commands
  - result-rendering
  - release-status-service
  - https://github.com/Raithlin/gha/pull/41
documentation:
  - docs/roadmap.md
modified_files:
  - internal/release/status_test.go
  - internal/release/status.go
  - internal/github/release_status_test.go
  - internal/github/release_status.go
  - internal/commands/release_status_test.go
  - internal/commands/release_status.go
  - internal/output/release_status.go
  - internal/commands/release.go
  - internal/commands/capabilities_test.go
  - internal/commands/capabilities.go
  - docs/command-guide.md
  - skills/gha/SKILL.md
  - docs/roadmap.md
  - docs/architecture.md
  - docs/design.md
  - README.md
  - internal/output/release_status_test.go
  - .groma/systems/gha/containers/gha-cli/components/github-provider.md
  - .groma/systems/gha/containers/gha-cli/components/release-status.md
  - .groma/systems/gha/containers/gha-cli/components/release-commands.md
  - .groma/systems/gha/containers/gha-cli/components/commands-release-status.md
  - .groma/systems/gha/containers/gha-cli/components/result-rendering.md
  - .groma/systems/gha/containers/gha-cli/components/output-release-status.md
  - .groma/systems/gha/containers/gha-cli/components/release-status-service.md
  - .groma/systems/gha/containers/gha-cli/components/status.md
parent_task_id: TASK-011
priority: medium
type: feature
ordinal: 57000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Both v0.1.3 and v0.1.4 tag publications initially returned unavailable workflow observations. The first workflow later failed and the second published successfully. Re-running tag mutation is not a safe status mechanism.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A bounded read-only status and optional wait workflow correlates repository, tag, exact commit, workflow, run, and attempt and can be used independently after publication.
- [x] #2 Results distinguish undiscovered or unavailable evidence, pending execution, failed or cancelled execution, successful execution, and hosted release availability; tag push alone never proves publication.
- [x] #3 Wait supports timeout and cancellation with structured partial observations, and inspection never creates tags or re-runs publication.
- [x] #4 JSON and terminal output, capability discovery, documentation, and regressions cover eventual consistency, mismatched identities, failure, timeout, and successful publication.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Roadmap is mapped and all documentation is current
- [x] #2 CI passes, test coverage is 94.51%+
- [x] #3 No `--confirm` arguments - the default call executes, `--dry-run` is the alternative
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Add a read-only `gha release status <tag>` command with optional `--wait` and bounded `--timeout`. Resolve --repo/--path through the existing target resolver, inspect the remote tag and its exact commit, then select a matching tag-triggered workflow run and latest attempt from bounded GitHub reads; inspect hosted release availability separately. Represent missing, pending, failed/cancelled, successful, and published states explicitly in JSON and text. Test provider identity filtering, eventual consistency, API uncertainty, timeout/cancellation, and command rendering before implementation. Update capability inventory, skill/help, and release documentation; run project checks and architecture review. Publish a PR with ci:packages and use a later push plus unrelated label event to verify TASK-032.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implementation is on PR #41. Local env -u CODEX_HOME make check passes at 95.0% coverage; Python packaging tests pass; Groma 0.7.7 architecture refresh/check are current. Fresh live ./bin/gha release status v0.4.0-beta.1 --repo Raithlin/gha --workflow .github/workflows/release.yml reports published with exact tag commit f1a9aa0d, successful workflow run 40762145144 attempt 1, and an available hosted release; JSON and text render consistently. Capability inventory exposes the read-only command. Hosted PR Go CI at 61e5c17 passed; final head CI pending after macOS smoke fixture repair.

Hosted PR #41 head 7572f5b passed Build and Test, CodeQL, package build, all three Linux installs, and both macOS installs (package run 43389114503). Regression tests cover exact tag/commit and run attempt identity, absent/unavailable evidence, terminal outcomes, timeout/cancellation partial JSON, and text/JSON rendering; prior live timeout and interrupt checks returned structured partial observations. The command is read-only and has no confirmation or write path.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Implemented read-only `gha release status <tag>` with exact tag/commit/workflow/run/attempt correlation, separate hosted release state, and bounded wait with structured partial results on timeout or cancellation. Updated JSON/text output, capability inventory, help, docs, skill guidance, tests, and architecture records. Local `make check` passed at 95.0% coverage; live published-release checks passed; PR #41 head 7572f5b passed Go CI, CodeQL, and the five-environment package matrix.
<!-- SECTION:FINAL_SUMMARY:END -->
