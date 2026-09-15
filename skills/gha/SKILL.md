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

The command never fetches. Treat `origin_state` as freshness information and
`permissions` or `can_push` values of `unavailable` as a blocker, not a
negative permission result. It refuses an already-tracked branch; use direct
`git push` for that straightforward update.

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
gha agent install --agent codex --dry-run
gha agent install --agent codex --confirm
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
```

`agent install` and `agent uninstall` modify the selected coding-agent
configuration only with `--confirm`; inspect their destination paths with
`--dry-run`. `pr prepare` is read-only; `pr create` repeats its preflight and
requires `--confirm` for the provider write. `branch create` changes only the
local checkout unless `--publish` is requested; branch publication, origin
rename, and origin deletion require `--confirm-origin`. `branch delete` always
requires an explicit `--local`, `--origin`, or both target. Its `--force` flag
overrides documented safety guardrails and should be used only after
independent verification.

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
```

`prs` is the bounded listing workflow, including `--queue`, `--assigned`, and
`--mine`; use `review <number>` for one decision-ready pull-request summary.
`releases` is a bounded listing of published releases that excludes drafts;
use direct `gh release` commands for provider-specific or unbounded release
operations. `release create-notes` is read-only and does not publish a GitHub
release. `gha release show` is a future roadmap item, not a command in the
current capability inventory.

If private API access is needed, inspect the current environment for
`GHA_GITHUB_TOKEN` before invoking GHA. For the full command surface, a
fine-grained token must be restricted to the target repositories and grant
`Contents: read`, `Pull requests: write`, `Checks: read`, and `Issues: read`.
`Pull requests: write` is required for `gha pr create --confirm`; Git branch
publication authenticates through the checkout remote instead. If the token is
absent, obtain one from an authenticated GitHub CLI only when available, then
pass it to that one GHA command using the current shell's native syntax. Do not
print, persist, overwrite, or commit either token. If neither token source is
available, report that authenticated GitHub inspection is unavailable.

## Validate GHA changes

When the target repository is GHA, rebuild after changing a command contract,
exercise the affected command in JSON and text modes, and run the relevant Go
tests. Keep unsupported optional signals as `unavailable`; do not make
automation infer facts from terminal text.

For branch writes, first run the exact command with `--dry-run`. Only perform
the real mutation when the current task explicitly authorizes it and the
command's required confirmation flags and target are present.
