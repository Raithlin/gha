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

For GitHub-backed inspection, use the command that owns the workflow:

```text
gha prs --format json
gha review 123 --format json
gha release --since 2026-09-01 --format json
```

If private API access is needed, inspect the current environment for
`GHA_GITHUB_TOKEN` before invoking GHA. If it exists, use it unchanged. If it
is absent, obtain a token from an authenticated GitHub CLI only when available,
then pass it to that one GHA command using the current shell's native syntax.
Do not print, persist, overwrite, or commit either token. If neither token
source is available, report that authenticated GitHub inspection is unavailable.

## Validate GHA changes

When the target repository is GHA, rebuild after changing a command contract,
exercise the affected command in JSON and text modes, and run the relevant Go
tests. Keep unsupported optional signals as `unavailable`; do not make
automation infer facts from terminal text.

For branch writes, first run the exact command with `--dry-run`. Only perform
the real mutation when the current task explicitly authorizes it and the
command's required confirmation flags and target are present.
