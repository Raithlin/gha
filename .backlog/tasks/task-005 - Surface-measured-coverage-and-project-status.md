---
id: TASK-005
title: Surface measured coverage and project status
status: Done
assignee:
  - '@codex'
created_date: '2026-10-25T13:54:00Z'
updated_date: '2026-10-28T11:12:00Z'
labels:
  - roadmap
  - next
dependencies: []
references:
  - docs/roadmap.md
  - DESIGN.md
  - .github/workflows/ci.yml
  - .github/workflows/architecture.yml
  - README.md
  - docs/development.md
  - https://github.com/Raithlin/gha/pull/29
modified_files:
  - tools/coverage/test_report.py
  - tools/coverage/report.py
  - .github/workflows/ci.yml
  - .github/workflows/architecture.yml
  - makefile
  - README.md
  - docs/development.md
  - docs/roadmap.md
priority: medium
type: enhancement
ordinal: 20000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Roadmap Next item 3. CI already enforces 95% statement coverage, but contributors and readers cannot readily see measured coverage and current build/release status.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Pull-request CI summaries publish the measured total statement coverage while retaining the existing 95% gate.
- [x] #2 README badges display coverage, CI, and latest release, including prereleases during alpha, and preserve the existing license badge.
- [x] #3 Badge values are derived from maintained automated sources rather than manually updated percentages or versions.
- [x] #4 Validate the reporting/badge paths, including an alpha prerelease, and document any required publishing configuration.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Roadmap is mapped and current
- [x] #2 CI passes, test coverage is 95%+
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add a tested Python coverage reporter using Go total statement coverage; append CI job summaries and generate a Shields endpoint with commit provenance. Keep the existing 95% CI gate. 2. Wire reporting into PR CI and generate coverage data alongside the existing Pages architecture export for its exact revision, without new credentials or external coverage services. 3. Add automated coverage, default-branch CI and latest-release badges (including alpha prereleases), preserving license; document publication and freshness. 4. Validate reporter failures and output, workflow wiring, real coverage, prerelease badge source, full quality gate and architecture freshness. Commit implementation and one final task record; leave hosted CI explicitly pending unless verified.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented in PR #29. TDD reporter regressions passed (6); full local make check and hosted Build and Test run 40677932052 passed at 95.2%. Hosted summary and unchanged 95% gate both succeeded. Actionlint and 14 architecture workflow tests passed; architecture refresh/check found no record changes. Runtime Go ownership is unchanged; Python/workflow files are outside enabled scanner coverage. Live CI/release badges verified, including v0.3.0-alpha. Real-profile generation validated the coverage endpoint and commit provenance. Public coverage badge awaits the first successful Pages deployment after merge; docs cover this initial unavailability, stale deployments and caching. No detailed coverage page was added. Backlog auto_commit was false and remains false. User authorized including existing board status/order edits.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added measured CI coverage summaries and automated README coverage, CI and alpha-aware release badges, preserving the license badge and 95% gate. Coverage data is published with the existing Pages artifact for the measured commit; no new secrets or write permissions are required. Local and hosted validation passed at 95.2%; the public coverage endpoint activates after merge and Pages deployment.
<!-- SECTION:FINAL_SUMMARY:END -->
