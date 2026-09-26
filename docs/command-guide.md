# Command guide

Use `gha --help` for an overview and command-specific help for detailed flags:

```bash
gha prs --help
gha review --help
gha releases --help
gha release --help
gha release create-notes --help
gha release publish --help
gha analyze --help
gha branches --help
gha branch --help
gha branch publish --help
gha dashboard --help
gha capabilities --help
gha version --help
```

## Agent-first contract

Every available command is safe to inspect by default and exposes accurate
Cobra help. Data commands provide a versioned JSON contract on stdout; text is
the human rendering of the same underlying result. Diagnostics and structured
errors are written to stderr, so an agent never has to parse a mixed stream.

Listings are bounded and say when results were truncated. Signals that GHA
cannot establish are `unavailable`, not guesses. Mutating commands identify
their local and/or remote target, require explicit confirmation before writes,
and support `--dry-run` where they can change repository or provider state.

## Capabilities

`gha capabilities --format json` is the versioned `Capabilities` v1 inventory
of every installed command. It reports `available` commands separately from
features that are intentionally `unavailable`; agents should use it before
planning work from this CLI. The current `dashboard` command is unavailable
and exits non-zero rather than pretending to launch a TUI.

## Installed build identity

`gha version --format json` returns the versioned `VersionInfo` v1 contract for
the installed binary. Release artifacts include the release version, source
commit, and build time so support tooling can identify the exact executable
without parsing help or terminal text. Development builds intentionally report
`dev`, `none`, and `unknown` for those fields when release linker metadata is
not present.

```bash
gha version
gha version --format json
```

## Agent guidance installation

`gha agent list` reports harnesses recorded by `gha agent install`, the exact
instruction and skill destinations GHA manages, and whether each file is
present, missing, or unavailable. An empty list means no harnesses are recorded by GHA; it
does not search the machine for manually configured or installed agents.
Structured output is available with `--format json` or `--format yaml`.

```bash
gha agent list
gha agent list --format json
```

`gha agent install` copies the skill bundled with the installed GHA binary to
Codex, Claude Code, Pi, OpenCode, GitHub Copilot, or Gemini CLI. It adds a
clearly marked GHA section to the selected global instruction file where the
harness supports a documented global file. Copilot uses its shared skill
directory without a separate global instructions file. Existing instruction
content is preserved; rerunning the command refreshes the skill without
duplicating the managed section. Installation records configured harnesses and their exact
destinations in the user's GHA config directory (`agent-installations.json`).
Uninstall uses those recorded destinations, removes only managed guidance and
the bundled GHA skill, and retains paths still shared by another configured
harness. The ownership record is removed after the last configured harness is
uninstalled.

The command prompts for one or more agents when `--agent` is omitted. Use
comma-separated names such as `--agent pi,gemini,copilot`. Pi, OpenCode, Copilot, and Gemini
share `~/.agents/skills/gha/SKILL.md` where their documented discovery supports
it. Inspect destinations with `--dry-run`, then pass `--confirm` to install.

```bash
# Preview a Codex installation without changing files.
gha agent install --agent codex --dry-run

# Install across several harnesses; shared skill paths are recorded once.
gha agent install --agent pi,opencode,copilot,gemini --dry-run

# Choose interactively, then install.
gha agent install --confirm

# Preview removal of the managed guidance section and bundled skill.
gha agent uninstall --agent codex --dry-run

# Confirm the removal after reviewing the target.
gha agent uninstall --agent codex --confirm
```

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

## Pull-request preparation and creation

`gha pr prepare` is the guarded alternative for a pull request that needs a
reviewable plan. It resolves the provider default base branch when omitted,
compares the selected refs, and reports existing open pull requests. It never
writes. `gha pr create` repeats that preflight, supports `--dry-run`, and only
creates the provider pull request with `--confirm`.
It fails closed when comparison, existing-pull-request lookup, or provider
creation permission is unavailable, or when the provider reports that the
caller cannot push.

```bash
# Preview the complete request. Bare heads default to the current checkout;
# pass --head explicitly when preparing from another branch or a detached HEAD.
gha pr prepare --title "Improve reviews" --head feature/reviews

# Return the exact creation plan without a provider write.
gha pr create --title "Improve reviews" --head feature/reviews --dry-run --format json

# Create only after the reviewed preflight is safe.
gha pr create --title "Improve reviews" --body "Adds decision-ready summaries." --head feature/reviews --confirm
```

## Local repository analysis

`gha analyze --format json` returns the versioned `RepositoryAnalysis` v1
schema. It combines a local worktree summary, current `HEAD`, bounded recent
commits, Git object-database storage, and the largest blobs tracked by `HEAD`.
It never fetches, contacts a remote, or changes Git state.

```bash
# Analyze the current checkout.
gha analyze

# Inspect another local checkout with bounded lists for automation.
gha analyze --path ../other-checkout --limit 20 --format json
```

`limit` applies independently to `worktree.changes`, `recent_commits`, and
`largest_files`; each has a matching `*_truncated` field. Worktree counts still
cover all changes when its path list is truncated. `storage` is Git's local
object-database estimate in KiB, not the size of a working tree or any remote.
`largest_files` describes committed `HEAD` blobs, so it is `unavailable` when
`HEAD` is unborn; it does not inspect uncommitted file contents.

## Branch inventory

`gha branches --format json` returns the versioned `BranchInventory` v1 schema.
By default it reads only local Git state: `origin_branches` are cached
remote-tracking refs and GHA never fetches implicitly. `origin_state` is
`cached`, `absent`, `unconfigured_cached`, or `refreshed`; agents must treat
`cached` data as potentially stale. `origin_refresh.state` is
`not_requested`, `planned`, or `completed`, so automation can distinguish a
cached view, a reviewed refresh plan, and refs refreshed during this invocation.

To refresh intentionally, first inspect the plan, then explicitly confirm the
fetch and prune of the configured `origin`:

```bash
gha branches --refresh-origin --dry-run --format json
gha branches --refresh-origin --confirm-origin --format json
```

The refresh updates local remote-tracking refs and contacts `origin`; it fails
if `origin` is not configured. `--refresh-origin` requires either `--dry-run`
or `--confirm-origin` so a normal inventory remains read-only.

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

## Branch writes

`gha branch create <name>` creates only a local branch by default; `--from`
selects its start point. Add `--publish --confirm-origin` to publish it and set
its upstream. `gha branch rename <old> <new>` is local by default; add
`--origin --confirm-origin` to rename the remote branch too. `gha branch delete
<name>` requires `--local`, `--origin`, or both; `--origin` also needs
`--confirm-origin`. If the selected local branch is checked out and is not the
default branch, GHA switches to the default branch before deleting it. It
refuses to delete the current default branch.

`gha branch publish <name>` is the guarded workflow for an existing committed
local branch that has no upstream. It reports the explicit `origin/<name>`
target, cached-origin freshness, local upstream and divergence, and provider
push permission before it writes. It never fetches. Use `--dry-run` to inspect
the complete plan; use `--confirm-origin` to push and set the upstream. It
refuses already-tracked branches because direct `git push` is clearer for an
ordinary update.

```bash
# Inspect the target and push permission without changing Git or origin.
gha branch publish feature/api --dry-run --format json

# Publish only after reviewing that plan.
gha branch publish feature/api --confirm-origin
```

`branch create`, `branch rename`, and `branch delete` return `BranchMutation`
v1 in JSON or YAML, identifying local and origin as `planned`, `completed`, or
`not_requested`. `branch publish` returns `BranchPublication` v1: its `target`,
`local`, `origin_branch`, `origin_state`, `permissions`, `can_push`, and
`publication` fields are the automation contract. A deletion that switches the
checkout reports the target in `checked_out`, including in `--dry-run` output.
Remote rename and deletion inspect push permission, default-branch status, and
protection first. They stop when a safety signal is unavailable or indicates a
default/protected branch; `--force` is the explicit override and should be used
only after independent verification.

## Branch cleanup candidates

`gha branches cleanup` is a read-only review, not a deletion command. It
examines a bounded set of local branches and proposes a branch only when its
tip is reachable from the selected base branch. By default the base is the
cached `origin/HEAD` branch; use `--base` when that cached default is absent or
when a different local integration branch is the intentional comparison point.
Every reviewed branch appears in either `candidates` or `excluded`, with a
machine-readable reason. The result is incomplete when `truncated` is true.

```bash
# Review candidates relative to the repository's cached default branch.
gha branches cleanup --format json

# Select the exact local integration branch instead.
gha branches cleanup --base main --limit 50
```

`branches cleanup --format json` returns `BranchCleanup` v1. It does not infer
provider safety facts, label a branch stale based on age, switch branches, or
delete local or origin refs.

## Releases

`gha releases` is a read-only, bounded listing of published releases. It
excludes drafts and returns the versioned `ReleaseList` v1 schema in JSON or
YAML, including the selected repository, `limit`, and `truncated` state. It
does not create, edit, or delete a GitHub release.

```bash
# List published releases for the current repository.
gha releases --limit 10

# Select a provider repository explicitly and retain a machine-readable bound.
gha releases --repo owner/repo --limit 25 --format json

# Use another checkout's origin remote to select the repository.
gha releases --path ../other-checkout
```

Use direct `gh release` commands for unbounded or provider-specific release
operations. `gha release show <tag>` remains a separate future workflow.

## General tag publication

`gha tag publish <name>` creates a tag at `--commit` (default `HEAD`) and
pushes only that exact ref to `origin`. The dry run resolves the commit and
checks local and origin tag collisions before reporting both planned effects.
Writing requires `--confirm-origin`. If the push fails after the local tag is
created, the command reports the partial result so the local tag can be
reviewed before retrying.

```bash
gha tag publish build-2026.09 --commit HEAD --dry-run --format json
gha tag publish build-2026.09 --commit HEAD --confirm-origin
```

The `TagPublication` v1 result includes the selected commit, blockers, and
local and origin states (`planned`, `completed`, or `failed`). This command
does not apply the SemVer, CI, or workflow checks owned by
`gha release publish`.

## Guarded release publication

`gha release publish <version>` plans an annotated SemVer tag at the checked-out
default-branch commit. It compares that commit with the current origin branch
tip, checks the clean worktree, local and provider tags, provider push
permission, CI check runs, and a matching tag push trigger in committed
`.github/workflows/*.yml` or `.yaml` files. It never fetches or updates a branch.
When the selected workflow declares `GHA_RELEASE_NOTES_DIR`, it also checks
that `HEAD` contains a nonempty, finished `<directory>/<tag>.md` file. The
`release_notes` result reports its state and path; missing notes block the tag.
GHA does not require this convention in other repositories.

```bash
gha release publish 1.2.3 --dry-run --format json
gha release publish 1.2.3 --confirm-origin
# Select one workflow when several match the tag.
gha release publish 1.2.3 --workflow .github/workflows/publish.yaml --dry-run
```

The version creates `v1.2.3`; an explicit `v1.2.3` is also accepted. A dry
run returns `ReleasePublication` v1 with `ready`, `blockers`, the exact commit
and tag ref, CI checks, and planned local and origin effects. A blocked plan is
readable but cannot publish. Publication rechecks the plan, creates an
annotated local tag, and pushes only that tag to origin. The result reports
each completed or failed effect. Workflow observation reports `triggered`,
`completed`, `failed`, or `unavailable`; a successful tag push alone does not
prove that the GitHub Release exists. Use `gha releases` or the Actions run to
check later. `--path` selects a checkout; `--repo` may select provider facts
only when it matches that checkout's origin.
The command checks for a tag trigger and observes the resulting Actions run;
it never runs a language-specific build or release tool. CI and the release
workflow remain responsible for those steps.
For this repository, [prepare the reviewed release notes](development.md#preparing-a-release)
before running the publication dry run.

## Release notes

`gha release create-notes --since ...` is read-only: it does not create a
GitHub release or change a repository. Give it the inclusive start of the
release window; it derives notes from merged pull requests and writes them to
standard output.

```bash
# Generate terminal-readable release notes for the current repository.
# A date or timezone-less ISO datetime means midnight/local time on this machine.
gha release create-notes --since 2026-09-01

# An explicit offset remains authoritative.
gha release create-notes --repo owner/repo --since 2026-09-01T00:00:00Z --format json

# Use another checkout's origin remote to select the repository.
gha release create-notes --path ../other-checkout --since 2026-09-01
```

`gha release create-notes --format json` returns the versioned `ReleaseNotes`
v1 schema.
Its `limit` and `truncated` fields make bounded release windows explicit.
`--since` accepts an RFC 3339 timestamp, an ISO datetime without a timezone, or
an ISO date. When no timezone is supplied, GHA uses the current machine
timezone; for example, in UTC+2, `2025-09-01` means
`2025-09-01T00:00:00+02:00` (the equivalent API instant is
`2025-08-31T22:00:00Z`).
