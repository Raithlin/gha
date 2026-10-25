---
id: TASK-022
title: Browse master architecture history and read full descriptions
status: To Do
assignee: []
created_date: '2026-10-25T14:53:00Z'
labels: []
dependencies: []
type: enhancement
ordinal: 22000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The published Groma site exports only one commit, so readers cannot browse master history. Its map title plate also clips the project overview. Provide a history selector with per-commit Groma snapshots and an expandable complete project description.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The published site lists recent compatible first-parent master commits and defaults to the newest; selecting a commit loads its exact map and source.
- [ ] #2 The complete project description can be expanded and read without clipping, including at narrow widths.
- [ ] #3 Regression tests cover history selection, unsupported commits, export failures, and safe rendering; the deployed site is verified in a browser.
<!-- AC:END -->
