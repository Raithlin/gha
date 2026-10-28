---
id: TASK-008
title: Evaluate whether coordinated branch cleanup adds workflow value
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
priority: low
type: spike
ordinal: 61000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Roadmap Next item 6. Cleanup already provides bounded read-only candidates and branch delete owns guarded single-branch writes. A combined mutation workflow is justified only if coordination adds meaningful safety and recovery value.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Compare the existing candidate-review plus branch-delete workflow with coordinated cleanup and record a Command Value Test decision.
- [ ] #2 The existing cleanup listing stays read-only and bounded, explains candidates, and excludes the base, current branch, and branches checked out in other worktrees.
- [ ] #3 Any accepted mutation proposal specifies complete per-branch preview, provider and worktree safety, explicit scope, dry-run behavior, and partial-failure recovery.
- [ ] #4 If coordination does not add sufficient value, record the decision to retain the existing workflows; if it does, identify focused delivery tasks rather than adding a thin wrapper.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Roadmap is mapped and current
- [ ] #2 CI passes, test coverage is 95%+
<!-- DOD:END -->
