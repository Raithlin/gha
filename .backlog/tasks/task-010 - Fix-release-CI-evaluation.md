---
id: TASK-010
title: Fix release CI evaluation
status: Done
assignee:
  - '@codex'
created_date: '2026-10-25T13:54:00Z'
updated_date: '2026-10-27T12:37:00Z'
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

6. Review correction: distinguish explicit protection.enabled=false in branch metadata from an unavailable classic policy lookup; evaluate active rulesets for ruleset-only branches. Add failing regression and denied/ambiguous-policy controls, rerun full checks, and verify the live GHA master policy through the provider path.
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

Review correction completed (2026-10-26):
- Added TestReleaseCIRulesetOnlyProtection. RED reproduced the ruleset-only branch falsely making policy unavailable. GREEN uses explicit branch metadata protection.enabled to distinguish known absence of classic protection; protected alone still includes rulesets. Missing enabled metadata retains classic-policy verification.
- Regression controls confirm ruleset access denied, ambiguous metadata plus classic 404, missing enabled state plus classic 404, and classic access denied all remain unavailable.
- Live read-only verification executed the corrected releaseRequiredChecks provider code against Raithlin/gha master and returned Build and Test bound to app 15368. An ephemeral Go test overlay supplied the verification without adding live-network tests to the repository or storing credentials.
- Rebuilt local binary and checked capabilities. Plain make check from a fresh interactive shell passed lint, race-enabled tests, and 95.2% statement coverage. git diff --check passed.
- Updated public readiness documentation. No release tag or remote write performed; existing user config edit remains excluded.

Hosted CI verified (2026-10-27):
- PR #26 Build and Test completed successfully on 2026-10-27 at 14:31:59 SAST (12:31:59 UTC), for branch head 393642b43716958ac1d6ab088dbdbe38fe876863.
- Verified through gha review 26 --format json (ci_status=success) and GitHub check metadata (status=COMPLETED, conclusion=SUCCESS).
- Run: https://github.com/Raithlin/gha/actions/runs/40504380755 ; job: https://github.com/Raithlin/gha/actions/runs/40504380755/job/122189815556 .
- This hosted result supersedes the earlier hosted-CI-unverified notes. Required reviewer approval remains pending and is separate from CI/task completion.
- Only this Backlog record changes after the successful run; folding it into the existing evidence commit preserves the eight-commit PR history.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Implemented release readiness for skipped/neutral checks alongside successful validation, with named structured blockers, exact-commit/latest-attempt workflow evidence, required-check policy evaluation, and explicit unavailable signals. Fixed the reviewed ruleset-only protection regression and verified the live GHA master policy.

All acceptance criteria and Definition of Done items are complete. Local make check passes with 95.2% coverage. Hosted Build and Test also passed on PR #26 at commit 393642b43716958ac1d6ab088dbdbe38fe876863 (run 40504380755, completed 2026-10-27 14:31:59 SAST). Reviewer approval remains pending; the PR has not been merged.
<!-- SECTION:FINAL_SUMMARY:END -->
