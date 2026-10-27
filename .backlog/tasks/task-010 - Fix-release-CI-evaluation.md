---
id: TASK-010
title: Fix release CI evaluation
status: Done
assignee:
  - '@codex'
created_date: '2026-10-25T13:54:00Z'
updated_date: '2026-10-27T19:13:00Z'
labels:
  - roadmap
  - next
dependencies: []
references:
  - docs/roadmap.md
  - DESIGN.md
  - internal/release/service.go
  - internal/github/release.go
  - internal/output/review.go
  - github-provider
  - release-service
  - result-rendering
modified_files:
  - internal/release/ci_test.go
  - internal/release/service.go
  - pkg/model/model.go
  - internal/release/ci.go
  - internal/release/service_test.go
  - internal/github/release_ci_test.go
  - internal/github/release_ci.go
  - internal/github/release.go
  - internal/commands/release_publish_test.go
  - internal/github/release_test.go
  - internal/output/release_test.go
  - internal/output/review.go
  - internal/output/release_ci.go
  - docs/command-guide.md
  - docs/roadmap.md
  - .groma/systems/gha/containers/gha-cli/components/github-provider.md
  - .groma/systems/gha/containers/gha-cli/components/release-ci.md
  - .groma/systems/gha/containers/gha-cli/components/release-service.md
  - .groma/systems/gha/containers/gha-cli/components/ci.md
  - .groma/systems/gha/containers/gha-cli/components/result-rendering.md
  - .groma/systems/gha/containers/gha-cli/components/output-release-ci.md
priority: high
type: bug
ordinal: 38000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Roadmap Next item 8. Release readiness incorrectly requires every attached check to conclude success. A skipped native job alongside passing package validation falsely blocked Plasma Shortcut Explorer v0.1.0. Evidence: https://github.com/Raithlin/plasma-shortcut-explorer/actions/runs/39160903347
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Completed skipped and neutral checks are non-failing, while readiness still requires successful validation; an entirely skipped suite cannot pass.
- [x] #2 Failures, pending checks, cancellations, and unavailable results remain blocking, and every blocker includes check name, status, and conclusion in structured and readable output.
- [x] #3 A regression covers an intentionally skipped native job from ordinary CI together with passing Arch, Debian, and Fedora jobs from a full package run on the same commit.
- [x] #4 Evaluate configured required checks and relevant workflow runs/attempts for the exact release commit; document the readiness policy and keep unavailable policy signals explicit.
- [x] #5 Tests cover mixed successful/skipped/neutral checks, all-skipped suites, each blocking state, and exact-commit/run relevance; update the public contract documentation.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Roadmap is mapped and current
- [x] #2 CI passes, test coverage is 95%+
<!-- DOD:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Implemented release readiness for skipped/neutral checks alongside successful validation, with named structured blockers, exact-commit/latest-attempt workflow evidence, required-check policy evaluation, and explicit unavailable signals. Fixed the reviewed ruleset-only protection regression and verified the live GHA master policy.

All acceptance criteria and Definition of Done items are complete. Local make check passes with 95.2% coverage. Hosted Build and Test also passed on PR #26 at commit 393642b43716958ac1d6ab088dbdbe38fe876863 (run 40504380755, completed 2026-10-27 14:31:59 SAST). Reviewer approval remains pending; the PR has not been merged.
<!-- SECTION:FINAL_SUMMARY:END -->
