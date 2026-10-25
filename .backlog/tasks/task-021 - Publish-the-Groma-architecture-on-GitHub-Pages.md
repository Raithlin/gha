---
id: TASK-021
title: Publish the Groma architecture on GitHub Pages
status: In Progress
assignee:
  - '@codex'
created_date: '2026-10-25T14:33:00Z'
updated_date: '2026-10-25T14:34:00Z'
labels: []
dependencies: []
references:
  - .github/workflows/architecture.yml
  - README.md
  - docs/README.md
  - docs/development.md
modified_files:
  - .github/workflows/architecture.yml
type: docs
ordinal: 21000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Readers should be able to explore the curated GHA architecture from GitHub without installing Groma. Publish the committed map as a static read-only site and link it from the documentation.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A pinned Groma export of the pushed master commit deploys through GitHub Pages without running a scanner.
- [ ] #2 README and developer documentation link to the architecture site and explain publication.
- [ ] #3 The export and workflow validate locally, and the published map loads successfully.
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add a Pages workflow using pinned Groma 0.7.7 to export HEAD without scanning; separate build and deploy permissions.
2. Link the hosted map and document its refresh process. Verify export assets, CLI behavior, and workflow syntax.
3. Enable Actions-based Pages, commit and push the publication changes with the preceding contributor documentation, monitor deployment, and verify the live map.
<!-- SECTION:PLAN:END -->
