---
id: TASK-020
title: Document the Backlog and Groma contributor workflow
status: Done
assignee:
  - '@codex'
created_date: '2026-10-25T14:20:00Z'
updated_date: '2026-10-25T14:22:00Z'
labels: []
dependencies: []
references:
  - docs/development.md
  - docs/README.md
  - README.md
modified_files:
  - docs/development.md
  - docs/README.md
  - README.md
type: docs
ordinal: 20000
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

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Document Backlog and Groma as optional, informational, highly recommended contributor tools.
2. Add practical task, architecture inspection, and linked-work examples to the development guide; link them from documentation entry points.
3. Verify CLI syntax, expected results, Markdown links, and the final diff; record evidence and finish the documentation task.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Documented both tools as optional, informational, and highly recommended. Added task lifecycle examples, Groma read-only inspection and scan side effects, CLI-only managed-record maintenance, source ownership links, and modified-file replacement semantics. Linked from README.md and docs/README.md. Verified syntax against Backlog 1.55.3 and Groma 0.7.7 help/instructions; exercised stored-map, container, source-owner, and full-component reads. All 32 local documentation links and section anchors resolve; git diff --check passes. Documentation-only changes; no Go tests rerun and no architecture scan or curation performed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added an optional contributor workflow for Backlog.md and Groma to docs/development.md, with links from both documentation indexes. Explains why and how to use task records and architecture context without making either tool a prerequisite for building, testing, or contributing. Verified local CLI behavior, links, anchors, and diff formatting.
<!-- SECTION:FINAL_SUMMARY:END -->
