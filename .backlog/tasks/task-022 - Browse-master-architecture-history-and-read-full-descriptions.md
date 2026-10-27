---
id: TASK-022
title: Browse master architecture history and read full descriptions
status: Done
assignee:
  - '@codex'
created_date: '2026-10-25T14:53:00Z'
updated_date: '2026-10-27T19:30:00Z'
labels: []
dependencies: []
references:
  - .github/workflows/architecture.yml
  - .groma/project.md
  - docs/development.md
modified_files:
  - tools/architecture/test_publish.py
  - tools/architecture/publish.py
  - .github/workflows/architecture.yml
  - docs/development.md
  - .groma/project.md
type: enhancement
ordinal: 10500
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Restore the native Groma site after the user rejected the custom history and description wrapper. Keep native static-history limitations and shorten the project overview so its map label fits.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The native Groma viewer and single-snapshot export are restored, with the custom history wrapper removed.
- [x] #2 The project label is concise and fits visibly inside the native Groma map title plate.
- [x] #3 The corrected site deploys successfully; unrelated branch-command work and the actor model remain unchanged.
<!-- AC:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Reverted the custom history/description wrapper at the user request. Restored the native Groma export workflow and publication documentation, deleted the history builder and tests, and rewrote the project overview through Groma to fit its original map label. Native export, actionlint, local label inspection, live native viewer, Pages deployment, and Go CI pass. Full native commit history is not populated by this static export. Actor model and unrelated TASK-003 work remain unchanged.
<!-- SECTION:FINAL_SUMMARY:END -->
