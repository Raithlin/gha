---
id: TASK-011.01
title: Follow release publication to an observed terminal outcome
status: To Do
assignee: []
created_date: '2026-10-27T19:07:00Z'
updated_date: '2026-10-27T19:21:00Z'
labels: []
dependencies: []
references:
  - docs/roadmap.md
  - https://github.com/Raithlin/plasma-shortcut-explorer/releases/tag/v0.1.4
documentation:
  - docs/roadmap.md
parent_task_id: TASK-011
priority: medium
type: feature
ordinal: 57000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Both v0.1.3 and v0.1.4 tag publications initially returned unavailable workflow observations. The first workflow later failed and the second published successfully. Re-running tag mutation is not a safe status mechanism.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A bounded read-only status and optional wait workflow correlates repository, tag, exact commit, workflow, run, and attempt and can be used independently after publication.
- [ ] #2 Results distinguish undiscovered or unavailable evidence, pending execution, failed or cancelled execution, successful execution, and hosted release availability; tag push alone never proves publication.
- [ ] #3 Wait supports timeout and cancellation with structured partial observations, and inspection never creates tags or re-runs publication.
- [ ] #4 JSON and terminal output, capability discovery, documentation, and regressions cover eventual consistency, mismatched identities, failure, timeout, and successful publication.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Roadmap is mapped and all documentation is current
- [ ] #2 CI passes, test coverage is 94.51%+
- [ ] #3 No `--confirm` arguments - the default call executes, `--dry-run` is the alternative
<!-- DOD:END -->
