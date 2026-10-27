---
id: TASK-023
title: Keep GHA architecture current when preparing pull requests
status: Done
assignee:
  - '@codex'
created_date: '2026-10-27T12:52:00Z'
updated_date: '2026-10-27T19:30:00Z'
labels: []
dependencies: []
references:
  - .github/workflows/architecture.yml
  - makefile
  - AGENTS.md
  - skills/gha/SKILL.md
  - docs/development.md
modified_files:
  - tools/architecture/test_pr.py
  - tools/architecture/pr.py
  - makefile
  - .github/workflows/ci.yml
  - docs/development.md
  - AGENTS.md
  - skills/gha/SKILL.md
priority: medium
type: enhancement
ordinal: 53000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The published Groma map only reflects committed architecture records, so PRs can ship code whose architecture evidence was not refreshed. Stephen requested automation for this repository only; GHA public commands and repositories without Groma must remain unchanged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A repository-local PR preparation workflow refreshes architecture evidence and stops with actionable guidance when updates need review or committing.
- [x] #2 A pull-request CI check detects scanner-generated architecture drift without modifying the author branch or publishing a PR preview.
- [x] #3 Checks preserve the curated architecture and report scanner failures or missing tooling as failures rather than fresh evidence.
- [x] #4 Tests cover clean evidence, changed and newly generated records, scanner failure, and safe PR creation gating; contributor and agent instructions explain the workflow and its semantic-review limit.
- [x] #5 The repository GHA skill and the installed skill require freshness and affected-map review before pushing architecture-related changes, scoped to GHA development.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Roadmap is mapped and all documentation is current
- [x] #2 CI passes, test coverage is 94.51%+
- [x] #3 No `--confirm` arguments - the default call executes, `--dry-run` is the alternative
<!-- DOD:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Merged repository-only architecture refresh/check and guarded PR preparation in PR #27. Contributor and agent instructions require affected-map review before publication; same-repository targeting is explicit and fork contributions use the documented upstream route. Hosted Go CI passed all quality and architecture steps at PR head 0c0ad90; merged into master as fd9cda0. Architecture semantic correctness remains an explicit review responsibility.
<!-- SECTION:FINAL_SUMMARY:END -->
