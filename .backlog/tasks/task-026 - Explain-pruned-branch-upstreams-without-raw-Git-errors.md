---
id: TASK-026
title: Explain pruned branch upstreams without raw Git errors
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
  - internal/git/branches_test.go
  - internal/git/branches.go
  - pkg/model/model.go
  - internal/git/coverage_test.go
  - internal/output/review.go
  - docs/command-guide.md
  - internal/output/review_test.go
  - .groma/systems/gha/containers/gha-cli/components/branch-inspection.md
priority: low
type: bug
ordinal: 56000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
After explicit origin pruning, divergence remains unavailable but includes raw ambiguous-revision usage text instead of explaining that the configured upstream is gone.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Inventory and branch inspection expose a stable missing-upstream reason and concise terminal explanation.
- [x] #2 Gone upstreams never infer ahead or behind counts, remain distinct from untracked branches and Git failures, and never cause implicit fetching.
- [x] #3 Regression tests cover pruned upstreams and ordinary divergence; documentation describes the additive contract.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Roadmap is mapped and all documentation is current
- [x] #2 CI passes, test coverage is 94.51%+
- [x] #3 No `--confirm` arguments - the default call executes, `--dry-run` is the alternative
<!-- DOD:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Inventory and inspection report upstream_gone with unavailable divergence and no inferred counts. Exact-ref checks prevent substitution by a conflicting local branch. Real-Git, error-control, and rebuilt-CLI checks pass. Hosted Build and Test passed at 4c802fb (run 40573010250); local coverage is 95.2%.
<!-- SECTION:FINAL_SUMMARY:END -->
