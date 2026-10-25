---
id: TASK-005
title: Surface measured coverage and project status
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
priority: medium
type: enhancement
ordinal: 39000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Roadmap Next item 3. CI already enforces 95% statement coverage, but contributors and readers cannot readily see measured coverage and current build/release status.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Pull-request CI summaries publish the measured total statement coverage while retaining the existing 95% gate.
- [ ] #2 README badges display coverage, CI, and latest release, including prereleases during alpha, and preserve the existing license badge.
- [ ] #3 Badge values are derived from maintained automated sources rather than manually updated percentages or versions.
- [ ] #4 Validate the reporting/badge paths, including an alpha prerelease, and document any required publishing configuration.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Roadmap is mapped and current
- [ ] #2 CI passes, test coverage is 95%+
<!-- DOD:END -->
