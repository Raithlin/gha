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
  preparation workflow with guarded creation.
- **Releases:** published-release discovery, read-only notes, guarded release
  publication, and exact-ref tag publication.
- **Branches:** inventory, safety inspection, guarded create/publish/rename/
  delete operations, and read-only cleanup candidates.
- **Repository analysis:** offline worktree, history, storage, and large-file
  analysis.
- **Agent guidance:** install, list, and uninstall for Codex, Claude Code, Pi,
  OpenCode, GitHub Copilot, and Gemini CLI, with recorded destination ownership.
- **Automation contract:** versioned capability inventory, structured output,
  and a CI-enforced 95% statement-coverage gate.

## Next

Priorities are ordered by expected user value and prerequisite:

1. **Simplify mutation modes.** Make `--dry-run` the execution-mode switch:
   execute by default and show the plan when requested. Remove redundant
   confirmation gates while retaining explicit targets, preflight checks, and
   safety guardrails such as `--force`. Align commands, capabilities, docs, and
   tests.
2. **Draft pull-request descriptions.** Extend `pr prepare` to summarize
   base-to-head commits and changed files, then propose a title and review-ready
   body. Return the draft and its source signals for review; let `pr create` use
   the reviewed content. Preparation remains read-only.
3. **Update installed guidance.** Add `gha update` to discover the latest
   published version and refresh bundled skills only for harnesses recorded by
   `agent install`. Preserve unrelated content and report configured targets
   and unavailable sources. Decide whether it updates only guidance or also
   the executable before implementation.
4. **Improve harness setup.** Detect supported harnesses during installation,
   offer a binary-only option, and explain when none are found. Do not configure
   absent harnesses or make their executables GHA dependencies. Verify skill
   discovery. Assess Cursor, Hermes Agent, and OpenClaw only after confirming
   their current loading and safe-removal contracts.
5. **Show coverage and project status.** Publish the CI coverage result and its
   95% pass/fail status in pull requests. Add README badges for coverage, CI,
   and the latest release; include prereleases during the alpha phase. Keep the
   existing license badge and avoid manually maintained values.
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
- **Later phases:** analytics, a TUI dashboard, plugins, offline
  synchronization, and additional providers such as GitLab and Azure DevOps.
