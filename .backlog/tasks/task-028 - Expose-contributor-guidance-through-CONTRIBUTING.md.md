---
id: TASK-028
title: Expose contributor guidance through CONTRIBUTING.md
status: Done
assignee:
  - '@codex'
created_date: '2026-10-27T19:13:00Z'
updated_date: '2026-10-27T19:54:00Z'
labels: []
dependencies: []
references:
  - https://github.com/rails/rails/blob/main/CONTRIBUTING.md
  - >-
    https://docs.github.com/en/communities/setting-up-your-project-for-healthy-contributions/setting-guidelines-for-repository-contributors
  - https://github.com/Raithlin/gha/pull/28
modified_files:
  - docs/development.md
  - docs/CONTRIBUTING.md
  - README.md
  - AGENTS.md
  - skills/gha/SKILL.md
  - tools/architecture/pr.py
  - docs/command-guide.md
  - docs/README.md
  - >-
    .backlog/tasks/task-020 -
    Document-the-Backlog-and-Groma-contributor-workflow.md
  - .backlog/tasks/task-021 - Publish-the-Groma-architecture-on-GitHub-Pages.md
  - >-
    .backlog/tasks/task-022 -
    Browse-master-architecture-history-and-read-full-descriptions.md
  - >-
    .backlog/tasks/task-023 -
    Keep-GHA-architecture-current-when-preparing-pull-requests.md
type: docs
ordinal: 60000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GitHub should surface a concise contributor-facing guide. Following the Rails example, help people report bugs, propose changes, improve documentation, and prepare useful pull requests, with detailed development commands kept in docs/development.md.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 docs/CONTRIBUTING.md provides project-specific guidance for bug reports, feature proposals, documentation changes, pull requests, and respectful collaboration.
- [x] #2 docs/development.md retains the detailed development reference, and both guides and maintained documentation/tooling links point to the appropriate destination.
- [x] #3 Guidance reflects actual GHA contracts and available repository channels, avoids inventing policies or support promises, and passes link and tooling checks.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Roadmap is mapped and all documentation is current
- [x] #2 CI passes, test coverage is 94.51%+
- [x] #3 No `--confirm` arguments - the default call executes, `--dry-run` is the alternative
<!-- DOD:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added a contributor-facing guide for bug reports, feature proposals, documentation, patches, and PRs; retained docs/development.md as the detailed technical reference. GitHub Issues was enabled with user approval. Verified 92 local links and 14 architecture-tool tests. Hosted Build and Test passed at 4c802fb (run 40573010250).
<!-- SECTION:FINAL_SUMMARY:END -->
