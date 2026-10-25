---
id: TASK-022
title: Browse master architecture history and read full descriptions
status: Done
assignee:
  - '@codex'
created_date: '2026-10-25T14:53:00Z'
updated_date: '2026-10-25T15:04:00Z'
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
ordinal: 10500
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The published Groma site exports only one commit, so readers cannot browse master history. Its map title plate also clips the project overview. Provide a history selector with per-commit Groma snapshots and an expandable complete project description.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The published site lists recent compatible first-parent master commits and defaults to the newest; selecting a commit loads its exact map and source.
- [x] #2 The complete project description can be expanded and read without clipping, including at narrow widths.
- [x] #3 Regression tests cover history selection, unsupported commits, export failures, and safe rendering; the deployed site is verified in a browser.
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add failing Python integration tests for bounded first-parent history, compatible revisions, safe page metadata, and exporter failures.
2. Export each of the latest 30 compatible master commits as a separate native Groma site. Add a lightweight history selector and expandable full project text around the unmodified viewer.
3. Fetch complete history in Pages CI, run the regression suite, update contributor docs, and validate workflow syntax and the local browser flow.
4. Publish and verify selecting an older master commit, returning to latest, reading the full overview, and successful CI/deployment.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
RED: the new integration suite failed because the history publisher did not exist. GREEN: ten tests now pass, covering first-parent history, pre-Groma exclusion, limits, safe metadata, export failures, existing-output protection, and incompatible envelopes. Pinned Groma exports five actual local revisions successfully. Browser checks confirmed changing to an older SHA loads its matching iframe, and the expanded complete overview wraps and scrolls to its final paragraph at 390px width. Generated JavaScript passes node --check; workflow passes actionlint; git diff --check passes. Actor model intentionally unchanged: coding agents use the same commands as developers and harnesses already appear as an external system.

Deployed fix commit 9c7af3d62cbb29990c0e262d5fe974a7cfe52997. Pages run 40204975835 and Go CI 40204975805 both passed. Live site lists 19 compatible snapshots in its 30-commit window. Browser selection of the earliest e8532f7 entry loaded revisions/e8532f7... with the exact historical commit metadata, and the full overview expands above the map. Local 390px-width preview verified wrapping and scrolling to the final sentence. Latest-default behavior, unavailable-revision fallback, and per-revision source exports verified. No architecture records or actors changed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added master history navigation around unmodified Groma exports, with up to 30 recent first-parent commits, exact per-commit maps/source, latest default, historical links, and explicit fallback outside the published window. Added an expandable full description that wraps and scrolls on narrow displays. Ten regression tests, generated-JavaScript syntax validation, actionlint, Go CI, Pages deployment, and browser checks pass. Actor model remains unchanged as requested.
<!-- SECTION:FINAL_SUMMARY:END -->
