---
id: TASK-025
title: Return release eligibility for unpublished branches
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
  - internal/release/checkout_test.go
  - internal/release/checkout.go
  - internal/release/service.go
  - docs/command-guide.md
  - internal/commands/release_publish_test.go
  - .groma/systems/gha/containers/gha-cli/components/release-checkout.md
priority: medium
type: bug
ordinal: 55000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Release dry-run aborts with raw ls-remote exit status on preparation branches absent from origin, hiding the default-branch eligibility explanation.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A missing origin branch ref returns a structured ineligible release plan with actionable branch and missing-ref blockers in JSON and text.
- [x] #2 Missing refs remain distinct from transport, authentication, and malformed evidence; none can make a release eligible.
- [x] #3 Regression tests cover unpublished branches, origin tip checks, and inspection failures; documentation explains blockers.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Roadmap is mapped and all documentation is current
- [x] #2 CI passes, test coverage is 94.51%+
- [x] #3 No `--confirm` arguments - the default call executes, `--dry-run` is the alternative
<!-- DOD:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Unpublished preparation branches return a structured blocked release plan. Missing refs remain distinct from transport, authentication, and malformed responses. Real-Git and JSON/text regressions pass. Hosted Build and Test passed at 4c802fb (run 40573010250); local coverage is 95.2%.
<!-- SECTION:FINAL_SUMMARY:END -->
