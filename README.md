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

On macOS, install the released binary through Homebrew:

```sh
brew install --cask raithlin/tap/gha
```

See [package installation and upgrades](docs/package-distribution.md) for
integrity checks, upgrades, native Linux packages, and the release-maintained
tap. Native Linux packages become available with the first release containing
the packaging configuration; earlier releases provide archives.

You can also download a Linux or macOS archive from
[GitHub Releases](https://github.com/Raithlin/gha/releases).
Install Git for local repository workflows and pushes. The
[GitHub CLI (`gh`)](https://cli.github.com/) is optional; GHA can reuse its login
for GitHub access.

You do not need Go to use a release binary.

Follow the [setup guide](docs/getting-started.md#install-a-release) to choose
your download, verify its checksum, and put `gha` on your PATH. The guide also
covers [building from source](docs/getting-started.md#build-from-source).

GHA currently publishes alpha releases. This README describes the current
source; an older release may have fewer commands. Run `gha --help` to check
your installation, or `gha capabilities --format json` if your build supports it.

## First use

If you use the GitHub CLI, sign in if you have not already:

```sh
gh auth login
```

GHA chooses GitHub credentials in this order:

1. Use `GHA_GITHUB_TOKEN` if set, without invoking `gh`.
2. Otherwise, try `gh auth token` to reuse your GitHub CLI login.
3. If `gh` is missing or cannot supply a token, continue without authentication.

Local inspection works without GitHub credentials. Public GitHub requests can
also work without authentication, subject to lower API rate limits. Private
repositories and operations that require authentication need a token or a
working `gh` login. See the [authentication guide](docs/getting-started.md#authentication)
for token setup and permissions.

Open a terminal in a GitHub checkout and start with:

```sh
gha prs --limit 10
```

This lists up to ten open pull requests. Pick a number from the list and
inspect it, replacing `123` with that number:

```sh
gha review 123
```

The review summary shows CI and merge readiness, outstanding reviews and
threads, risk signals, and recommended next actions. Missing or unsupported
signals appear as unavailable.

## Common workflows

These examples describe the current source build. Use your installed command's
`--help` to check its options.

| What you want to do | Command |
| --- | --- |
| Find PRs awaiting your review | `gha prs --queue --limit 10` |
| Inspect another GitHub repository | `gha prs --repo owner/repo --limit 10` |
| Inspect local branches | `gha branches` |
| See branches worth checking for cleanup | `gha branches cleanup` |
| List local worktrees | `gha worktrees` |
| Analyze repository history, storage, and large files | `gha analyze` |
| List recent published releases | `gha releases --limit 10` |
| Get a structured review for an agent or script | `gha review 123 --format json` |

Local branch information uses cached Git refs unless you explicitly request a
refresh. Cleanup inspection lists candidates; it does not delete them.

Commands that change state execute by default. Preview their effects with
`--dry-run` first. For example, for an existing local branch:

```sh
gha branch publish feature/reviews --dry-run
```

The preview shows the publication target and safety checks without pushing.
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
explicitly. Existing releases may support fewer agents.

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
