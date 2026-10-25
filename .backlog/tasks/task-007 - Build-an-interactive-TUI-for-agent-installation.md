---
id: TASK-007
title: Build an interactive TUI for agent installation
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
ordinal: 41000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Roadmap Next item 5. Interactive users currently need to know and type agent names; setup should make supported harnesses and installation destinations discoverable.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Interactive installation lists detected and supported coding agents and supports selecting one or more without typing agent names.
- [ ] #2 Before applying changes, the interaction previews the selected destinations and supports cancellation without writes.
- [ ] #3 Explicit --agent selection remains suitable for scripts, --binary-only remains available, and noninteractive input does not hang awaiting a TUI.
- [ ] #4 Tests and documentation cover selection, destination preview, cancellation, scripted use, and preservation of modified managed files.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Roadmap is mapped and current
- [ ] #2 CI passes, test coverage is 95%+
<!-- DOD:END -->
