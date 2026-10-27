---
id: TASK-023
title: Keep GHA architecture current when preparing pull requests
status: In Progress
assignee:
  - '@codex'
created_date: '2026-10-27T12:52:00Z'
updated_date: '2026-10-27T14:15:00Z'
labels: []
dependencies: []
references:
  - .github/workflows/architecture.yml
  - docs/development.md
  - makefile
  - AGENTS.md
  - skills/gha/SKILL.md
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
- [ ] #2 CI passes, test coverage is 94.51%+
- [x] #3 No `--confirm` arguments - the default call executes, `--dry-run` is the alternative
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Baseline: rescan a disposable clone of committed HEAD; existing curated .groma records are stable (zero Git changes).
2. Add test-first repository tooling: refresh scans the selected worktree; check scans an isolated committed snapshot and detects tracked/untracked architecture drift; PR wrapper refreshes, requires a clean committed worktree, then delegates to local GHA without committing or pushing. Dry-run checks the isolated snapshot and leaves local files untouched.
3. Add Make targets and a read-only pull_request CI job with pinned Groma; test it using clean/drift/new-record/failure fixtures. Keep source/meaning review explicit and preserve master-only publication.
4. Document local PR entrypoint and agent requirements; record changed paths immediately. Run workflow/script tests, stable real rescan, and the project quality gate. Commit code/docs logically with one final Backlog commit on a separate branch.

5. User steering: add the architecture-before-push requirement to skills/gha/SKILL.md and the installed /home/stephen/.agents/skills/gha/SKILL.md; preserve unrelated guidance and validate both skill entrypoints.

Implementation refinement: place the architecture freshness check inside the existing required Build and Test CI job, so stale records block the existing required check without changing GitHub rulesets. Verify fresh origin head equals the scanned local commit before PR creation; no fetch, commit, or push occurs implicitly.

Review follow-up: restrict the local PR helper to same-repository Raithlin/gha origins and bind its provider target explicitly; reject fork/unknown origins before scanning, document the fork-to-upstream GitHub/gh route with CI enforcement, and add regression tests. Synchronize the installed GHA skill from the canonical copy after correcting remaining confirmation wording. Leave hosted CI DoD pending and record fresh local validation without claiming hosted success.

Publication: PR #26 remains open, so create a focused stacked PR against fix/release-ci-evaluation. Run CI for all pull_request target branches so this PR and future stacked PRs receive the same architecture and Go gates. Commit follow-up implementation and one final task-record update; verify committed freshness, publish the exact head, then use make pr-dry-run and make pr.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Validation and acceptance evidence (2026-10-27):
- RED: initial workflow tests could not import the missing implementation; additional regressions demonstrated that unpublished commits could reach PR creation and missing remote branches lacked actionable guidance. GREEN: all 12 tests pass through make test-architecture.
- AC1: repository make pr refreshes records, blocks dirty source/map or unpublished/mismatched origin head, preserves literal title/body arguments, and delegates to the current branch with the repository-local GHA. No automatic commit/push. Dry-run checks an isolated committed snapshot.
- AC2/AC3: make architecture-check scans a disposable clone of committed HEAD, detects modified/untracked records, preserves the author checkout, and propagates scanner/tool/version/configuration failures. Tests also exercise shallow CI clones. CI includes tests and freshness verification in the existing required Build and Test job, with read-only permissions and non-persisted checkout credentials. Master-only publishing is unchanged.
- Real baseline scan, local refresh, isolated freshness check, and freshness check using a new temporary npm installation of pinned Groma 0.7.7 all passed with zero .groma changes. Invalid scanner configuration returned non-zero. Existing informational Go scope/external-package findings remain explicit; scanner stability is not proof of correct architectural prose.
- AC4: developer docs and AGENTS.md describe review/commit/push/PR sequencing and the semantic-review limit. actionlint passes for the modified CI workflow; documentation links and git diff --check pass.
- AC5: skills/gha/SKILL.md and /home/stephen/.agents/skills/gha/SKILL.md now require architecture freshness and affected-map review before publishing GHA architecture-related changes. Both validate with the skill-creator validator under system Python. Public GHA commands/capabilities and other repositories are unchanged.
- Full normal-shell make check passed lint, race-enabled Go tests, skill contract tests, and 95.2% coverage. Architecture workflow tests are run separately and by CI; ordinary local Go quality checks remain independent of Groma.
- Implemented on separate local branch feature/groma-pr-preparation. Hosted execution of the new CI steps is not yet verified because this branch has not been pushed; the new enforcement becomes active for PRs containing this workflow change and after merge for future PRs.

Review corrections (2026-10-27): local PR creation is explicitly scoped to branches published directly to Raithlin/gha. The helper validates the configured origin before scanning, rejects fork/unknown origins, and passes --repo Raithlin/gha explicitly. Developer, agent, and skill instructions describe the fork-to-upstream GitHub/gh route with the same architecture review and CI gate. The installed skill is synchronized byte-for-byte with the corrected canonical skill; obsolete confirmation requirements are removed. RED tests demonstrated wrong-target delegation and scanning on unsupported origins; GREEN: all 14 workflow tests pass, including HTTPS/SSH origins, fork/unknown origin rejection for execution and dry-run, and explicit provider targeting. Both skill validators and comparison pass. Full make check passes lint, race tests, skill contracts and 95.2% coverage; architecture refresh produced no record changes and the existing committed-HEAD freshness check passed. Follow-up edits remain local and uncommitted; hosted CI for these changes is still pending. Status remains In Progress with CI DoD #2 unchecked until hosted validation at the published revision succeeds.

Publication preparation: all 14 workflow tests pass again; architecture refresh produced zero record changes; actionlint and git diff --check pass after expanding pull_request triggers to all target branches. PR #26 is still open, so this architecture-only PR will target fix/release-ci-evaluation and can be retargeted to master after that dependency merges. Hosted CI remains pending and DoD #2 stays unchecked. Follow-up fixes and this record are now being committed for publication.
<!-- SECTION:NOTES:END -->
