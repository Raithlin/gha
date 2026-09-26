# Roadmap

GHA follows a phased approach to deliver workflows incrementally while
maintaining a stable automation contract.

## Command-value guardrail

GHA does not replace `git` or `gh`. Every roadmap item must pass the
[Command Value Test](../DESIGN.md#command-value-test): it must combine signals,
provide a stable machine contract, make scope or risk explicit, or guide a
safe multi-step workflow. When a direct `git` or `gh` command is clearer, GHA
should say so in its help and documentation rather than duplicate it.

### Workflow automation

When a developer or agent explicitly asks for an outcome, GHA may orchestrate
the necessary Git and provider operations as one workflow. It must first
inspect the relevant state, make the planned local and remote effects explicit,
honour dry-run and confirmation boundaries, and return the steps that actually
occurred. Git and `gh` remain the underlying tools; GHA contributes the
decision-ready plan, safety checks, and reliable result contract.

## Delivered foundation

- Basic CLI structure with Cobra and a Go module
- Build system (`makefile`)
- Versioned, machine-readable command capability inventory
- Race-enabled CI tests with a failing gate below 95% total statement coverage
- GitHub-backed pull-request listings and single-PR review with decision-ready structured output
- Agent guidance installation and managed guidance removal for Codex and Claude Code, with bundled skills and explicit write confirmation
- `gha agent list` reports GHA-configured harnesses, managed destinations, and current guidance/skill file presence
- Versioned agent destination ownership tracking, recorded harness configuration, and shared-path retention during uninstall

## Delivered core workflows

- Pull-request listings and filtering with a bounded, provider-normalized contract
- Repository-scoped review workflow with review state, risk signals, recommended actions, CI check-run, and unresolved-thread signals
- Bounded published-release discovery with a versioned contract that excludes drafts
- Read-only release notes and contributor summaries from merged pull requests
- Local and `origin` branch inventory with tracking, divergence, cached freshness, and an explicit confirmed origin refresh
- Read-only `branch show` safety inspection
- Offline local Git repository analysis of worktree, history, object storage, and largest tracked files
- Pull-request preparation and creation with explicit base/head resolution, comparison, existing-PR detection, dry runs, and confirmation
- Guarded publication of existing committed local branches with explicit origin target, upstream/divergence, provider push permission, dry runs, and confirmation
- Guarded local and origin branch creation, renaming, and deletion, including a safe checkout transition before deleting a checked-out non-default branch
- Read-only, bounded local branch cleanup candidates with documented reachability and exclusion rules
- Guarded annotated SemVer tag publication with origin, CI, and tag-triggered workflow preflight and explicit workflow observation
- Reviewed, versioned release notes consumed by GoReleaser, with a committed-notes preflight before tag publication

### Delivered branch lifecycle

The branch command surface is `gha branches` for inventory and `gha branch`
subcommands for inspection and write operations. Mutations declare
whether they affect the local repository, `origin`, or both; support `--dry-run`;
and require confirmation before remote changes. Default and protected branches
are guarded from destructive operations unless `--force` deliberately overrides
the safety guardrail. GHA branch writes earn their place only when they provide
that reviewable plan or provider-enriched safety context; use direct Git for a
simple ref operation.

Working-tree operations such as switching branches, staging, and committing
remain intentionally outside GHA's scope: native Git commands are clearer for
those direct local operations. GHA should instead own the workflows that add
safe, decision-ready context around them.

For example, deleting a merged branch that is currently checked out is not a
raw `git branch -d` replacement: GHA verifies that it is not the default or a
protected branch, switches to the default branch when safe, deletes the chosen
local and/or origin refs after confirmation, and records the checkout
transition in its result.

### Delivered cleanup-candidate review

`gha branches cleanup` is a read-only candidate review. A local branch is a
candidate only when its tip is reachable from a selected local base branch;
the default base is the cached `origin/HEAD` branch and `--base` selects one
explicitly. The output records every bounded reviewed branch either as a
candidate or with an exclusion reason. It never calls a branch "stale", never
infers provider safety data, and never deletes branches. A future cleanup
action requires a separate roadmap item and must use the existing dry-run,
confirmation, provider-safety, and checkout-transition rules.

### Release command migration

`gha release create-notes --since ...` generates read-only release notes. Its
explicit name avoids suggesting that GHA creates or displays a GitHub Release.
The command contract is:

```bash
gha releases                         # List published GitHub releases
gha release show v0.1-alpha          # Inspect one published release
gha release create-notes --since ... # Generate notes from merged pull requests
```

`gha releases` is available as a bounded, versioned discovery workflow that
excludes drafts and makes truncation explicit. `gha release show` should be
added only when it combines release data with decision-ready local or provider
context, or offers a stable automation contract that direct `gh release` output
cannot. Neither command should become an alias for the corresponding `gh`
command. `gha release --since ...` is not supported.

### Delivered release publication

`gha release publish <version>` inspects the selected checkout and commit,
fresh origin branch tip, existing local and origin tags, CI checks, and a
committed GitHub Actions workflow with a matching tag push trigger. Its dry
run names the annotated tag and planned local and origin effects. An explicit
version and `--confirm-origin` are required to publish; the command rechecks
the plan before creating and pushing the tag. It reports the tag push and
release workflow as triggered, completed, failed, or unavailable without
claiming a GitHub Release was created before the workflow proves it. The
workflow's own CI and release steps decide whether the release succeeds; GHA
does not run language-specific build or release tools. Release notes remain a
separate read-only input from `gha release create-notes`. In this repository,
the release workflow also requires a reviewed `docs/releases/<tag>.md` file in
the tagged commit. GHA checks it before creating the tag.

`gha tag publish <name>` covers release and non-release tags. It resolves the
selected commit (default `HEAD`), checks local and origin tag collisions, and
shows both planned effects. `--confirm-origin` is required to create the local
tag and push exactly that ref; a failed push reports the completed local tag
and failed origin effect.

## Prioritized next delivery

The 95% statement-coverage gate is enforced by `make check` and CI. Guarded
release publication, general-purpose one-step tag publication, and PR coverage
gating are delivered. The remaining work is ordered by prerequisite and
expected user value:

1. **Delivered: refactor guidance installation around shared destinations and ownership.**
   `agent install` records configured harnesses and exact instruction/skill
   destinations in a versioned manifest under the user GHA config directory.
   `agent uninstall` uses those recorded paths, retains destinations still
   shared by another configured harness, and removes ownership records as
   harnesses are removed.
   `gha agent list` exposes the recorded harnesses, paths, and current file
   presence in text, JSON, and YAML. It does not infer configuration for
   agents that GHA has not recorded.
2. **Add `gha update` for installed agent guidance.** Discover the latest
   published GHA version from GitHub, retrieve its bundled `skills.md`, and
   refresh the skill only in harnesses recorded as configured by
   `gha agent install`. Reuse the shared destination and ownership model above,
   preserve unrelated content, and make the configured harnesses, result, and
   any unavailable update source explicit. Define whether this command updates
   only guidance or also the GHA executable before implementation; do not imply
   a binary update if only the skill file was refreshed.
3. **Extend coding-agent guidance support (in progress).** Added Pi, OpenCode,
   GitHub Copilot, and Gemini CLI to `gha agent install` and `gha agent uninstall`,
   with comma-separated selection and shared skill ownership. Cursor remains
   pending until a supported global installation and safe removal contract is
   established; its documented guidance is project-scoped or stored in UI
   settings, and no global skill path was verified. Remaining work includes
   installation-time harness detection and setup: prompt during interactive GHA
   installation, detect supported harnesses noninteractively, report configured
   harnesses or why none were found, and offer a binary-only opt-out. Do not
   create configuration for absent harnesses or require their executables as
   GHA dependencies. Verify skill discovery in each supported harness. Evaluate
   Hermes Agent and OpenClaw after checking their current skill loading,
   configuration, and safe removal behavior.
4. **Report coverage in PRs and show useful README badges.** Publish the total
   statement coverage measured in CI in a pull request-visible check summary,
   including the 95% pass/fail result. Add a README coverage badge backed by
   the same CI measurement, a badge for the default branch's CI status, and a
   badge linking to the latest published release. Include prereleases in the
   release badge while GHA is in its alpha phase. Keep the existing license
   badge and 95% CI failure gate; avoid manually maintained percentages and
   badges without a useful destination or current project signal.
5. **Specify a cleanup action separately.** Extend `gha branches cleanup` only
   after a dedicated design defines how a user selects reviewed candidates,
   how each local and origin target is confirmed, and how the existing
   provider-safety and checkout-transition rules apply. Do not turn the
   read-only candidate list into an implicit bulk delete.

### Decisions to keep scope focused

- Do not add `gha release show` now. `gh release view` remains clearer until a
  proposed workflow can combine release data with actionable local or provider
  context beyond a stable single-release rendering.
- Do not add a `gha pr merge` wrapper. GHA already owns merge readiness;
  native `gh pr merge` remains the direct execution tool until a broader,
  safety-checked completion workflow is specified.
- Keep analytics, dashboard, plugins, offline synchronization, and additional
  providers as later phases. Each needs an evidence-backed workflow proposal
  that passes the Command Value Test before entering this delivery sequence.

## Future phases

### Analytics

- Engineering metrics, such as cycle time and throughput
- Hotspot identification
- Pull-request and branch risk assessment
- Ownership and knowledge-distribution analysis

### Enhanced experience

- TUI dashboard with real-time updates
- Plugin architecture for extensibility
- Multiple provider support, including GitLab and Azure DevOps
- Offline caching and background synchronization
