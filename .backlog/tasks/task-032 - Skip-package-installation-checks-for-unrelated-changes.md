---
id: TASK-032
title: Skip package installation checks for unrelated changes
status: In Progress
assignee:
  - '@codex'
created_date: '2026-10-28T20:50:00Z'
updated_date: '2026-10-28T20:53:00Z'
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
type: chore
ordinal: 70000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Documentation and image-only pull requests currently build packages and run five installation environments, wasting CI resources without exercising a relevant change.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Documentation and image-only changes do not trigger package installation on PRs or master pushes.
- [x] #2 Go source, embedded assets, dependencies, package tooling, release configuration and package workflow changes still trigger installation checks; manual runs remain available.
- [x] #3 New runs cancel superseded runs for the same workflow and ref.
- [x] #4 Regression tests and package documentation explain and verify the trigger policy.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Roadmap is mapped and all documentation is current
- [ ] #2 CI passes, test coverage is 94.51%+
- [x] #3 No `--confirm` arguments - the default call executes, `--dry-run` is the alternative
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Add behavior tests for package workflow change selection and concurrency, observe failure, then add matching PR/push path filters and concurrency. Include embedded skill assets as binary inputs. Document scope, run packaging tests and architecture checks, and update PR 36.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added PR/master change filters and cancellation of superseded package runs, including embedded binary inputs. Regression test failed before the change and passes afterward; full race-enabled Go suite passes at 95.2% coverage, packaging Python tests and lint pass. Hosted CI is pending on PR 36; keep In Progress until that result is known.
<!-- SECTION:FINAL_SUMMARY:END -->
