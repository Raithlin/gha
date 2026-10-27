# Roadmap

GHA adds value when it combines signals, provides a stable automation contract,
makes scope or risk explicit, or guides a safe multi-step workflow. It
complements Git and `gh`; it should not duplicate a direct command that is
clearer. See the [Command Value Test](../DESIGN.md#command-value-test).
Use `gha capabilities --format json` for the exact command inventory; this
roadmap groups shipped workflows and tracks product gaps.

Last checked against the local implementation on **25 October 2026**. Delivered
means implemented in this checkout; it does not establish availability in an
older installed binary or a published release. The source and test links below
record the basis for each status.

When GHA automates an explicitly requested outcome, it must inspect the
relevant state, show planned local and remote effects, respect dry-run and
preflight safety boundaries, and report what actually happened.

## Delivered

- **Pull requests:** bounded listings, decision-ready review, and a read-only
  preparation workflow with guarded creation and bounded, source-reported local
  description drafts. Reviewed pull-request content can be created by default
  while `--dry-run` previews without writing. Mutation mode simplification is
  complete: `--dry-run` selects preview mode, with explicit targets, preflight
  checks, and safety guardrails retained. See the [PR workflow tests](../internal/commands/pr_test.go),
  [listing tests](../internal/commands/prs_test.go), and [review service](../internal/review/service.go).
- **Releases:** published-release discovery, read-only notes, guarded release
  publication, and exact-ref tag publication. Release preflight checks committed
  reviewed notes when the selected workflow declares `GHA_RELEASE_NOTES_DIR`;
  this repository's workflow publishes those notes with GoReleaser. Tag and
  release publication report partial completion when a push fails after local
  tag creation. A successful push or observed workflow trigger does not prove a
  completed GitHub Release. See the [release service](../internal/release/service.go),
  [reviewed-notes tests](../internal/release/checkout_test.go),
  [tag tests](../internal/commands/tag_test.go), and [release workflow](../.github/workflows/release.yml).
- **Branches:** bounded inventory with explicit origin refresh, safety
  inspection, create/publish/rename/delete workflows, read-only cleanup
  candidates, provider safety checks bound to the origin write target, and
  read-only operation preflight before branch mutation plans or writes. Create,
  rename, and delete report completed and failed effects on execution failure,
  including checkout switches, with a nonzero exit and separate diagnostics.
  Origin rename updates its two remote refs atomically. Delivered in TASK-003;
  see the [branch workflow tests](../internal/commands/branch_mutation_test.go)
  and [partial-failure regressions](../internal/commands/branch_partial_test.go).
- **Git worktrees:** structured inventory and guarded add/remove with dry-run,
  default execution, dirty/locked/prunable safety, and cross-worktree branch
  deletion and cleanup checks. See the [worktree command tests](../internal/commands/worktrees_test.go)
  and [local worktree service tests](../internal/git/worktrees_test.go).
- **PR worktree lifecycle:** `pr checkout` combines fresh PR/review signals,
  isolated checkout and non-destructive resume with explicit fork push targets.
  `pr retire` explains keep/remove decisions and preserves dirty, ignored,
  unpublished, locked or ambiguous work while retaining branches. New work
  composes worktree creation with `pr create`. Delivered in TASK-004; see the
  [lifecycle guide and value assessment](pr-worktree-lifecycle.md),
  [command regressions](../internal/commands/pr_worktree_test.go) and
  [Git safety tests](../internal/git/pr_worktree_test.go).
- **Repository analysis:** offline worktree, history, storage, and large-file
  analysis. See the [analysis tests](../internal/commands/analyze_test.go).
- **Agent guidance:** install, list, and uninstall for Codex, Claude Code, Pi,
  OpenCode, GitHub Copilot, Gemini CLI, Cursor, Hermes Agent, and OpenClaw, with
  recorded destination ownership and modified-skill preservation. `agent list`
  reports recorded installations and current files, rather than discovering
  unconfigured harnesses. See the [installer tests](../internal/commands/agent_test.go)
  and [inventory tests](../internal/commands/agent_list_test.go).
- **Guidance updates:** `gha update` discovers the latest published release and
  refreshes skills and managed instructions at recorded destinations; the GHA
  executable is not updated. Updates already check the skill's declared required
  capabilities against the current binary and preserve incompatible skills.
  See the [update tests](../internal/commands/update_test.go) and
  [compatibility tests](../internal/commands/skill_compatibility_test.go).
- **Harness setup:** `gha agent install` detects configured supported harnesses,
  installs without requiring their executables, supports `--binary-only`, and
  explains how to continue when no harness is detected. Skill paths were checked
  against harness documentation during implementation. See the
  [detection and path tests](../internal/commands/agent_test.go). Installation
  automatically selects detected harnesses or accepts `--agent`; a visual
  selection and destination-preview TUI remains planned (Next #5).
- **Compact default guidance:** a held-out local pilot and PR/release check
  supported shortening the always-present GHA instructions while keeping the
  detailed skill installed for explicit use. Managed and user-modified skill
  preservation remains in place. See the [evaluation results](agent-efficiency-test-results.md)
  and [bundled guidance](../skills/gha/SKILL.md).
- **Automation contract:** `gha version` build identification, a versioned
  capability inventory, structured output, and a CI-enforced 95%
  statement-coverage gate. See the [capability inventory](../internal/commands/capabilities.go),
  [contract tests](../internal/commands/capabilities_test.go), and [CI workflow](../.github/workflows/ci.yml).

Release CI readiness now accepts completed skipped and neutral checks alongside
successful validation, reports individual blockers, and evaluates required
checks plus exact-commit workflow runs and attempts. The regression covers
native skips alongside successful Arch, Debian, and Fedora package validation.
Unavailable policy and incomplete provider inventories block publication.
See the [readiness policy](command-guide.md#release-ci-readiness),
[CI evaluator](../internal/release/ci.go), and
[provider regression tests](../internal/github/release_ci_test.go).

## Next

Priorities are ordered by expected user value and prerequisite:

Six audit priorities remain after TASK-003 delivered partial branch mutation
reporting (original priority 1) and TASK-004 delivered PR worktree lifecycle
(original priority 2). Existing building blocks and remaining gaps
are mapped here; original priority numbers are retained for task references.

| Priority | Current implementation evidence | Remaining gap |
| --- | --- | --- |
| 3. Coverage and status | [CI](../.github/workflows/ci.yml) enforces 95% coverage; the [README](../README.md) has Go and license badges. | Publish measured PR coverage and add automated coverage, CI, and release badges. |
| 4. Package managers | [GoReleaser](../.goreleaser.yaml) builds macOS/Linux archives and checksums; [skill compatibility](../internal/commands/skill_compatibility.go) is already enforced. | Add maintained package-manager distribution. Retain the existing compatibility check when refreshing guidance. |
| 5. Installer TUI | [Installer](../internal/commands/agent.go) automatically selects detected harnesses or accepts comma-separated `--agent` names; dry-run prints destinations. The text selection prompt is used by uninstall. | Add a visual multi-selection and destination-preview flow to installation. |
| 6. Coordinated cleanup | [Cleanup candidates](../internal/git/cleanup.go) are read-only; [branch delete](../internal/commands/branch.go) owns individual writes. | Keep this boundary unless a coordinated workflow meets the stated value and recovery requirements. |
| 7. Repository creation | The [command tree](../internal/commands/root.go) and [inventory](../internal/commands/capabilities.go) have no `repo create` command. | Coordinate repository creation, remote setup, and optional branch publication with per-step recovery. |

3. **Surface coverage and project status.** The 95% coverage gate is already
   enforced in CI. Publish the measured total coverage in the pull-request
   summary, then add README badges for coverage, CI, and the latest release;
   include prereleases during the alpha phase. Keep the existing license badge
   and avoid manually maintained values.
4. **Package-manager distribution.** Add maintained package-manager paths for
   released GHA binaries, starting with a Homebrew tap and assessing Linux
   options. Keep executable upgrades owned by the selected package manager and
   retain the existing check that agent skills declare GHA capabilities
   supported by the installed binary before refreshing them.
5. **Build a proper TUI for `gha agent install`.** List detected and supported
   coding agents, let the user select one or more without typing agent names,
   and preview the destinations before applying changes. Keep `--agent` for
   scripts and retain `--binary-only`.
6. **Keep cleanup read-only unless coordinated cleanup adds value.** The
   current listing returns bounded candidates with reasons and excludes the
   base, current branch, and branches checked out in other worktrees;
   `branch delete` owns guarded single-branch writes. Add a combined mutation
   workflow only if it can provide a complete per-branch preview, provider and
   worktree safety, and clear partial-failure recovery beyond reviewing
   candidates and invoking `branch delete` directly.
7. **Create repositories from existing checkouts.** Add `gha repo create` when
   it can coordinate repository creation, local remote setup, and optional
   branch publication beyond a thin wrapper around `gh repo create`. Preview
   the owner, name, visibility, default branch, origin target, and intended
   local and remote changes with `--dry-run`. Check repository and remote
   conflicts before writing. Keep choosing the GitHub default branch separate
   from explicitly renaming the local trunk branch; never rename it implicitly.
   Support publishing the selected committed branch and setting its upstream.
   Return structured per-step outcomes and recovery guidance if creation,
   remote setup, branch rename, or publication only partially succeeds.

## Deferred

Each proposal must pass the Command Value Test before entering the delivery
sequence.

- **Release inspection:** do not add `gha release show` while `gh release view`
  remains clearer. Reconsider when GHA can add actionable context or a meaningfully
  stronger stable contract.
- **PR merging:** keep `gh pr merge` as the direct execution tool until a
  broader, safety-checked completion workflow is specified.
- **Later phases:** product analytics, a TUI dashboard, plugins, offline
  synchronization, and additional providers such as GitLab and Azure DevOps.

## Evidence for agent efficiency

Before adding product analytics, maintain a reproducible paired evaluation of
agent workflows. Run the same task and repository snapshot with and without GHA,
pin the agent model and instructions, and record complete run token usage,
completion, elapsed time, and objective task correctness. Report tokens per
correct result alongside correctness; a shorter failed run is not an
improvement. Keep the raw run records and task definitions for audit. See the
[test results](agent-efficiency-test-results.md) and
[evaluation method](agent-efficiency-evaluation.md).

The [implemented-capability sweep](agent-efficiency-test-results.md#all-implemented-capabilities)
now covers PR review, branch safety, and release preparation in addition to
the offline pilot. Blind human review is still required for subjective quality
criteria. Results inform roadmap choices; they are not GHA command contracts.

Measure developer terminal effort separately: commands and characters entered,
time to a correct decision, and mistakes or rework. Compare familiar-use and
first-use paths. Agent token usage does not measure developer keystrokes.

The three-arm pilot and the 48-attempt implemented-capability sweep compared
binary-only, compact guidance, and the bundled skill with its install guidance.
The short guide matched the full skill's reviewed correctness with fewer
tokens in both runs. A follow-up [git/gh-only control](agent-efficiency-test-results.md#gitgh-only-control)
matched the short guide's 14/14 correctness on seven comparable scenarios;
the short guide used 30% fewer tokens overall, but the prompts were designed
around GHA's commands and PR inspection favored `gh`. A small independently
written local test then supported compact default guidance with the detailed
skill still installed. Next, test additional external repositories and
independent provider tasks, then measure human terminal effort.
