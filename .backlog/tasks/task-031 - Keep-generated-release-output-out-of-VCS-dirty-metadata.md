---
id: TASK-031
title: Keep generated release output out of VCS dirty metadata
status: In Progress
assignee:
  - '@codex'
created_date: '2026-10-28T20:29:00Z'
updated_date: '2026-10-28T20:31:00Z'
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
- [ ] #2 CI passes, test coverage is 94.51%+
- [x] #3 No `--confirm` arguments - the default call executes, `--dry-run` is the alternative
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Reproduce with a disposable Git repository and actual Go builds; ignore only root dist output; validate clean and changed source metadata, then run release snapshot and repository checks.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
RED reproduced vcs.modified=true from generated dist output. GREEN covers clean output, tracked edits, and untracked internal/dist source. make check passes with 95.2% coverage; four Python packaging checks and GoReleaser config validation pass. A clean clone snapshot builds all four binaries and six native packages; the executed Linux binary reports dirty:false and the expected 9104c30 revision. Architecture refresh/check produce no record changes. Hosted CI pending PR.
<!-- SECTION:NOTES:END -->
