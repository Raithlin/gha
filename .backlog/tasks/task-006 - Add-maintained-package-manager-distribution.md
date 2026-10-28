---
id: TASK-006
title: Add maintained package-manager distribution
status: Done
assignee:
  - '@codex'
created_date: '2026-10-25T13:54:00Z'
updated_date: '2026-10-28T11:49:00Z'
labels:
  - roadmap
  - next
dependencies: []
references:
  - docs/roadmap.md
  - .goreleaser.yaml
  - .github/workflows/release.yml
  - internal/commands/update.go
  - https://github.com/Raithlin/homebrew-tap
  - https://github.com/Raithlin/homebrew-tap/actions/runs/40680667262
  - docs/design.md
  - https://github.com/Raithlin/gha/pull/31
  - https://github.com/Raithlin/gha/actions/runs/40682475290
  - https://github.com/Raithlin/gha/actions/runs/40682475242
  - https://github.com/Raithlin/gha/actions/runs/40682323115
modified_files:
  - tools/packaging/package_test.go
  - .goreleaser.yaml
  - tools/packaging/smoke-linux.sh
  - .github/workflows/release.yml
  - .github/workflows/packages.yml
  - tools/packaging/test_smoke_macos.py
  - tools/packaging/smoke_macos.py
  - docs/package-distribution.md
  - README.md
  - docs/roadmap.md
priority: medium
type: feature
ordinal: 40000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Roadmap Next item 4. Released binaries need maintained installation and upgrade paths, beginning with Homebrew and an evidence-based assessment of Linux distribution options.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A maintained Homebrew tap installs a released GHA binary and has documented installation and upgrade instructions with release asset integrity verification.
- [x] #2 Linux package-manager options are assessed for maintenance burden and workflow fit, with a recorded choice and rationale.
- [x] #3 Executable upgrades remain owned by the selected package manager; guidance refresh only applies skills whose declared capabilities are supported by the installed binary.
- [x] #4 Validation covers install/upgrade behavior and preservation of incompatible or modified agent guidance.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Roadmap is mapped and current
- [x] #2 CI passes, test coverage is 95%+
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Approved by Stephen: deliver a dedicated Raithlin/homebrew-tap with a GoReleaser-generated macOS binary cask, plus native Linux deb/rpm/Arch packages. Bootstrap tap from an existing checksum-verified release; automate subsequent updates with scoped tap credentials. Use TDD for packaging contracts and validate actual package installs/upgrades in disposable CI environments, including both macOS architectures where runners permit. Preserve existing capability and modified-skill protections, never run guidance writes from packaging hooks. Document Homebrew setup/credentials, install/upgrade/integrity checks and Linux choice. Update roadmap, review architecture freshness, run 95% quality gate, commit logical changes with one final task-record commit, and create a focused PR. Keep criteria pending until objective live tap/install and CI evidence exists.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Delivery evidence: live tap CI 40680667262 passed published v0.2.0-alpha installation and v0.3.0-alpha upgrade/removal on Intel and Apple Silicon. All eight archive downloads were verified against release checksum manifests before bootstrap. Source package CI 40682475290 passed actual two-version installs/upgrades on Debian, Fedora, Arch and both Mac architectures, including Homebrew wrong-digest rejection, Linux package ownership, and unchanged customized guidance. Go CI 40682475242 passed at 95.2% statement coverage; local make check, GoReleaser 2.19.0 validation/full snapshot packaging, Python fixture tests, shellcheck, and committed architecture-check passed. Manual read-only Release run 40682323115 verified the user-configured tap token permissions while skipping publication. No tag or release was created. Architecture refresh produced no record changes; existing guidance-update ownership and responsibilities remain accurate, with scanner coverage excluding release YAML/Python tooling. Backlog auto_commit was false and remains false. Linux packages first ship when this configuration is released; hosted package repositories and AUR submission are deferred. All task changes are included in one final task-record commit.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added maintained Homebrew distribution at Raithlin/homebrew-tap, release-generated casks with pinned archive digests, and native Debian/Fedora/Arch packages. Package managers own executable upgrades; agent guidance remains explicit and retains compatibility/customization protections. Documented installation, upgrade, integrity, credential rotation and the Linux maintenance choice. Verified live tap upgrades, all five native packaging environments, 95.2% coverage, architecture freshness and scoped token access. Implementation is ready for review in PR #31; no new release was published.
<!-- SECTION:FINAL_SUMMARY:END -->
