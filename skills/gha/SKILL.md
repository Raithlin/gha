---
name: gha
description: Use GHA to inspect GitHub and local-repository workflows when its installed command can provide a safer, structured answer.
---

# GHA

Use this skill in any repository when a GHA command can answer a GitHub or
local-repository question with a bounded, structured result. Prefer GHA's
domain output over ad-hoc Git parsing or raw provider output when its available
workflow covers the question. It complements rather than replaces `git` and
`gh`; use those tools directly when GHA does not add context, safety, or a
stable contract.

## Find the available GHA command

Use the installed `gha` command from the target repository. Confirm its actual
surface before relying on it:

```text
gha capabilities --format json
```

If you are developing GHA itself, rebuild after Go-source changes using the
project's documented build command, then invoke the resulting local binary
with the syntax appropriate to the current platform and shell.

Treat `capabilities` as the executable's source of truth. Do not assume a
command or feature is available because it appears in a roadmap or another
installation. Use JSON for agent decisions and assertions; use text output
when checking the human terminal experience.

`gha version --format json` identifies the installed build. `gha dashboard` is
intentionally unavailable in current builds; do not attempt to launch it.
`help` and `completion` are standard Cobra assistance rather than GHA workflow
contracts.

## Inspect before implementing

Choose the narrowest read-only workflow that answers the question. Useful
local checks include:

```text
gha analyze --format json
gha branches --format json
gha branch show main --format json
```

Use `--path` to inspect another checkout rather than changing directories or
cloning it. Local branch and origin data can be cached or unavailable; retain
the state GHA reports instead of treating missing information as a negative
result.

When fresh origin-tracking refs are material to the decision, first obtain the
explicit refresh plan:

```text
gha branches --refresh-origin --dry-run --format json
```

This reports `origin_refresh.state: planned` and does not contact origin. Only
when the refresh is explicitly authorized, run:

```text
gha branches --refresh-origin --confirm-origin --format json
```

It runs `git fetch --prune origin`, changing only cached remote-tracking refs;
the result reports `origin_state: refreshed` and
`origin_refresh.state: completed`. Do not use either flag for ordinary branch
inventory: `gha branches` never fetches implicitly.

## Publish an existing local branch

Use `gha branch publish <name>` when an agent needs a reviewable publication
plan for an existing, committed local branch that does not yet have an
upstream. It adds value beyond a direct push by reporting the explicit
`origin/<name>` target, cached-origin state, local upstream and divergence,
and provider push permission in the `BranchPublication` result.

First inspect the exact plan in structured output:

```text
gha branch publish feature/example --dry-run --format json
```

The command never fetches. Treat `origin_state` as freshness information.
An unavailable `permissions` or `can_push` signal is unknown, not a denial;
when publishing is attempted, Git's authenticated push result determines
whether the operation succeeds. An explicitly denied permission still blocks.
It refuses an already-tracked branch; use direct `git push` for that
straightforward update.

Only when the requested publication is explicitly authorized and the reviewed
plan is safe, make the remote write with:

```text
gha branch publish feature/example --confirm-origin
```

## Other guarded workflows

Use the command that owns the requested workflow; inspect a dry run before any
command that supports one, and make a write only when it is explicitly
authorized.

```text
gha agent list --format json
gha agent install --agent codex --dry-run
gha agent install --agent codex --confirm
gha agent install --agent pi,opencode,copilot,gemini --dry-run
gha agent install --agent pi,opencode,copilot,gemini --confirm
gha agent uninstall --agent codex --dry-run
gha agent uninstall --agent codex --confirm

gha pr prepare --title "Improve reviews" --head feature/reviews --format json
gha pr create --title "Improve reviews" --head feature/reviews --dry-run --format json
gha pr create --title "Improve reviews" --head feature/reviews --confirm

gha branch create feature/example --dry-run --format json
gha branch create feature/example --publish --confirm-origin
gha branch rename old-name new-name --dry-run --format json
gha branch rename old-name new-name --origin --confirm-origin
gha branch delete feature/example --local --dry-run --format json
gha branch delete feature/example --local
gha tag publish build-2026.09 --commit HEAD --dry-run --format json
gha tag publish build-2026.09 --commit HEAD --confirm-origin
```

`agent list` reports harnesses recorded by GHA and whether their managed
instruction and skill files are present, missing, or unavailable. An empty
result does not mean that no other agent software is installed. `agent install`
and `agent uninstall` accept comma-separated harnesses; Pi, OpenCode, Copilot,
and Gemini share `~/.agents/skills/gha/SKILL.md`. Copilot has no separate
global instructions file in this workflow. These commands modify the selected
coding-agent configuration only with `--confirm`; inspect their destinations
with `--dry-run`. `pr prepare` is read-only; `pr create` repeats its preflight and
requires `--confirm` for the provider write. `branch create` changes only the
local checkout unless `--publish` is requested; branch publication, origin
rename, and origin deletion require `--confirm-origin`. `branch delete` always
requires an explicit `--local`, `--origin`, or both target. Its `--force` flag
overrides documented safety guardrails and should be used only after
independent verification.

`tag publish` reports local and origin tag collisions and resolves its commit
before writing; inspect the dry run first. It creates the local tag, then
pushes only that tag ref. If the push fails, inspect the reported partial state
before retrying.

`gha branches cleanup --format json` is a read-only, bounded explanation of
local cleanup candidates. It never deletes branches, fetches, switches the
checkout, or infers provider safety. Use `--base` to select the local base when
the cached `origin/HEAD` default is unsuitable.

For GitHub-backed inspection, use the command that owns the workflow:

```text
gha prs --format json
gha review 123 --format json
gha releases --format json
gha release create-notes --since 2026-09-01 --format json
gha release publish 1.2.3 --dry-run --format json
```

`prs` is the bounded listing workflow, including `--queue`, `--assigned`, and
`--mine`; use `review <number>` for one decision-ready pull-request summary.
`releases` is a bounded listing of published releases that excludes drafts;
use direct `gh release` commands for provider-specific or unbounded release
operations. `release create-notes` is read-only and does not publish a GitHub
release. `gha release show` is a future roadmap item, not a command in the
current capability inventory.

`gha release publish <version>` is the guarded tag workflow. It requires a
clean checkout at the provider default-branch tip, passing CI checks, no local
or origin tag collision, provider push permission, and a matching committed
GitHub Actions tag trigger. Use `--workflow` when multiple workflows match the
tag. If that workflow declares `GHA_RELEASE_NOTES_DIR`, GHA also requires a
committed, nonempty `<directory>/<tag>.md` file without `REPLACE_ME` template
tokens. For this repository, use `docs/releases/TEMPLATE.md` to prepare the
versioned notes before tagging. Review `--dry-run` output first;
use `--confirm-origin` only for the authorized publication. Its result
separates the local tag, origin push, and release workflow observation.
`triggered` is not the same as a completed GitHub Release; `unavailable` means
the Actions run has not been verified.

GHA uses `GHA_GITHUB_TOKEN` when set. Otherwise, it automatically tries
`gh auth token` when the GitHub CLI is installed; with neither credential,
public API requests remain unauthenticated. For private API access and the full
command surface, a fine-grained token must be restricted to the target
repositories and grant `Contents: read`, `Pull requests: write`, `Checks: read`,
`Actions: read`, and `Issues: read`. `Pull requests: write` is required for
`gha pr create --confirm`; Git branch publication authenticates through the
checkout remote instead. Never print, persist, overwrite, or commit tokens. If
the API request fails for lack of access, report that authentication or
repository permission is required.

## Validate GHA changes

When the target repository is GHA, rebuild after changing a command contract,
exercise the affected command in JSON and text modes, and run the relevant Go
tests. Keep unsupported optional signals as `unavailable`; do not make
automation infer facts from terminal text.

For branch writes, first run the exact command with `--dry-run`. Only perform
the real mutation when the current task explicitly authorizes it and the
command's required confirmation flags and target are present.
