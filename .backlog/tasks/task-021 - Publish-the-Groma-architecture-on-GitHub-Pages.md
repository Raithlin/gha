---
id: TASK-021
title: Publish the Groma architecture on GitHub Pages
status: Done
assignee:
  - '@codex'
created_date: '2026-10-25T14:33:00Z'
updated_date: '2026-10-27T19:30:00Z'
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

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Published the curated Groma architecture through GitHub Pages with a pinned export and separate build/deploy permissions. Added README/index links and developer publication instructions; included the preceding optional Backlog/Groma contributor documentation. Verified npm export locally, actionlint, live map navigation and source inspection, successful Pages deployment, and successful Go CI. Site: https://raithlin.github.io/gha/.
<!-- SECTION:FINAL_SUMMARY:END -->
