---
id: TASK-031
title: Keep generated release output out of VCS dirty metadata
status: Done
assignee:
  - '@codex'
created_date: '2026-10-28T20:29:00Z'
updated_date: '2026-11-12T22:22:00Z'
labels: []
dependencies: []
references:
  - .gitignore
  - tools/packaging/package_test.go
  - internal/buildinfo/buildinfo.go
modified_files:
  - tools/packaging/build_metadata_test.go
  - .gitignore
  - docs/roadmap.md
priority: medium
type: bug
ordinal: 69000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The v0.4.0-beta.1 binaries report dirty:true because GoReleaser creates unignored root dist files before Go records VCS status. The published tag and asset checksums verify. Correct future builds without replacing the published beta.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Generated root release output leaves a clean committed build with vcs.modified=false.
- [x] #2 Tracked source changes and unrelated untracked source still produce vcs.modified=true.
- [x] #3 A regression exercises embedded Go build metadata and package validation passes.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Roadmap is mapped and all documentation is current
- [x] #2 CI passes, test coverage is 94.51%+
- [x] #3 No `--confirm` arguments - the default call executes, `--dry-run` is the alternative
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Reproduce with a disposable Git repository and actual Go builds; ignore only root dist output; validate clean and changed source metadata, then run release snapshot and repository checks.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
RED reproduced vcs.modified=true from generated dist output. Regression covers clean output, tracked edits, and untracked internal/dist source. Local make check and packaging checks passed with 95.2% coverage; a clean-clone GoReleaser snapshot built four binaries and six native packages, and its Linux binary reported dirty:false at revision 9104c30. Architecture refresh/check found no record changes. Hosted PR #35 Go CI run 40764660165 passed at head cb2014507a24142da67a722a4fe167de2de04093, including the metadata regression and 95.2% coverage enforcement. PR merged as 47cf6344d0170f4f5fd21ada197398e22b4fde90.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Ignored generated root /dist/ output and added a build metadata regression covering clean release output, tracked edits, and unrelated untracked source. Local packaging and snapshot checks passed. Hosted Go CI passed at PR #35 head with 95.2% coverage; PR #35 merged.
<!-- SECTION:FINAL_SUMMARY:END -->
