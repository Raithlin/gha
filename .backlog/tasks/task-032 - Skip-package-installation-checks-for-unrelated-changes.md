---
id: TASK-032
title: Skip package installation checks for unrelated changes
status: In Progress
assignee:
  - '@codex'
created_date: '2026-10-28T20:50:00Z'
updated_date: '2026-10-28T21:01:00Z'
labels: []
dependencies: []
references:
  - https://github.com/Raithlin/gha/pull/36
  - .github/workflows/packages.yml
  - docs/roadmap.md
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
- [ ] #2 Adding ci:packages runs the matrix for a PR, and later pushes or reopening rerun it while that label remains; unrelated label events do not rerun or cancel it.
- [x] #3 Release tag publication requires successful Linux and macOS package installation tests; manual release dispatch remains credential-only.
- [x] #4 Manual package runs and cancellation of superseded runs remain available.
- [x] #5 Contributor, release and package documentation describe the policy, and regression tests verify workflow wiring.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Roadmap is mapped and all documentation is current
- [ ] #2 CI passes, test coverage is 94.51%+
- [x] #3 No `--confirm` arguments - the default call executes, `--dry-run` is the alternative
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Replace path-based triggers with a reusable packages.yml workflow and a label-gated PR caller. Make tag publication depend on that same reusable matrix, preserving credential-only manual release dispatch. Revise workflow contract tests first, verify RED/GREEN, document the label in contributor/development/package guides, create the label if absent, and update PR 36 after validation.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Replaced path-based CI with a reusable installation matrix, label-gated PR caller and mandatory release publication dependency. Documented ci:packages in contributor, development/release and package guides; created the repository label. RED/GREEN workflow contract tests, full race-enabled Go suite (95.2%), packaging Python tests, lint, actionlint and architecture checks pass; 67 local documentation targets resolve. Hosted label-trigger verification and final CI remain pending on PR 36.
<!-- SECTION:FINAL_SUMMARY:END -->
