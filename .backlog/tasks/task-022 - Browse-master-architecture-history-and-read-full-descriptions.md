---
id: TASK-022
title: Browse master architecture history and read full descriptions
status: In Progress
assignee:
  - '@codex'
created_date: '2026-10-25T14:53:00Z'
updated_date: '2026-10-25T15:11:00Z'
labels: []
dependencies: []
references:
  - .github/workflows/architecture.yml
  - docs/development.md
  - .groma/project.md
modified_files:
  - tools/architecture/test_publish.py
  - tools/architecture/publish.py
  - .github/workflows/architecture.yml
  - docs/development.md
  - .groma/project.md
type: enhancement
ordinal: 22000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Restore the native Groma site after the user rejected the custom history and description wrapper. Keep native static-history limitations and shorten the project overview so its map label fits.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The native Groma viewer and single-snapshot export are restored, with the custom history wrapper removed.
- [ ] #2 The project label is concise and fits visibly inside the native Groma map title plate.
- [ ] #3 The corrected site deploys successfully; unrelated branch-command work and the actor model remain unchanged.
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Revert only the custom history implementation and restore the native Groma export workflow and documentation.
2. Rewrite the project overview through Groma so it fits the map label, then verify the native rendering locally.
3. Publish the correction, verify Pages and CI, and record the native-history limitation without adding custom controls.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
RED: the new integration suite failed because the history publisher did not exist. GREEN: ten tests now pass, covering first-parent history, pre-Groma exclusion, limits, safe metadata, export failures, existing-output protection, and incompatible envelopes. Pinned Groma exports five actual local revisions successfully. Browser checks confirmed changing to an older SHA loads its matching iframe, and the expanded complete overview wraps and scrolls to its final paragraph at 390px width. Generated JavaScript passes node --check; workflow passes actionlint; git diff --check passes. Actor model intentionally unchanged: coding agents use the same commands as developers and harnesses already appear as an external system.

Deployed fix commit 9c7af3d62cbb29990c0e262d5fe974a7cfe52997. Pages run 40204975835 and Go CI 40204975805 both passed. Live site lists 19 compatible snapshots in its 30-commit window. Browser selection of the earliest e8532f7 entry loaded revisions/e8532f7... with the exact historical commit metadata, and the full overview expands above the map. Local 390px-width preview verified wrapping and scrolling to the final sentence. Latest-default behavior, unavailable-revision fallback, and per-revision source exports verified. No architecture records or actors changed.

User correction: native Groma history was intended. Remove the custom history selector and external description panel; retain native viewer limitations and rewrite the map label to fit. Actor model remains unchanged.

Rollback applied in an isolated checkout of origin/master to avoid including ongoing TASK-003 commits and working-tree edits. Backlog saved criterion/file updates but its automatic commit initially refused while Git revert bookkeeping was active; git revert --quit preserved the rollback and cleared that bookkeeping.

Restored native single-snapshot export and removed the custom history builder and tests. Rewrote .groma/project.md through groma edit project. Native local export succeeds; browser overhead view confirms the complete shortened overview fits the original title plate. actionlint and git diff --check pass. Full native history is not supported by this static export, as accepted by the user.
<!-- SECTION:NOTES:END -->
