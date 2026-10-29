# Command guide

Use `gha --help` for an overview and command-specific help for detailed flags:

```bash
gha pr checkout --help
gha pr retire --help
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
gha worktrees --help
gha worktree --help
gha worktree add --help
gha worktree remove --help
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
their local and/or remote target and support `--dry-run`. Worktree add and
remove execute by default after their safety preflight.

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
without parsing help or terminal text. Unset release linker fields fall back
individually to Go's embedded main-module version, VCS revision, and VCS time.
Explicit release values take precedence. A module version of `(devel)` is not a
release identity. If no corresponding metadata exists, the defaults remain
`dev`, `none`, and `unknown`. The optional `dirty` boolean reports the embedded
VCS modification state when known; absence means unknown. VCS time is the
commit time when used as the `date` fallback, rather than a measured build time.
Terminal output includes the same known dirty state.

```bash
gha version
gha version --format json
```

## Agent guidance installation

`gha agent list` reports harnesses recorded by `gha agent install`, the exact
instruction and skill destinations GHA manages, and whether each file is
present, modified, missing, or unavailable. An empty list means no harnesses are recorded by GHA; it
does not search the machine for manually configured or installed agents.
Structured output is available with `--format json` or `--format yaml`.

```bash
gha agent list
gha agent list --format json
```

`gha agent install` detects configured supported harnesses and copies the skill bundled with the installed GHA binary to
Codex, Claude Code, Pi, OpenCode, GitHub Copilot, Gemini CLI, Cursor, Hermes
Agent, or OpenClaw. It adds a
clearly marked GHA section to the selected global instruction file where the
harness supports a documented global file. Copilot uses its shared skill
directory without a separate global instructions file. Existing instruction
content is preserved; rerunning the command refreshes the skill without
duplicating the managed section. Installation records configured harnesses and their exact
destinations in the user's GHA config directory (`agent-installations.json`).
Uninstall uses those recorded destinations, removes only managed guidance and
an unchanged GHA skill, and retains paths still shared by another configured
harness. Existing unowned skills are never overwritten; a skill changed after
installation is preserved during update and uninstall. The ownership record is
removed after the last configured harness is uninstalled.

When `--agent` is omitted, the command configures detected harnesses based on
their setup directories; it does not require harness executables. If none are
found, it explains how to select one later. Use `--binary-only` to skip harness
setup, or comma-separated names such as `--agent pi,gemini,copilot` to select
explicitly. Pi, OpenCode, Copilot, and Gemini
share `~/.agents/skills/gha/SKILL.md` where their documented discovery supports
it. Cursor uses `~/.cursor/skills/gha/SKILL.md`, Hermes Agent uses
`$HERMES_HOME/skills/gha/SKILL.md` (default `~/.hermes/skills/gha/SKILL.md`),
and OpenClaw uses `$OPENCLAW_STATE_DIR/skills/gha/SKILL.md` (default
`~/.openclaw/skills/gha/SKILL.md`). Harness setup directories are detected
without requiring executables. Inspect destinations with `--dry-run`; otherwise
the selected operation runs.

```bash
# Preview a Codex installation without changing files.
gha agent install --agent codex --dry-run

# Install across several harnesses; shared skill paths are recorded once.
gha agent install --agent pi,opencode,copilot,gemini --dry-run

# Choose interactively, then install.
gha agent install --agent codex

# Preview removal of the managed guidance section and bundled skill.
gha agent uninstall --agent codex --dry-run

# Remove the selected managed guidance and skill.
gha agent uninstall --agent codex

# Preview and then refresh guidance for every harness recorded by install.
gha update --dry-run
gha update
gha update --dry-run --format json
```

`gha update` discovers the latest published GitHub release, then fetches that
release's bundled skill and managed instruction block. It updates only the
destinations recorded by `gha agent install`, preserving content outside the
managed block. Before writing, it checks the skill's declared GHA command
requirements against this executable's available capability inventory. If any
required command is unavailable or the skill has no compatibility manifest,
the update is rejected and existing files are preserved. Writing runs by
default; use `--dry-run` to preview. This command does not update the GHA
executable; update that through its installation method first. Structured
results report `compatibility_state` and any `missing_capabilities`.

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

`gha pr prepare` resolves the provider default base branch when omitted,
compares the selected refs, and reports existing open pull requests. It also
proposes a title and body from up to 50 local commit subjects between base and
head. JSON also includes the bounded changed file paths as source context and
their state; the generated body stays focused on commit descriptions. Pass
`--title` or `--body` to override either proposal. If refs are
unavailable locally, the draft state is `unavailable`; no fetch occurs. Review
the proposal before using it with `gha pr create`, which repeats provider
preflight and supports `--dry-run`.
It fails closed when comparison, existing-pull-request lookup, or provider
creation permission is unavailable, or when the provider reports that the
caller cannot push.

```bash
# Preview the complete request. Bare heads default to the current checkout;
# pass --head explicitly when preparing from another branch or a detached HEAD.
gha pr prepare --head feature/reviews --format json

# Return the exact creation plan without a provider write.
gha pr create --title "Improve reviews" --head feature/reviews --dry-run --format json

# Create only after the reviewed preflight is safe.
gha pr create --title "Improve reviews" --body "Adds decision-ready summaries." --head feature/reviews
```

## PR checkout, resume and retirement

Use `gha pr checkout <number> --worktree <path>` for an existing PR. It combines
fresh review readiness with local safety, creates an isolated checkout or
resumes it without resetting changes, and returns the actual PR push destination,
including forks. Use `gha pr retire <number> --worktree <path>` after the PR
closes. Retirement explains keep/remove decisions and preserves dirty,
unpublished or ambiguous work. It retains the local branch.

Both commands execute by default. Add `--dry-run` to preview without fetching
or writing. JSON/YAML expose the `PRWorktree` v1 contract, including per-effect
outcomes on execution failures; diagnostics remain on stderr. Explicit
`--repo` must match the source checkout's fetch origin.

For new development, create a branch/worktree first and use `gha pr create`
after publishing commits. Follow the [complete lifecycle guide](pr-worktree-lifecycle.md)
for examples, fork push guidance, safety rules and the structured contract.

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

Inspect the plan with `--dry-run`. Without that flag, GHA fetches and prunes
the configured `origin`:

```bash
gha branches --refresh-origin --dry-run --format json
gha branches --refresh-origin --format json
```

The refresh updates local remote-tracking refs and contacts `origin`; it fails
if `origin` is not configured. Add `--dry-run` to preview this operation; a
normal inventory remains read-only.

Pass `--path /path/to/checkout` to inspect another local checkout. This is a
local path, not an `owner/repo` identifier; GHA does not clone or fetch it.

`limit`, `local_truncated`, and `origin_truncated` make bounded results
explicit. A local branch has `divergence_state` of `available`, `not_tracked`,
or `unavailable`; an origin branch is `not_applicable`. Ahead/behind counts
exist only when divergence is available. A configured upstream missing from
local refs keeps `divergence_state: unavailable` and adds
`divergence_reason: upstream_gone` with a concise `divergence_message`. This
state describes cached local refs; it does not claim the branch is absent on
the provider. Branch inventory and individual inspection both report it,
without fetching or treating an unrelated similarly named local ref as the
upstream. Other Git failures remain unavailable without the gone-upstream
reason. Structured failures are written to
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
selects its start point. Add `--publish` to publish it and set
its upstream. `gha branch rename <old> <new>` is local by default; add
`--origin` to rename the remote branch too. `gha branch delete <name>` requires
`--local`, `--origin`, or both. Remote rename and deletion run after safety
checks; `--force` overrides those checks. If the selected local branch is
checked out and is not the default branch, GHA switches to the default branch
before deleting it. It refuses to delete the current default branch. Add
`--dry-run` to preview these operations.

`gha branch publish <name>` is the guarded workflow for an existing committed
local branch that has no upstream. It reports the explicit `origin/<name>`
target, cached-origin freshness, local upstream and divergence, and provider
push permission before it writes. It never fetches. Use `--dry-run` to inspect
the complete plan; omit `--dry-run` to push and set the upstream. An
explicitly denied permission blocks publication. If the provider cannot report
permission, GHA attempts the Git push and reports its actual result; Git's
authenticated transport is authoritative for that write. It refuses
already-tracked branches because direct `git push` is clearer for an ordinary
update.

For origin writes, a selected `--repo` or `GHA_REPOSITORY` must match the
checkout's GitHub `origin` push URL (or its origin URL when no push URL is
configured). GHA rejects a mismatch before changing local or
remote branches, including with `--force`. If no provider repository is
selected, provider-neutral Git writes such as `branch create --publish` remain
available for other origin types. Multiple origin push URLs cannot be checked
as one provider target.

```bash
# Inspect the target and push permission without changing Git or origin.
gha branch publish feature/api --dry-run --format json

# Publish after reviewing that plan.
gha branch publish feature/api
```

`branch create`, `branch rename`, and `branch delete` return `BranchMutation`
v1 in JSON or YAML, identifying local and origin as `planned`, `completed`,
`failed`, or `not_requested`. `planned` means not yet attempted, including steps
left unattempted after an earlier failure. `branch publish` returns
`BranchPublication` v1: its `target`,
`local`, `origin_branch`, `origin_state`, `permissions`, `can_push`, and
`publication` fields are the automation contract. A deletion that switches the
checkout reports the target in `checked_out`, including in `--dry-run` output.
The additive `checkout_state` field identifies whether that switch is `planned`,
`completed`, or `failed`; it is omitted when no switch is needed. `checked_out`
alone identifies the target, not proof that a switch completed.
Remote rename and deletion inspect push permission, default-branch status, and
protection first. They stop when a safety signal is unavailable or indicates a
default/protected branch; `--force` is the explicit override and should be used
only after independent verification.

Once execution starts, create, rename, and delete failures still return the
accumulated `BranchMutation` on stdout, exit nonzero, and send diagnostics to
stderr (`branch_mutation_failed` in JSON/YAML). Text output shows the same
effect states. Preflight failures produce diagnostics without a mutation result.
Completed effects are retained; GHA does not roll them back or retry. Inspect
the result and current refs before choosing a recovery action. A failed Git
transport can leave the remote outcome uncertain if its acknowledgement was
lost, so verify origin before retrying a failed origin effect.

For example, a rejected create-and-publish push leaves the new local branch:

```json
{
  "schema_version": "v1",
  "operation": "create",
  "name": "feature/api",
  "dry_run": false,
  "local": "completed",
  "origin": "failed"
}
```

Resolve the push rejection and inspect the remote before publishing that existing
branch; repeating `branch create` would collide with the completed local branch.
Origin rename uses an atomic push: creating the new remote ref and deleting the
old ref succeed together or neither ref changes. Servers without atomic push
support reject that step, leaving the local rename completed and origin failed.

Each branch dry run performs the same read-only preflight as execution before
returning a plan. Create checks the branch name, start commit, and local name
collision; `--publish` also reads origin refs to check for a collision and
validates the origin write target. Rename checks that the local source exists
and the destination is free; `--origin` also checks both origin refs and
provider safety. Delete verifies every selected ref, checks local worktree and
merge safety, and checks origin provider safety when deleting remotely. Origin
refs are read from Git's effective push URL without fetching into the checkout;
multiple push URLs are rejected because the write target is ambiguous. An
invalid or unverifiable operation returns an error without a `planned` result.

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

Use fresh GitHub evidence to recognise merged branches after squash or rebase:

```bash
gha branches cleanup --merged-prs --base main --limit 50 --format json
```

The result uses `rule: fresh_merged_pr_head_matches`. A candidate requires one
PR whose head and base repositories match the checkout's fetch origin, whose
base matches `--base`, and whose merged head commit equals the exact local tip.
An open or unmerged PR, multiple PR identities, a reused name, or a different
local tip retains the branch. A missing origin ref never proves a merge.
Fork or deleted head repositories remain excluded when identity cannot be
verified. Provider failures return local facts with unavailable evidence.

Each `merge_evidence` includes `state` (`fresh` or `unavailable`), `reason`,
`message`, `repository`, `observed_at`, and the selected `pull_request` when
available. For example, a matching branch has this additional evidence:

```json
{
  "reason": "merged_pr_head_matches",
  "merge_evidence": {
    "state": "fresh",
    "reason": "merged_pr_head_matches",
    "message": "fresh merged PR matches this repository, branch and exact local tip; merge strategy does not affect this evidence",
    "repository": {"owner": "acme", "name": "project"},
    "observed_at": "2026-10-28T12:00:00Z"
  }
}
```

This excerpt omits the full PR and local branch objects. Evidence describes
this inspection only; it does not authorise a later deletion. Repeat inspection
after local or provider state changes. The lookup reads one repository, at most
two PR identities per branch, and one PR detail for an unambiguous identity.
`--limit` bounds local branches and `truncated` reports omitted branches. Local
base selection still uses local Git refs and cached `origin/HEAD`; this command
does not fetch or refresh them.

Checked-out branches remain excluded with their worktree path. When their PR
head matches, `worktree_retirement` previews the existing PR-retirement checks:
`decision: remove` means the linked checkout passed, while `decision: keep`
includes reasons such as dirty, untracked, ignored, locked or unpublished work.
Primary and current checkouts remain protected. This preview never removes a
worktree or branch. Use the explicit execution workflow below to apply cleanup.

`branches cleanup --format json` returns `BranchCleanup` v1 with additive,
optional evidence fields. Without `--merged-prs`, it keeps the offline
reachability rule. Neither mode labels branches stale based on age, switches
branches, or deletes local or origin refs.

### Apply selected cleanup

Name every branch and select `--local`, `--origin`, or both. Review the plan first:

```bash
gha branches cleanup apply feature/reviews --base main --local --dry-run --format json
```

The `CleanupExecution` v1 result lists each branch's exact local and origin tips,
force selection, worktree path and planned or retained effects. Normal execution
requires fresh merged-PR evidence matching the local tip. Squash and rebase merges
need no force override. Omit `--dry-run` to execute after fresh revalidation.
Local deletion compares the exact inspected commit before deleting the ref.
Origin deletion checks permissions and branch policy, then uses an exact commit
lease. A missing origin branch is reported as `absent`.

To retire a clean linked checkout before deleting its branch, select its path:

```bash
gha branches cleanup apply feature/reviews --base main --local \
  --worktree ../gha-reviews --dry-run
```

Expect `worktree: planned` only when that branch owns the selected eligible
checkout. Otherwise the result retains the branch and explains the blocker.
Repeat `--worktree` for additional selected branches. Branches checked out
elsewhere remain retained, including for origin-only cleanup. Default/base
branches, primary/current checkouts, locks and unavailable worktrees stay protected.

### Force one named branch

Use `--force <branch>` only when you intend to discard that selected branch's
unmerged or unpublished commits. If you also select its linked worktree, force
permits discarding dirty, untracked and ignored files there:

```bash
gha branches cleanup apply feature/reviews --base main --local \
  --worktree ../gha-reviews --force feature/reviews --dry-run --format json
```

The preview reports `forced: true` for that name. Repeat `--force <branch>` for
each intended override. An unnamed or different branch never inherits force.
Force retains default/base and protected origin branches, primary/current
checkouts, locked worktrees and in-progress Git operations. It does not bypass
origin permissions, repository identity or commit leases.

After reviewing the dry-run, omit `--dry-run`. GHA prints the branch, exact tips,
selected targets and worktree path on stderr. It asks the agent or user to type
the exact branch name on stdin before any writes. Every forced branch needs its
own confirmation. Missing or mismatched input retains the entire plan. Dry-run
never prompts. Agents must obtain confirmation for the displayed destructive
plan before supplying that name; do not automatically answer a force prompt.

Execution retires the selected worktree, deletes the selected origin ref, then
deletes the selected local ref. Each branch reports `completed`, `retained` or
`failed` effects. A failed write exits nonzero, returns the observed effects on
stdout and puts diagnostics on stderr. Completed effects are not rolled back.
The recovery field keeps the original commit ID. Inspect the remaining branch
and origin refs before retrying. Recreate a removed checkout from its retained
branch, or recreate a deleted branch at the recorded commit promptly before Git
can prune unreachable objects. Git cannot recover discarded uncommitted files.

## Git worktrees

`gha worktrees --format json` returns the bounded `WorktreeInventory` v1
contract for every registered checkout in the selected repository. Each entry
includes its path, HEAD, branch or detached state, whether it is the main or
selected checkout, lock and prunable state, and working-tree status. Status is
`clean`, `dirty`, `unavailable`, or `not_applicable`. `total` and `truncated`
show whether `--limit` omitted any worktrees. `--path` may point at any checkout
in the same worktree set.

```bash
gha worktrees --format json
gha worktrees --path ../gha-feature --limit 10 --format json
```

`gha worktree add <path>` adds a checkout for an existing local branch, or
creates a new local branch with `--new-branch`. `--from` selects the new
branch's start point and defaults to `HEAD`. The command validates the parent
directory and target path, branch, and ref collisions before it reports a plan.
Use `--dry-run` to preview; otherwise the operation executes after preflight.
An existing branch must not already be checked out in another worktree.

```bash
# Review adding an existing branch, then create it.
gha worktree add ../gha-existing --branch feature/existing --dry-run --path .
gha worktree add ../gha-existing --branch feature/existing --path .

# Review a new branch starting at main, then create it.
gha worktree add ../gha-new --branch feature/new --new-branch --from main --dry-run --path .
gha worktree add ../gha-new --branch feature/new --new-branch --from main --path .
```

`gha worktree remove <path>` only targets a registered, clean, unlocked linked
worktree other than the checkout selected with `--path`. It refuses the main
worktree and retains Git's own removal checks; there is no force option. Use
`--dry-run` to preview; otherwise removal executes after preflight.
Local branch deletion detects branches checked out in sibling worktrees, and
`branches cleanup` excludes those branches with their worktree path.

```bash
gha worktree remove ../gha-existing --dry-run --path .
gha worktree remove ../gha-existing --path .
```

Add and remove return the `WorktreeMutation` v1 contract with the operation,
resolved target path, selected branch, whether this was a dry run, and a
`planned` or `completed` state.

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
Writing runs unless `--dry-run` is specified. If the push fails after the local tag is
created, the command reports the partial result so the local tag can be
reviewed before retrying.

```bash
gha tag publish build-2026.09 --commit HEAD --dry-run --format json
gha tag publish build-2026.09 --commit HEAD
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
gha release publish 1.2.3
# Select one workflow when several match the tag.
gha release publish 1.2.3 --workflow .github/workflows/publish.yaml --dry-run
```

The version creates `v1.2.3`; an explicit `v1.2.3` is also accepted. A dry
run returns `ReleasePublication` v1 with `ready`, `blockers`, the exact commit
and tag ref, CI checks, and planned local and origin effects. A blocked plan is
readable but cannot publish. The additive `origin_branch_state` reports
`available` or `absent` after a fresh origin query. An unpublished preparation
branch returns a blocked plan explaining the absent ref and default-branch
requirement; switch to the provider default branch and ensure the intended
commit is its origin tip. Origin transport/authentication errors and malformed
responses remain operational failures, never evidence that a ref is absent.
Publication rechecks the plan, creates an
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

### Release CI readiness

Readiness requires at least one successful validation check for the exact
release commit. Completed `skipped` and `neutral` checks are non-failing;
an entirely skipped or neutral suite cannot make a release ready. Failures,
cancellations, timeouts, action-required or stale results, pending checks, and
unknown conclusions block publication. All relevant checks are evaluated,
including optional checks; required checks must additionally be present and
match their configured provider application when one is specified.

GHA reads classic default-branch protection and active branch rulesets to
collect required check contexts. An explicit `protection.enabled: false` in
branch metadata means classic protection is absent, even when `protected: true`
reports active rulesets; GHA then reads the rulesets without requesting the
classic protection endpoint. Missing classic-protection metadata does not
establish absence, and denied or failed policy reads still block publication.
It also reads legacy commit statuses. For
GitHub Actions it selects the newest run per workflow on the provider default
branch at the exact release SHA, excludes pull-request events and other branches,
and uses that run's latest attempt and its job identities to select check
results. Superseded runs and previous attempts cannot supply successful
validation or stale blockers. A pending or failing selected workflow blocks
publication even when old successful jobs remain attached to its check suite.

Each provider inventory is bounded to 100 items. Truncation, denied access,
ambiguous protection responses, missing identities, unsupported required-workflow
rules, or an unavailable API make the affected signal `unavailable` and block
publication. Reading classic protection may require GitHub `Administration:
read` in addition to the normal Checks, Actions, Contents, and Metadata read
permissions. A failed policy lookup never means that no checks are required.
If a partial rerun omits an earlier job, that earlier attempt's result cannot
satisfy a required check; run complete validation before publishing.

`ReleasePublication` v1 retains `checks` and readable `blockers` and adds `ci`:

```json
{
  "ci": {
    "checks": {"state": "available"},
    "policy": {"state": "available"},
    "required_checks": [{"context": "Arch", "app_id": 15368}],
    "workflows": {"state": "available"},
    "runs": [{"id": 42, "workflow_id": 7, "check_suite_id": 90,
              "run_attempt": 2, "name": "Packages", "head_sha": "abc123",
              "head_branch": "main", "event": "workflow_dispatch",
              "status": "in_progress", "conclusion": ""}],
    "blockers": [{"name": "Arch", "status": "in_progress", "conclusion": ""}]
  }
}
```

This is a result fragment; the complete plan includes the existing tag,
checkout, permission, and workflow fields. `ci.blockers` gives agents check
names, statuses, and conclusions without parsing prose. Signal messages explain
unavailable evidence. Terminal output shows the same policy states, required
checks, workflow run IDs and attempts, checks, and blockers.

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
