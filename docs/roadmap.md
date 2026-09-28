# Roadmap

GHA adds value when it combines signals, provides a stable automation contract,
makes scope or risk explicit, or guides a safe multi-step workflow. It
complements Git and `gh`; it should not duplicate a direct command that is
clearer. See the [Command Value Test](../DESIGN.md#command-value-test).

When GHA automates an explicitly requested outcome, it must inspect the
relevant state, show planned local and remote effects, respect dry-run and
confirmation boundaries, and report what actually happened.

## Delivered

- **Pull requests:** bounded listings, decision-ready review, and a read-only
  preparation workflow with guarded creation and bounded, source-reported local
  description drafts. Reviewed pull-request content can be created by default
  while `--dry-run` previews without writing. Mutation mode simplification is
  complete: `--dry-run` selects preview mode, with explicit targets, preflight
  checks, and safety guardrails retained.
- **Releases:** published-release discovery, read-only notes, guarded release
  publication, and exact-ref tag publication.
- **Branches:** inventory, safety inspection, create/publish/rename/delete
  workflows, read-only cleanup candidates, and provider safety checks bound to
  the origin write target.
- **Repository analysis:** offline worktree, history, storage, and large-file
  analysis.
- **Agent guidance:** install, list, and uninstall for Codex, Claude Code, Pi,
  OpenCode, GitHub Copilot, Gemini CLI, Cursor, Hermes Agent, and OpenClaw, with
  recorded destination ownership and modified-skill preservation.
- **Guidance updates:** `gha update` discovers the latest published release and
  refreshes skills and managed instructions at recorded destinations; the GHA
  executable is not updated.
- **Harness setup:** `gha agent install` detects configured supported harnesses,
  installs without requiring their executables, supports `--binary-only`, and
  explains how to continue when no harness is detected. Skill paths were checked
  against current harness documentation.
- **Compact default guidance:** a held-out local pilot and PR/release check
  supported shortening the always-present GHA instructions while keeping the
  detailed skill installed for explicit use. Managed and user-modified skill
  preservation remains in place.
- **Automation contract:** versioned capability inventory, structured output,
  and a CI-enforced 95% statement-coverage gate.

## Next

Priorities are ordered by expected user value and prerequisite:

1. **Make branch dry runs validate the proposed operation.** Run the same
   read-only ref, collision, origin, and provider safety checks needed for
   branch create, rename, and delete before reporting a plan. A missing start
   ref or target branch, or an unsafe origin operation, must not appear as an
   executable `planned` result. Keep dry runs free of Git and provider writes.
2. **Report partial branch mutations.** If any branch write or checkout switch
   succeeds before a later step fails, return the `BranchMutation` result with
   each effect's completed or failed state and a nonzero exit code.
   Cover create with publish, origin rename, and combined local/origin delete
   failures so agents can identify the state that needs recovery.
3. **Show coverage and project status.** Publish the CI coverage result and its
   95% pass/fail status in pull requests. Add README badges for coverage, CI,
   and the latest release; include prereleases during the alpha phase. Keep the
   existing license badge and avoid manually maintained values.
4. **Package-manager distribution.** Add maintained package-manager paths for
   released GHA binaries, starting with a Homebrew tap and assessing Linux
   options. Keep executable upgrades owned by the selected package manager and
   refresh agent skills only when their declared GHA capabilities are
   supported by the installed binary.
5. **Build a proper TUI for `gha agent install`.** List detected and supported
   coding agents, let the user select one or more without typing agent names,
   and preview the destinations before applying changes. Keep `--agent` for
   scripts and retain `--binary-only`.
6. **Design cleanup actions.** Before adding writes to `branches cleanup`,
   define candidate selection, per-target confirmation, provider safety, and
   checked-out-branch transitions. The current command stays read-only.

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
