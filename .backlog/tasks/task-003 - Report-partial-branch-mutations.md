---
id: TASK-003
title: Report partial branch mutations
status: Done
assignee:
  - '@codex'
created_date: '2026-10-25T13:54:00Z'
updated_date: '2026-10-25T15:12:00Z'
labels:
  - roadmap
  - next
dependencies: []
references:
  - docs/roadmap.md
  - DESIGN.md
  - internal/commands/branch.go
  - internal/git/branch_writer.go
  - pkg/model/model.go
modified_files:
  - internal/commands/branch_partial_test.go
  - pkg/model/model.go
  - internal/commands/branch.go
  - internal/git/branch_writer.go
  - internal/output/review.go
  - internal/commands/capabilities_test.go
  - internal/commands/capabilities.go
  - docs/command-guide.md
  - docs/roadmap.md
priority: high
type: bug
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Roadmap Next item 1. A later failure in a multi-step branch operation can hide successful earlier effects, leaving agents unable to identify the actual recovery state.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 If a branch write or checkout switch succeeds before a later failure, stdout includes a parseable BranchMutation result and the command exits nonzero with actionable diagnostics on stderr.
- [x] #2 Each attempted effect reports whether it completed or failed, preserving enough detail to identify remaining recovery work.
- [x] #3 Regression tests cover create with publish, origin rename, and combined local/origin deletion failing after an earlier successful effect.
- [x] #4 Text output explains the same partial outcomes; documentation and capability guidance describe the failure contract without breaking existing JSON fields.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Roadmap is mapped and current
- [x] #2 CI passes, test coverage is 95%+
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add regression tests for partial create, rename, delete, and checkout outcomes in JSON and text.
2. Preserve completed effects and report failed effects additively in BranchMutation with actionable diagnostics.
3. Update guidance and roadmap; run CI-equivalent checks and coverage and finalize.

Origin rename will use an atomic push so its two ref updates cannot produce an unreported partial remote rename.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
RED reproduced missing stdout on partial failures and a partially applied origin rename. GREEN: JSON, YAML and text regressions now pass with completed local/checkout effects and failed origin status; atomic rename preserves both origin refs on rejection.

Rebuilt-binary smoke tests passed for create/publish, origin rename, and combined delete in JSON and text: safe dry runs, exit 1, parseable stdout, diagnostics only on stderr. Added coverage for servers without atomic push support and for stdout write failures preserving the original cause.

Final validation: CI-equivalent make check passed with golangci-lint v2.13.2, all race-enabled tests, dependency tidy verification, and total coverage 95.1%. Rebuilt bin/gha; end-to-end local bare-repository tests confirmed dry-run safety, exit code 1, stdout result and stderr diagnostics for all three workflows in JSON and text. git diff --check passed. No hosted CI run was triggered because implementation was not pushed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Implemented additive BranchMutation failure reporting for create, rename and delete: execution failures retain completed effects on stdout, mark failed effects, preserve unattempted planned steps, and return nonzero with actionable diagnostics. Added checkout_state without removing existing v1 fields. Origin rename now requires atomic push support to prevent partially updated remote refs. Regression tests cover JSON/YAML/text partial outcomes, first-effect and checkout failures, unsupported atomic push and output-write failures. Updated command help, capabilities, command guide and roadmap. CI-equivalent checks pass with 95.1% total statement coverage; rebuilt-binary smoke tests pass. Servers lacking atomic push support reject remote rename after the local rename; the result reports that recovery state.
<!-- SECTION:FINAL_SUMMARY:END -->
