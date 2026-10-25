---
id: TASK-010
title: Fix release CI evaluation
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
ordinal: 10000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Roadmap Next item 8. Release readiness incorrectly requires every attached check to conclude success. A skipped native job alongside passing package validation falsely blocked Plasma Shortcut Explorer v0.1.0. Evidence: https://github.com/Raithlin/plasma-shortcut-explorer/actions/runs/39160903347
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Completed skipped and neutral checks are non-failing, while readiness still requires successful validation; an entirely skipped suite cannot pass.
- [ ] #2 Failures, pending checks, cancellations, and unavailable results remain blocking, and every blocker includes check name, status, and conclusion in structured and readable output.
- [ ] #3 A regression covers an intentionally skipped native job from ordinary CI together with passing Arch, Debian, and Fedora jobs from a full package run on the same commit.
- [ ] #4 Evaluate configured required checks and relevant workflow runs/attempts for the exact release commit; document the readiness policy and keep unavailable policy signals explicit.
- [ ] #5 Tests cover mixed successful/skipped/neutral checks, all-skipped suites, each blocking state, and exact-commit/run relevance; update the public contract documentation.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Roadmap is mapped and current
- [ ] #2 CI passes, test coverage is 95%+
<!-- DOD:END -->
