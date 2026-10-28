---
id: TASK-030
title: Make installation and first use clear in the README
status: Done
assignee:
  - '@codex'
created_date: '2026-10-28T11:25:00Z'
updated_date: '2026-10-28T11:50:00Z'
labels: []
dependencies: []
references:
  - README.md
  - docs/getting-started.md
  - /home/stephen/Work/kde-shortcuts/INSTALL.md
modified_files:
  - README.md
  - docs/getting-started.md
type: docs
ordinal: 68000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The README leads with a development build and mixes first use with contributor and agent internals. Follow the Plasma Shortcut Explorer documentation pattern so users can install a release and reach a useful first command.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 README explains GHA and offers a short path from installation to first useful commands.
- [x] #2 Setup guide covers release downloads, verification, authentication, PATH, updates and removal with current supported platforms.
- [x] #3 Documented commands match the local capability inventory and local documentation links resolve.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Roadmap is mapped and all documentation is current
- [x] #2 CI passes, test coverage is 94.51%+
- [x] #3 No `--confirm` arguments - the default call executes, `--dry-run` is the alternative
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Use the Plasma Shortcut Explorer split between a short user README and detailed setup guide. Rewrite README around purpose, install, first use and optional agent setup; expand docs/getting-started.md with verified release assets, platform selection, checksum and PATH steps, authentication, updates and uninstall. Validate documented command forms against the rebuilt local inventory and check local links. Keep release and development capabilities distinct; no runtime changes.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Reworked README around release installation, first PR review, common workflows and optional agent setup, following Plasma Shortcut Explorer. Expanded getting-started with platform selection, checksums, PATH, source builds, authentication, updates and removal. Clarified that gh is optional: GHA prefers GHA_GITHUB_TOKEN, otherwise tries gh auth token, then continues unauthenticated. Verified published Linux archive checksum, extraction and live PR listing; current local command forms, text/JSON results, previews and documentation links. make check passed lint and race tests with 95.2% coverage; git diff --check passed. Documentation-only change; no runtime changes.
<!-- SECTION:FINAL_SUMMARY:END -->
