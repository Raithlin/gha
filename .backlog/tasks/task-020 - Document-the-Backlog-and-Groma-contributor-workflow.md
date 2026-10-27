---
id: TASK-020
title: Document the Backlog and Groma contributor workflow
status: Done
assignee:
  - '@codex'
created_date: '2026-10-25T14:20:00Z'
updated_date: '2026-10-27T19:30:00Z'
labels: []
dependencies: []
references:
  - docs/README.md
  - README.md
  - docs/development.md
modified_files:
  - docs/development.md
  - docs/README.md
  - README.md
type: docs
ordinal: 22000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The development guide explains build and release work but omits the task and architecture tools now tracked in this repository. Contributors need guidance for these optional, informational, highly recommended tools without making them build, test, or contribution prerequisites.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Development documentation presents Backlog and Groma as optional and highly recommended, with CLI examples for tasks and architecture.
- [x] #2 Documentation explains scan side effects, CLI-managed records, and task-to-architecture links.
- [x] #3 Documentation entry points link to the workflow; CLI syntax and local Markdown links are verified.
<!-- AC:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added an optional contributor workflow for Backlog.md and Groma to docs/development.md, with links from both documentation indexes. Explains why and how to use task records and architecture context without making either tool a prerequisite for building, testing, or contributing. Verified local CLI behavior, links, anchors, and diff formatting.
<!-- SECTION:FINAL_SUMMARY:END -->
