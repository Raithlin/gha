---
id: TASK-011.02
title: Verify published release assets against downloaded bytes
status: To Do
assignee: []
created_date: '2026-10-27T19:08:00Z'
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
ordinal: 58000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The release exercise required another client to inventory and download assets, check all provider SHA-256 digests, and check SHA256SUMS. GHA can close the release workflow with explicit byte-verification evidence.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A bounded release asset inventory reports stable asset identities, names, sizes, provider digests, draft/prerelease state, and completeness.
- [ ] #2 An explicit verification workflow checks downloaded bytes against supported provider digests and optional checksum manifests, including the manifest itself when a provider digest exists.
- [ ] #3 Missing or unsupported digests, incomplete inventories, missing or unsafe manifest entries, and mismatched bytes have explicit states and never imply verification success.
- [ ] #4 Download destinations and existing-file behavior are explicit and safe; remote release state is never modified.
- [ ] #5 JSON and terminal contracts, capability discovery, documentation, and regressions cover valid assets and failure or unavailable evidence.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Roadmap is mapped and all documentation is current
- [ ] #2 CI passes, test coverage is 94.51%+
- [ ] #3 No `--confirm` arguments - the default call executes, `--dry-run` is the alternative
<!-- DOD:END -->
