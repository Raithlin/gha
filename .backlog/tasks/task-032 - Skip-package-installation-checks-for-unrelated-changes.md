---
id: TASK-032
title: Skip package installation checks for unrelated changes
status: Done
assignee:
  - '@codex'
created_date: '2026-10-28T20:50:00Z'
updated_date: '2026-11-13T07:57:00Z'
labels: []
dependencies: []
references:
  - https://github.com/Raithlin/gha/pull/36
  - .github/workflows/packages.yml
  - docs/roadmap.md
  - https://github.com/Raithlin/gha/pull/41
modified_files:
  - tools/packaging/workflow_test.go
  - .github/workflows/packages.yml
  - docs/package-distribution.md
  - docs/roadmap.md
  - tools/packaging/package_test.go
  - .github/workflows/package-pr.yml
  - .github/workflows/release.yml
  - docs/CONTRIBUTING.md
  - docs/development.md
  - tools/packaging/test_smoke_macos.py
  - tools/packaging/smoke_macos.py
type: chore
ordinal: 70000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Documentation and image-only pull requests currently build packages and run five installation environments, wasting CI resources without exercising a relevant change.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Ordinary PRs and master pushes do not run the package installation matrix.
- [x] #2 Adding ci:packages runs the matrix for a PR, and later pushes or reopening rerun it while that label remains; unrelated label events do not rerun or cancel it.
- [x] #3 Release tag publication requires successful Linux and macOS package installation tests; manual release dispatch remains credential-only.
- [x] #4 Manual package runs and cancellation of superseded runs remain available.
- [x] #5 Contributor, release and package documentation describe the policy, and regression tests verify workflow wiring.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Roadmap is mapped and all documentation is current
- [x] #2 CI passes, test coverage is 94.51%+
- [x] #3 No `--confirm` arguments - the default call executes, `--dry-run` is the alternative
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Replace path-based triggers with a reusable packages.yml workflow and a label-gated PR caller. Make tag publication depend on that same reusable matrix, preserving credential-only manual release dispatch. Revise workflow contract tests first, verify RED/GREEN, document the label in contributor/development/package guides, create the label if absent, and update PR 36 after validation.

Hosted PR #41 package CI exposed a Homebrew tap trust gate before the checksum check. Add an explicit trust step for the disposable local test tap, with a regression test for its command; rerun the package smoke tests and hosted matrix on the same labelled PR.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Reusable installation matrix, label-gated PR caller, and release publication dependency are implemented and documented. Local workflow contract tests, race-enabled Go suite (95.2% coverage), packaging Python tests, lint, actionlint, architecture checks, and 67 documentation links passed. Hosted PR #36 Go CI run 40768446597 passed with 95.2% coverage. PR event labeled ci:packages at 2026-10-28T21:01:43Z; package run 40768448511 began at 21:01:45Z for head 6f7f4f667e025e8c4297db44885fa9309d9122fb and passed all five Linux/macOS install environments. Workflow source and TestPackagePROptIn cover opened, reopened, synchronize, labeled, and unrelated-label guards. Hosted history does not show a later push or reopening while ci:packages remained; acceptance criterion #2 remains open. PR #36 merged as 27a7fabce702e7c934a41843f2f7293a653be8e4.

PR #41 label-triggered run 43324321920 and later-push run 43388590750 both reached macOS smoke testing, where Homebrew refused the local gha-ci/packages tap before checking the intentionally wrong digest. This is a CI fixture compatibility repair needed to validate the package matrix.

PR #41 package runs 43324321920 (label event, head 801369e) and 43388590750 (later push, head 61e5c17) each started the full matrix. Unrelated enhancement label event produced skipped run 43324374291 and did not cancel the first run. The later push run passed build, three Linux jobs, and Intel macOS; arm64 macOS failed solely at the temporary tap trust gate. Local Python smoke tests pass after explicit tap trust; hosted rerun pending.

PR #41 head 7572f5b package run 43389114503 completed successfully after the disposable Homebrew tap trust fix: build, deb, rpm, Arch, macOS arm64, and macOS Intel all passed. The labelled run 43324321920, unrelated-label skipped run 43324374291, and later-push full runs 43388590750/43389114503 prove criterion #2's hosted event behavior. Go CI and CodeQL also passed on the repaired head.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Completed the `ci:packages` opt-in policy and verified it on PR #41: adding the label and later pushes started the full matrix, while an unrelated label event produced only a skipped caller and did not cancel the running matrix. Fixed the disposable macOS Homebrew tap trust fixture exposed by current CI. The final hosted package run passed in all five install environments; Go CI, CodeQL, local checks, and architecture checks passed.
<!-- SECTION:FINAL_SUMMARY:END -->
