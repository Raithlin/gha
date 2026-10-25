---
id: TASK-021
title: Publish the Groma architecture on GitHub Pages
status: Done
assignee:
  - '@codex'
created_date: '2026-10-25T14:33:00Z'
updated_date: '2026-10-25T14:38:00Z'
labels: []
dependencies: []
references:
  - .github/workflows/architecture.yml
  - README.md
  - docs/README.md
  - docs/development.md
modified_files:
  - .github/workflows/architecture.yml
  - README.md
  - docs/README.md
  - docs/development.md
type: docs
ordinal: 21000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Readers should be able to explore the curated GHA architecture from GitHub without installing Groma. Publish the committed map as a static read-only site and link it from the documentation.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A pinned Groma export of the pushed master commit deploys through GitHub Pages without running a scanner.
- [x] #2 README and developer documentation link to the architecture site and explain publication.
- [x] #3 The export and workflow validate locally, and the published map loads successfully.
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add a Pages workflow using pinned Groma 0.7.7 to export HEAD without scanning; separate build and deploy permissions.
2. Link the hosted map and document its refresh process. Verify export assets, CLI behavior, and workflow syntax.
3. Enable Actions-based Pages, commit and push the publication changes with the preceding contributor documentation, monitor deployment, and verify the live map.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented separate export/deploy jobs with Groma 0.7.7 pinned, Node 24, master-only deployment, Pages URL from configure-pages, and temporary artifact output. Local export from the pinned npm package renders correctly in the browser. actionlint v1.7.12 and git diff --check pass; documentation links resolve. Enabled GitHub Pages build_type=workflow at https://raithlin.github.io/gha/. Deployment and live verification remain pending.

Published commit 91b6b8359d2c8380417316b662a4d5df62168070 to master. Pages run 40203011112 and Go CI run 40203011068 both succeeded. Live browser verification at https://raithlin.github.io/gha/ confirmed the deployed revision, system/container/component navigation, Branch commands details, owned source list, and rendered internal/commands/branch.go contents. No scanner or curation ran. Unrelated local AGENTS.md and .backlog/config.yml edits remain untouched.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Published the curated Groma architecture through GitHub Pages with a pinned export and separate build/deploy permissions. Added README/index links and developer publication instructions; included the preceding optional Backlog/Groma contributor documentation. Verified npm export locally, actionlint, live map navigation and source inspection, successful Pages deployment, and successful Go CI. Site: https://raithlin.github.io/gha/.
<!-- SECTION:FINAL_SUMMARY:END -->
