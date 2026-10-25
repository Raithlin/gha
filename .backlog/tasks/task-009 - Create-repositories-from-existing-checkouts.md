---
id: TASK-009
title: Create repositories from existing checkouts
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
priority: medium
type: feature
ordinal: 42000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Roadmap Next item 7. Creating a provider repository from an existing checkout can require coordinated remote setup and branch publication; partial success must be recoverable. Add gha repo create only when this exceeds the value of gh repo create.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Document a Command Value Test decision establishing workflow value beyond gh repo create before adding the command.
- [ ] #2 An accepted command previews owner, name, visibility, default branch, origin target, and intended local/remote effects with --dry-run and checks repository and remote conflicts before writing.
- [ ] #3 Choosing the GitHub default branch is separate from explicit local trunk renaming; the local branch is never renamed implicitly.
- [ ] #4 The workflow supports publishing the selected committed branch and setting its upstream, with explicit mutation targets and safety guardrails.
- [ ] #5 Structured per-step outcomes and recovery guidance cover partial repository creation, remote setup, branch rename, and publication; tests, text output, capability inventory, and documentation cover those contracts.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Roadmap is mapped and current
- [ ] #2 CI passes, test coverage is 95%+
<!-- DOD:END -->
