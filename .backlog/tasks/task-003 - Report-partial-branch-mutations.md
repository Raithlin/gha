---
id: TASK-003
title: Report partial branch mutations
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

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Implemented additive BranchMutation failure reporting for create, rename and delete: execution failures retain completed effects on stdout, mark failed effects, preserve unattempted planned steps, and return nonzero with actionable diagnostics. Added checkout_state without removing existing v1 fields. Origin rename now requires atomic push support to prevent partially updated remote refs. Regression tests cover JSON/YAML/text partial outcomes, first-effect and checkout failures, unsupported atomic push and output-write failures. Updated command help, capabilities, command guide and roadmap. CI-equivalent checks pass with 95.1% total statement coverage; rebuilt-binary smoke tests pass. Servers lacking atomic push support reject remote rename after the local rename; the result reports that recovery state.
<!-- SECTION:FINAL_SUMMARY:END -->
