---
id: TASK-022
title: Browse master architecture history and read full descriptions
status: In Progress
assignee:
  - '@codex'
created_date: '2026-10-25T14:53:00Z'
updated_date: '2026-10-25T14:58:00Z'
labels: []
dependencies: []
references:
  - .github/workflows/architecture.yml
  - docs/development.md
  - tools/architecture/publish.py
modified_files:
  - tools/architecture/test_publish.py
  - tools/architecture/publish.py
  - .github/workflows/architecture.yml
  - docs/development.md
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

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add failing Python integration tests for bounded first-parent history, compatible revisions, safe page metadata, and exporter failures.
2. Export each of the latest 30 compatible master commits as a separate native Groma site. Add a lightweight history selector and expandable full project text around the unmodified viewer.
3. Fetch complete history in Pages CI, run the regression suite, update contributor docs, and validate workflow syntax and the local browser flow.
4. Publish and verify selecting an older master commit, returning to latest, reading the full overview, and successful CI/deployment.
<!-- SECTION:PLAN:END -->
