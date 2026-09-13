# Command guide

Use `gha --help` for an overview and command-specific help for detailed flags:

```bash
gha prs --help
gha review --help
gha release --help
gha branches --help
gha branch --help
gha dashboard --help
gha capabilities --help
```

## Agent-first contract

Every available command is safe to inspect by default and exposes accurate
Cobra help. Data commands provide a versioned JSON contract on stdout; text is
the human rendering of the same underlying result. Diagnostics and structured
errors are written to stderr, so an agent never has to parse a mixed stream.

Listings are bounded and say when results were truncated. Signals that GHA
cannot establish are `unavailable`, not guesses. Future commands that mutate
local or remote state will require an explicit target and confirmation, and
will support `--dry-run`.

## Capabilities

`gha capabilities --format json` is the versioned `Capabilities` v1 inventory
of every installed command. It reports `available` commands separately from
features that are intentionally `unavailable`; agents should use it before
planning work from this CLI. The current `dashboard` command is unavailable
and exits non-zero rather than pretending to launch a TUI.

## Pull-request review

`gha review <number> --format json` returns the versioned `ReviewSummary` v1
schema. It is the supported automation surface for single-PR inspection; text
output is intended for people.

```bash
# Review a specific PR
gha review 123

# Inspect a repository outside the current directory
gha review 123 --repo owner/repo

# Resolve the repository from another local checkout's origin remote
gha review 123 --path ../other-checkout

# Change output format
gha review 123 --format json
```

```json
{
  "schema_version": "v1",
  "pull_request": { "number": 123, "title": "Improve automation" },
  "reviews": [],
  "readiness": {
    "mergeable": true,
    "ci_status": "success",
    "review_threads_state": "none",
    "approved_by": [],
    "changes_requested_by": [],
    "pending_reviewers": []
  },
  "risk_signals": [],
  "recommended_actions": []
}
```

`ci_status` is derived from GitHub check runs for the PR head commit and is
one of `success`, `pending`, `failure`, or `none`. `review_threads_state` is
one of `none`, `resolved`, or `unresolved`. Either signal is `unavailable`
when GHA cannot retrieve it. Fields in this schema will be changed additively
within v1.

## Pull-request listings

`gha prs --format json` returns the versioned `PullRequestList` v1 schema. It
includes the selected `repository`, requested `limit`, and `truncated` so
agents never have to infer whether a bounded result may omit matching PRs. All
GitHub-backed commands emit `CommandError` v1 diagnostics to stderr for JSON
and YAML requests; successful data remains exclusively on stdout.

```bash
# List open pull requests
gha prs

# List work relevant to you
gha prs --queue
gha prs --assigned
gha prs --mine

# Filter repository pull requests
gha prs --state all --author octocat --base main
gha prs --reviewer @me --format json

# Restrict a query to PRs updated at or after this RFC 3339 instant
gha prs --since 2026-09-01T00:00:00Z --limit 20 --format json

# Resolve the target repository from another local checkout
gha prs --path ../other-checkout --format json
```

## Branch inventory

`gha branches --format json` returns the versioned `BranchInventory` v1 schema.
It reads only local Git state: `origin_branches` are cached remote-tracking refs
and GHA never fetches implicitly. `origin_state` is `cached`, `absent`, or
`unconfigured_cached`; agents must treat `cached` data as potentially stale.

Pass `--path /path/to/checkout` to inspect another local checkout. This is a
local path, not an `owner/repo` identifier; GHA does not clone or fetch it.

`limit`, `local_truncated`, and `origin_truncated` make bounded results
explicit. A local branch has `divergence_state` of `available`, `not_tracked`,
or `unavailable`; an origin branch is `not_applicable`. Ahead/behind counts
exist only when divergence is available. Structured failures are written to
stderr as `CommandError` v1 with a stable code and message, leaving stdout
reserved for successful data.

## Branch safety inspection

`gha branch show <name> --format json` returns the versioned
`BranchInspection` v1 schema. It reads the selected local and cached-origin
branch without fetching or changing state. When GHA can resolve a GitHub
repository, it also reports open pull requests, protection, caller push
permission, default-branch status, and mergeability. Each provider signal is
independently marked `available`, `unavailable`, or `not_applicable`; missing
data must never be interpreted as a negative safety result.

Text output identifies the repository and provider, shows the provider
`checked_at` time, and summarizes unavailable provider data once. Use JSON or
YAML when an automation client needs the detailed diagnostic message for an
individual signal.

Use `--repo owner/repo` when the checkout's origin is not a GitHub remote, and
`--path /path/to/checkout` to inspect another local checkout.

## Release notes

The current `gha release --since ...` command is read-only: it does not create
a GitHub release or change a repository. Give it the inclusive start of the
release window; it derives notes from merged pull requests and writes them to
standard output.

```bash
# Generate terminal-readable release notes for the current repository.
# A date or timezone-less ISO datetime means midnight/local time on this machine.
gha release --since 2026-09-01

# An explicit offset remains authoritative.
gha release --repo owner/repo --since 2026-09-01T00:00:00Z --format json

# Use another checkout's origin remote to select the repository.
gha release --path ../other-checkout --since 2026-09-01
```

`gha release --format json` returns the versioned `ReleaseNotes` v1 schema.
Its `limit` and `truncated` fields make bounded release windows explicit.
`--since` accepts an RFC 3339 timestamp, an ISO datetime without a timezone, or
an ISO date. When no timezone is supplied, GHA uses the current machine
timezone; for example, in UTC+2, `2025-09-01` means
`2025-09-01T00:00:00+02:00` (the equivalent API instant is
`2025-08-31T22:00:00Z`).
