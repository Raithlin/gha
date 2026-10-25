---
id: TASK-006
title: Add maintained package-manager distribution
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
ordinal: 40000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Roadmap Next item 4. Released binaries need maintained installation and upgrade paths, beginning with Homebrew and an evidence-based assessment of Linux distribution options.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A maintained Homebrew tap installs a released GHA binary and has documented installation and upgrade instructions with release asset integrity verification.
- [ ] #2 Linux package-manager options are assessed for maintenance burden and workflow fit, with a recorded choice and rationale.
- [ ] #3 Executable upgrades remain owned by the selected package manager; guidance refresh only applies skills whose declared capabilities are supported by the installed binary.
- [ ] #4 Validation covers install/upgrade behavior and preservation of incompatible or modified agent guidance.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Roadmap is mapped and current
- [ ] #2 CI passes, test coverage is 95%+
<!-- DOD:END -->
