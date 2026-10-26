---
id: TASK-010
title: Fix release CI evaluation
status: Done
assignee:
  - '@codex'
created_date: '2026-10-25T13:54:00Z'
updated_date: '2026-10-26T20:18:00Z'
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

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add failing regressions for mixed package validation, individual CI blockers, exact commit and workflow attempt selection, and required checks.
2. Add an additive CI readiness result with explicit policy and workflow signal availability; preserve existing checks and blocker fields.
3. Inspect bounded exact-SHA checks and Actions runs, retain latest relevant runs/attempts, and evaluate classic branch protection plus active ruleset required checks without inferring unavailable policy.
4. Render the same readiness evidence in JSON and text; document policy and update roadmap.
5. Rebuild, exercise release dry-runs, and run the full lint/race/coverage gate; finalize task evidence and logical local commits.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Acceptance evidence (2026-10-26):
- RED reproduced the reported native-skip failure, generic blockers, missing structured blocker fields, old-attempt leakage, malformed inventory acceptance, and missing selected-job evidence before their fixes.
- AC1/AC3: release CI service regressions cover native skipped plus Arch/Debian/Fedora success and neutral advisory; all-skipped/all-neutral suites remain blocked.
- AC2/AC5: table-driven failures, cancellations, timeouts, action_required, stale, missing conclusions, queued/in_progress/waiting/unknown states; ci.blockers exposes name/status/conclusion. Command-level JSON/text dry-run tests verify ready/blocked results and separate stdout/stderr.
- AC4/AC5: provider tests cover exact SHA/default branch, newest workflow run/latest attempt, superseded failures, app-bound required contexts, classic protection and active rulesets, commit statuses, truncated/missing inventories, denied/unavailable APIs, invalid identities, and missing selected-job check results.
- Full make check passed: lint, race-enabled complete suite, and 95.2% statement coverage. Rebuilt local binary, checked capabilities/help, and verified documentation links and git diff --check.
- Live release dry-run could not reach readiness because this session cannot authenticate the origin SSH remote. No tag or provider write was attempted. Hosted CI has not run; local make check is the same quality gate.
- New files are linked into the existing provider, release service, and rendering components through Groma CLI. Existing user Backlog config change is preserved and excluded from commits.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Fixed release readiness for skipped/neutral checks alongside successful validation. Added named structured blockers, exact-commit and latest-attempt workflow selection, required-check policy evaluation, bounded provider inventories, and matching JSON/text output. Updated public policy documentation and delivered roadmap status. Local CI gate passes at 95.2% coverage; live origin verification is limited by SSH authentication and hosted CI is unverified.
<!-- SECTION:FINAL_SUMMARY:END -->
