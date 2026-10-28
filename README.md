# GHA — GitHub Assistant

![GitHub](https://img.shields.io/badge/go-1.26.5-blue.svg)
[![CI](https://github.com/Raithlin/gha/actions/workflows/ci.yml/badge.svg?branch=master&event=push)](https://github.com/Raithlin/gha/actions/workflows/ci.yml?query=branch%3Amaster)
[![Latest release](https://img.shields.io/github/v/release/Raithlin/gha?include_prereleases&sort=date)](https://github.com/Raithlin/gha/releases)
![License](https://img.shields.io/badge/license-MIT-green.svg)

GHA helps you review pull requests, understand local branches, and prepare
releases. It combines GitHub, Git, and CI information into a summary with
suggested next steps, so you can see what needs attention before acting.

GHA is built for coding agents and developers. Use readable output in your
terminal or add `--format json` for scripts and agents. It works alongside
`git` and the GitHub CLI (`gh`).

- Review a pull request's CI status, reviewers, unresolved threads, and risks.
- Inspect local branches and worktrees, including cleanup candidates.
- Preview branch, pull-request, and release changes before applying them.
- Give your coding agent the same workflows through a bundled GHA skill.

## Install

On macOS, install GHA through Homebrew:

```sh
brew install --cask raithlin/tap/gha
```

On Linux, download a Debian/Ubuntu, Fedora, or Arch package from
[GitHub Releases](https://github.com/Raithlin/gha/releases). Linux and macOS
archives are also available for amd64 and arm64.

Follow the [package installation guide](docs/package-distribution.md) to verify
your download, install a package, or upgrade GHA. For manual archive installation
or building from source, follow the [setup guide](docs/getting-started.md).

Install Git for local repository workflows and pushes. The
[GitHub CLI (`gh`)](https://cli.github.com/) is optional; GHA can reuse its login
for GitHub access. Release binaries do not require Go.

GHA is prerelease software. Review the
[release notes](https://github.com/Raithlin/gha/releases) before upgrading.

## First use

If you use the GitHub CLI, sign in if you have not already:

```sh
gh auth login
```

Follow the prompts to sign in with the GitHub account you want to use.
GHA reuses your `gh` login. To use a token instead, set `GHA_GITHUB_TOKEN`.

Local inspection works without GitHub credentials. Public GitHub requests can
also work without authentication, subject to lower API rate limits. Private
repositories and operations that require authentication need a token or a
working `gh` login. See the [authentication guide](docs/getting-started.md#authentication)
for token setup and permissions.

Open a terminal in a GitHub checkout and start with:

```sh
gha prs --limit 10
```

This lists up to ten open pull requests. For example:

```text
Pull Requests (1 returned):

#42    Fix login redirect                       [open] by alex
```

The examples here use an illustrative PR. Your repository's numbers, titles,
and results will differ. Pick a number from the list and inspect it:

```sh
gha review 42
```

The review summary shows CI and merge readiness, outstanding reviews and
threads, risk signals, and recommended next actions. An excerpt might look like:

```text
Review readiness:
  Mergeable: true
  CI: failure
  Review threads: none

Risk signals:
  [high] ci_failure: one or more CI checks have failed

Recommended next actions:
  fix_ci: one or more CI checks have failed
```

Here, GitHub can combine the branches, but CI has failed. Check the failing
jobs before proceeding. Missing or unsupported signals appear as unavailable.

For an agent or script, request the same review as JSON:

```sh
gha review 42 --format json
```

JSON excerpt for the same PR:

```json
{
  "schema_version": "v1",
  "readiness": {
    "mergeable": true,
    "mergeable_state": "blocked",
    "ci_status": "failure",
    "review_threads_state": "none"
  },
  "recommended_actions": [
    {
      "action": "fix_ci",
      "reason": "one or more CI checks have failed"
    }
  ]
}
```

## Common workflows

Run `gha <command> --help` for command options.

| What you want to do | Command |
| --- | --- |
| Find PRs awaiting your review | `gha prs --queue --limit 10` |
| Inspect another GitHub repository | `gha prs --repo owner/repo --limit 10` |
| Inspect local branches | `gha branches` |
| See branches worth checking for cleanup | `gha branches cleanup` |
| List local worktrees | `gha worktrees` |
| Analyze repository history, storage, and large files | `gha analyze` |
| List recent published releases | `gha releases --limit 10` |
| Get a structured review for an agent or script | `gha review 42 --format json` |

Local branch information uses cached Git refs unless you explicitly request a
refresh. Cleanup inspection lists candidates; it does not delete them.

Commands that change state execute by default. Preview their effects with
`--dry-run` first. To preview creating a local branch:

```sh
gha branch create feature/reviews --dry-run
```

```text
Branch mutation: create feature/reviews
Dry run: no changes were made.
Local: planned
Origin: not_requested
```

`Local: planned` shows the intended change. `Origin: not_requested` means this
operation targets only the local repository. The dry run creates no branch.
See the [command guide](docs/command-guide.md) for the full workflows, or the
[PR worktree guide](docs/pr-worktree-lifecycle.md) to work on a PR in an isolated
checkout and safely retire it afterward.

## Use with a coding agent

Install the bundled skill and guidance so your agent knows when and how to use
GHA. Preview the files first:

```sh
gha agent install --dry-run
```

The preview lists destinations for detected agent setups. Apply it with:

```sh
gha agent install
```

GHA supports Codex, Claude Code, Pi, OpenCode, GitHub Copilot, Gemini CLI,
Cursor, Hermes Agent, and OpenClaw. Use `--agent codex,claude` to select agents
explicitly.

See the [agent setup guide](docs/getting-started.md#set-up-a-coding-agent) for
selection, installed-file checks, and removal. Agent setup is optional for
terminal use.

## Updates and removal

Install a newer binary using the same method you used originally.
`gha update` refreshes installed **agent guidance**, not the GHA executable.
See [updates](docs/getting-started.md#update) and
[uninstall](docs/getting-started.md#uninstall) for the steps.

## Documentation and contributing

- [Setup guide](docs/getting-started.md) — downloads, authentication, troubleshooting, and first use
- [Command guide](docs/command-guide.md) — commands, examples, and structured output contracts
- [Contributing](docs/CONTRIBUTING.md) — report problems, suggest features, and submit changes
- [Development](docs/development.md) — build, test, and contributor tooling
- [Roadmap](docs/roadmap.md) — current implementation and planned work
- [Architecture](docs/architecture.md) · [Interactive map](https://raithlin.github.io/gha/) — how GHA fits together
- [Agent evaluation results](docs/agent-efficiency-test-results.md) — measured correctness and command effort, with limitations
- [All documentation](docs/README.md)

[MIT licensed](LICENSE).
