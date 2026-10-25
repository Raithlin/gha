---
id: TASK-004
title: Tie worktree lifecycle to pull-request work
status: To Do
assignee: []
created_date: '2026-10-25T13:54:00Z'
labels:
  - roadmap
  - next
dependencies: []
references:
  - docs/roadmap.md
  - DESIGN.md
priority: high
type: feature
ordinal: 4000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Roadmap Next item 2. Agents need a PR-centered way to start, resume, and retire isolated work while considering review readiness and existing local state. The workflow must demonstrate value beyond gh pr checkout --worktree.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A documented value assessment establishes whether combining PR review, branch, and worktree signals warrants a GHA workflow; if it does not, the recorded outcome keeps lifecycle operations with Git and gh.
- [ ] #2 An accepted workflow starts or resumes a PR checkout after checking branch/path conflicts, preserves the current checkout, and returns PR identity, branch, worktree path, readiness, and next actions.
- [ ] #3 Retirement considers PR state, local changes, and unpublished commits, explains keep/remove decisions, and preserves dirty, unpublished, or ambiguous work.
- [ ] #4 Any accepted mutations have explicit targets, safety preflight and --dry-run previews; JSON and text describe actual outcomes, with tests and documentation for conflicts, resume, and safe retirement.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Roadmap is mapped and current
- [ ] #2 CI passes, test coverage is 95%+
<!-- DOD:END -->
