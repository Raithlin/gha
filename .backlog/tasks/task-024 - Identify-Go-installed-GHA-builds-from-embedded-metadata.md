---
id: TASK-024
title: Identify Go-installed GHA builds from embedded metadata
status: Done
assignee:
  - '@codex'
created_date: '2026-10-27T19:07:00Z'
updated_date: '2026-10-27T19:54:00Z'
labels: []
dependencies: []
references:
  - docs/roadmap.md
  - https://github.com/Raithlin/plasma-shortcut-explorer/releases/tag/v0.1.4
  - https://github.com/Raithlin/gha/pull/28
documentation:
  - docs/roadmap.md
modified_files:
  - internal/buildinfo/buildinfo_test.go
  - internal/commands/version_test.go
  - internal/buildinfo/buildinfo.go
  - pkg/model/model.go
  - internal/output/review.go
  - docs/command-guide.md
  - .groma/systems/gha/containers/gha-cli/components/build-identity.md
priority: medium
type: bug
ordinal: 54000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The 27 October 2026 release exercise used a Go-installed binary returning dev/none/unknown despite embedded module and VCS identity. Multiple installed copies made the tested build ambiguous.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Go-installed builds expose available embedded version, revision, timestamp, and dirty-tree status without replacing explicit release identity.
- [x] #2 Absent metadata retains honest defaults; JSON and terminal output preserve existing fields.
- [x] #3 Regression tests cover metadata fallback and release override precedence; documentation explains identity.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Roadmap is mapped and all documentation is current
- [x] #2 CI passes, test coverage is 94.51%+
- [x] #3 No `--confirm` arguments - the default call executes, `--dry-run` is the alternative
<!-- DOD:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Go-installed builds fall back to embedded module/VCS identity and report known dirty state while explicit release values retain precedence. Regression and rebuilt-binary JSON/text checks pass. Hosted Build and Test passed at 4c802fb (run 40573010250); local coverage is 95.2%.
<!-- SECTION:FINAL_SUMMARY:END -->
