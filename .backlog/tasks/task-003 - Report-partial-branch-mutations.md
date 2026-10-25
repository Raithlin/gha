---
id: TASK-003
title: Report partial branch mutations
status: To Do
assignee: []
created_date: '2026-10-25T13:54:00Z'
labels:
  - roadmap
  - next
dependencies: []
references:
  - docs/roadmap.md
  - DESIGN.md
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
- [ ] #1 If a branch write or checkout switch succeeds before a later failure, stdout includes a parseable BranchMutation result and the command exits nonzero with actionable diagnostics on stderr.
- [ ] #2 Each attempted effect reports whether it completed or failed, preserving enough detail to identify remaining recovery work.
- [ ] #3 Regression tests cover create with publish, origin rename, and combined local/origin deletion failing after an earlier successful effect.
- [ ] #4 Text output explains the same partial outcomes; documentation and capability guidance describe the failure contract without breaking existing JSON fields.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Roadmap is mapped and current
- [ ] #2 CI passes, test coverage is 95%+
<!-- DOD:END -->
