# Pull-request worktree lifecycle

Use GHA to connect isolated local work with a pull request. Choose the path
that matches your starting point. Every command below executes by default;
add `--dry-run` to preview a mutation.

## Why this belongs in GHA

`gh pr checkout --worktree` already checks out a PR in a linked worktree.
Checkout creation alone would not justify another command. GHA adds one
versioned result that combines PR review readiness, the exact local checkout,
publication comparison, safe resume and a conservative retirement decision.
An agent can decide what to do next without parsing several terminal outputs.
Git still owns branches and worktrees; GitHub supplies PR and review facts.

The workflow preserves the selected checkout and never resets an existing
branch. Retirement uses fresh PR state and local evidence together. A merged
PR alone does not prove that a checkout has no unpublished work.

## Start new development and create a PR

Create an isolated branch before a PR exists:

```bash
gha worktree add ../gha-feature --branch feature/improve-reviews --new-branch --from master
```

The result reports a completed worktree addition at the resolved absolute
path. Your original checkout stays on its current branch. Make changes in
`../gha-feature`, run the relevant tests, and commit them. Preview publication:

```bash
gha branch publish feature/improve-reviews --path ../gha-feature --dry-run --format json
```

The result identifies the origin branch and publication checks. When the plan
is suitable, omit `--dry-run` to publish. Then create the PR:

```bash
gha pr create --path ../gha-feature --title "Improve reviews" --dry-run --format json
```

The result reports the selected head and base, comparison and creation plan.
Omit `--dry-run` to create it. Keep the returned PR number and the worktree path.
There is no PR to pass to `pr checkout` until creation succeeds.

## Check out an existing PR

Inspect the plan, then execute it:

```bash
gha pr checkout 42 --worktree ../gha-pr-42 --dry-run --format json
gha pr checkout 42 --worktree ../gha-pr-42 --format json
```

The preview reports `decision: create`, `state: planned` and planned fetch,
branch and worktree effects when a local branch is needed. It contacts the
provider for PR/review facts, but never fetches or writes. Execution fetches
only `refs/pull/42/head` into `refs/gha/pull/42/head`, checks its SHA against
the provider head, and creates the isolated checkout. It does not push,
create a PR, change the current branch, initialize submodules or run setup.

For same-repository PRs, the default local branch is the PR head branch. For
forks, it is `pr/42`. Use `--branch review/42` to select another local name.
GHA refuses existing paths, branch namespace conflicts and branches checked
out at another path. A local branch can be reused when its history identifies
the PR; an unassociated branch behind or diverged from the PR is ambiguous.

`--path` selects the source local checkout. Relative `--worktree` paths resolve
from that checkout's root. `--repo` must match its single fetch origin; it
cannot redirect provider checks to another repository. Ambient repository
configuration does not override this local binding. No clone or implicit
origin refresh occurs. The private PR ref is refreshed only when checkout
needs to create a local branch.

## Resume and update the existing PR

Run the same checkout command with the existing path:

```bash
gha pr checkout 42 --worktree ../gha-pr-42 --format json
```

A matching checkout returns `decision: resume` and `state: resumed`. It reports
local changes and commits ahead/behind the fresh PR head. It never fetches,
resets, rebases, switches or overwrites that checkout. On the first successful
resume, it records a repository/PR association in local branch metadata so
later resumes can identify the branch even after published history changes. A prior GHA association
allows it to report a behind or diverged checkout without modifying it.
Inspect those differences before synchronizing. Unavailable checkout status,
locks, conflicting associations and unrelated branches stop checkout.

Commit changes in the returned path. Use `push.repository`, `push.url` and
`push.refspec` to update the existing PR. For a fork, a result might contain:

```json
{
  "push": {
    "state": "available",
    "repository": {"owner": "contributor", "name": "renamed-fork"},
    "url": "https://github.com/contributor/renamed-fork.git",
    "refspec": "HEAD:refs/heads/feature"
  }
}
```

Push explicitly after reviewing that destination:

```bash
git -C ../gha-pr-42 push https://github.com/contributor/renamed-fork.git HEAD:refs/heads/feature
```

Git reports the updated fork branch, or an authentication/permission failure.
The contributor's fork can have a different repository name. GHA does not
configure a remote, assume fork push permission, set an upstream or push to the
base repository. A missing/deleted head repository yields
`push.state: unavailable`; inspect the destination before pushing. Do not run
`pr create` again to update this PR.

## Retire a finished checkout

Preview retirement from another checkout:

```bash
gha pr retire 42 --worktree ../gha-pr-42 --dry-run --format json
```

The result explains `decision: keep` or `decision: remove`. An open PR stays.
A closed PR, including a merged PR, is eligible only when the exact registered
linked checkout is clean and no local commits are absent from the provider's
PR head. Comparison uses the PR head, including after squash merges, rather
than assuming that its commits must be ancestors of the base branch.

GHA preserves main/current checkouts, detached heads, locks, prunable entries,
untracked or ignored files, unavailable status, shallow/missing history,
conflicting PR associations, unpublished/divergent commits, in-progress Git
operations and index flags that can hide changes. It does not
fetch to repair uncertain history. A conservative keep result succeeds with
exit code 0 and explains its reasons. Inspect and resolve those reasons before
trying again.

When the preview says remove, omit `--dry-run` to execute. GHA rechecks provider
identity/state and local safety, uses ordinary `git worktree remove` without
force, and retains the local branch and cached refs. The result reports
`state: completed` and `worktree_state: completed`. Branch deletion is a
separate explicit workflow.

## Structured result and recovery

Both commands expose `PRWorktree` schema `v1` in JSON/YAML and render the same
facts as text. The result includes:

| Field | Meaning |
| --- | --- |
| `repository`, `pull_request` | Selected base repository and fresh PR identity/head/state |
| `branch`, `path`, `worktree` | Local branch, absolute target and observed checkout facts |
| `review_state`, `review` | Review availability, CI, merge readiness, reviewers and threads |
| `comparison` | Local commits ahead/behind the published PR head, or `unavailable` |
| `decision` | `create`, `resume`, `keep` or `remove` |
| `state` | `planned`, `completed`, `resumed`, `kept` or `failed` |
| `fetch`, `branch_state`, `worktree_state` | Per-effect plan/outcome; branch retirement always retains the branch |
| `association_state` | PR association metadata planned/completed/not needed/failed; retries can repair a failed association |
| `provider_state` | `fresh`; local comparison does not treat cached origin refs as fresh |
| `push` | Explicit destination for updating this PR, or `unavailable` |
| `reasons`, `recommended_actions` | Safety explanations and local/review next steps |

Review lookup failures remain `unavailable`; GHA can still plan local work if
the PR identity is available. It refuses a provider head/state change during
preflight. A head change during fetch leaves the refreshed private ref and
refuses checkout creation; rerun inspection rather than forcing a reset.

Validation and operational failures exit nonzero. Diagnostics belong on stderr
with stable error code `pr_worktree_failed` for JSON/YAML. Preflight failures
emit no result on stdout. If execution began, stdout retains the result with
completed and failed effects. For example, a fetch can succeed before worktree
creation fails, or a checkout can exist before recording its PR association
fails. Inspect the result and checkout before retrying. Git may have created a
branch even when worktree addition failed; `branch_state` reports that fact.
