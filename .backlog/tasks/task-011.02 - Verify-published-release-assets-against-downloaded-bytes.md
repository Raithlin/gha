---
id: TASK-011.02
title: Verify published release assets against downloaded bytes
status: In Progress
assignee:
  - Stephen
created_date: '2026-10-27T19:08:00Z'
updated_date: '2026-11-15T18:36:00Z'
labels: []
dependencies: []
references:
  - docs/roadmap.md
  - https://github.com/Raithlin/plasma-shortcut-explorer/releases/tag/v0.1.4
  - internal/commands/release.go
  - internal/release/status.go
  - internal/github/release_status.go
  - internal/output/release_status.go
  - internal/commands/capabilities.go
  - release-commands
  - github-provider
  - result-rendering
  - release-assets
documentation:
  - docs/roadmap.md
modified_files:
  - internal/release/assets_test.go
  - internal/release/assets.go
  - internal/github/release_assets_test.go
  - internal/github/release_assets.go
  - internal/commands/release_assets_test.go
  - internal/commands/release_assets.go
  - internal/output/release_assets.go
  - internal/commands/release.go
  - internal/commands/capabilities.go
  - docs/command-guide.md
  - docs/roadmap.md
  - internal/commands/capabilities_test.go
  - README.md
  - skills/gha/SKILL.md
  - internal/output/release_assets_test.go
  - .groma/systems/gha/containers/gha-cli/components/assets.md
  - .groma/systems/gha/containers/gha-cli/components/commands-release-assets.md
  - .groma/systems/gha/containers/gha-cli/components/github-release-assets.md
  - .groma/systems/gha/containers/gha-cli/components/output-release-assets.md
  - .groma/systems/gha/containers/gha-cli/components/release-commands.md
  - .groma/systems/gha/containers/gha-cli/components/github-provider.md
  - .groma/systems/gha/containers/gha-cli/components/result-rendering.md
  - .groma/systems/gha/containers/gha-cli/components/release-assets.md
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
- [x] #1 A bounded release asset inventory reports stable asset identities, names, sizes, provider digests, draft/prerelease state, and completeness.
- [x] #2 An explicit verification workflow checks downloaded bytes against supported provider digests and optional checksum manifests, including the manifest itself when a provider digest exists.
- [x] #3 Missing or unsupported digests, incomplete inventories, missing or unsafe manifest entries, and mismatched bytes have explicit states and never imply verification success.
- [x] #4 Download destinations and existing-file behavior are explicit and safe; remote release state is never modified.
- [x] #5 JSON and terminal contracts, capability discovery, documentation, and regressions cover valid assets and failure or unavailable evidence.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Roadmap is mapped and all documentation is current
- [ ] #2 CI passes, test coverage is 94.51%+
- [x] #3 No `--confirm` arguments - the default call executes, `--dry-run` is the alternative
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Implement `release assets <tag>` with a bounded GitHub asset inventory, stable IDs, release draft/prerelease metadata, digests, and explicit completeness.
2. Implement `release verify <tag> --output-dir <existing-dir>` with --dry-run. Reject unsafe, duplicate, and occupied destinations. Stage authenticated downloads, compare size and SHA-256 provider digests, then place verified files without overwriting existing files.
3. Check optional SHA256SUMS or checksums.txt entries and the manifest's own provider digest. Expose per-asset missing, unsupported, unavailable, mismatch, and verified states; report incomplete inventories as structured non-success.
4. Cover provider pagination and failures, safe destinations, byte and manifest verification, JSON/text output, and capability discovery. Update README, command guide, bundled skill, roadmap, and Groma meaning. Rebuild, verify a live hosted release, run make check and architecture tests, then open a PR and confirm hosted CI before finalizing the task.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented inventory and verification commands; live GHA v0.4.0-beta.1 verification matched all 11 provider digests and checksums.txt entries. `make check` currently passes lint and tests but coverage is 93.7% against the 95% gate. Adding focused failure-path regressions before finalization.

Final local validation: `make check` passed lint, tests, and 95.0% coverage with isolated HOME/CODEX_HOME; `make test-architecture` passed 14 tests; `git diff --check` passed. Live `release assets --limit 1` showed `truncated: true` and `complete: false` in JSON and text. Live `release verify --limit 1 --dry-run` returned structured `incomplete` and exit 1. Full v0.4.0-beta.1 verification downloaded and matched all 11 provider digests and checksums.txt entries. Groma 0.7.7 refresh created no new records on the second scan after reviewed curation. Hosted CI remains to be checked on the PR.
<!-- SECTION:NOTES:END -->
