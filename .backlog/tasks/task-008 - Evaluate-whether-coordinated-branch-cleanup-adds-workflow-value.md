---
id: TASK-008
title: Safely clean up merged PR branches and leftover worktrees
status: In Progress
assignee:
  - codex
created_date: '2026-10-25T13:54:00Z'
updated_date: '2026-10-29T10:10:00Z'
labels:
  - roadmap
  - next
dependencies: []
references:
  - docs/roadmap.md
  - docs/design.md
  - internal/git/cleanup.go
  - internal/commands/branch.go
  - docs/pr-worktree-lifecycle.md
  - branch-service
  - domain-models
  - branch-commands
  - branch-writer
  - result-rendering
modified_files:
  - docs/roadmap.md
  - internal/branch/cleanup_test.go
  - pkg/model/model.go
  - pkg/model/cleanup.go
  - internal/branch/cleanup.go
  - internal/commands/cleanup_evidence_test.go
  - internal/commands/branches.go
  - internal/output/review.go
  - internal/git/cleanup_worktree_test.go
  - internal/git/cleanup.go
  - internal/commands/capabilities.go
  - README.md
  - docs/command-guide.md
  - .groma/systems/gha/containers/gha-cli/components/branch-service.md
  - .groma/systems/gha/containers/gha-cli/components/domain-models.md
  - internal/commands/capabilities_test.go
  - .groma/systems/gha/containers/gha-cli/components/branch-commands.md
  - .groma/systems/gha/containers/gha-cli/components/branch-inspection.md
  - internal/output/cleanup_test.go
  - internal/git/cleanup_delete_test.go
  - internal/git/cleanup_delete.go
  - internal/git/branches.go
  - internal/commands/cleanup_apply_test.go
  - internal/commands/cleanup_apply.go
  - internal/output/cleanup.go
  - internal/git/cleanup_test.go
  - skills/gha/SKILL.md
  - docs/design.md
  - docs/architecture.md
  - .groma/systems/gha/containers/gha-cli/components/branch-writer.md
  - .groma/systems/gha/containers/gha-cli/components/result-rendering.md
priority: high
type: feature
ordinal: 61000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GHA currently identifies cleanup candidates through commit reachability, so branches from merged squash or rebase PRs can be excluded even though their work has landed. Leftover linked worktrees also block branch cleanup, including /tmp/gha-task-006-worktree. Users must manually cross-check PRs, inspect worktrees and consider force deletion. Provide a safe cleanup workflow that combines merged-PR evidence with local branch and worktree state, without losing unpublished work.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Cleanup recognises branches associated with merged PRs across merge-commit, squash and rebase strategies; a missing remote branch alone is never evidence of a merged PR.
- [x] #2 Candidate decisions bind the PR and repository to the relevant branch commits; ambiguous or reused branch names, open or unmerged PRs, and commits added after the merged PR are retained with actionable reasons.
- [x] #3 Inspection is read-only, bounded and available as JSON and readable text, with merge evidence, freshness, unavailable signals and exclusion reasons explicitly represented.
- [x] #4 Deletion supports explicit branch selection and local/origin targets with a complete dry-run preview; verified squash or rebase merges do not require users to bypass safety with force.
- [x] #5 Coordinated cleanup reports per-branch completed, retained and failed effects, with recovery guidance after partial failure.
- [x] #6 Regression tests cover all merge strategies, stale or unavailable provider evidence, reused branch names, post-merge local commits and worktree safety; help and user documentation explain the complete workflow.
- [x] #7 Default/base branches and primary/current checkouts remain protected. Branches used by another worktree are retained unless that linked worktree is explicitly selected and safely retired in the same reviewed cleanup plan.
- [x] #8 Cleanup identifies linked worktrees that block merged-branch deletion and supports explicitly retiring eligible leftovers before deleting their branches; dirty, untracked or ignored content and unpublished commits are retained unless an explicitly named branch force override is confirmed. Locked worktrees remain protected.
- [x] #9 A repeatable --force <branch> applies only to supplied selected branches, shows the exact destructive targets, and requires agent/user confirmation of each exact branch name before any writes. Dry-run never prompts; missing confirmation retains the entire plan.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Roadmap is mapped and current
- [ ] #2 CI passes, test coverage is 95%+
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add bounded fresh merged-PR inspection with exact repository/base/head binding while preserving offline reachability inspection. 2. Coordinate explicit branch, local/origin and linked-worktree targets through complete dry-run plans and fresh revalidation. 3. Apply named force overrides only after exact branch-name confirmation for every forced branch before any writes. Preserve default/base, primary/current, lock and origin-policy protections. 4. Verify merge strategies, stale/reused heads, worktree content, partial effects, tip leases, configuration cleanup and confirmation boundaries with regression tests. 5. Update command/agent documentation and reviewed Groma ownership and relationships; run make check and architecture-check. Hosted CI remains a publication-stage verification.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented fresh merged-PR inspection and selected local/origin cleanup, with exact tip checks, linked-worktree retirement and confirmed branch-specific force. Local make check passed with lint, race-enabled tests and 95.3% coverage; committed architecture freshness passed. Live inspection and force dry-run were read-only; no real cleanup ran. All nine acceptance criteria are verified locally. Publishing for hosted CI; keep In Progress until its Definition of Done is verified.
<!-- SECTION:NOTES:END -->
