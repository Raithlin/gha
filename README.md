# GHA - GitHub Assistant

![GitHub](https://img.shields.io/badge/go-1.26.5-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

GHA is an agent-first developer tool written in Go. It turns GitHub, Git, CI, and local-repository signals into bounded, versioned, decision-ready workflows for coding agents, with readable terminal output for developers.

It complements rather than replaces `git` and `gh`: a GHA command should add context, safety, or workflow value beyond a raw provider invocation.

## Install

Prerequisites: Go 1.26.5 or later and Git. A GitHub account is required for
GitHub-backed workflows.

```bash
git clone https://github.com/raithlin/gha.git
cd gha
make build

# Or build and install gha into GOBIN or GOPATH/bin.
make install
```

The examples below use an installed `gha` command. If you only ran `make build`, use `bin/gha` in its place, or run `make install`.

## Coding-agent guidance

GHA bundles a portable `gha` skill that helps coding agents use its structured GitHub and local-repository workflows. Install or refresh it with:

```bash
gha agent install --confirm
```

The command can configure Codex, Claude Code, Pi, OpenCode, GitHub Copilot, and Gemini CLI. Select multiple harnesses with comma-separated names. It copies the skill from the installed GHA binary into each supported global skill directory and idempotently adds a marked GHA guidance block where a documented global instruction file exists. `gha agent list` shows recorded harnesses and current file presence.

```bash
# Preview paths without writing files.
gha agent install --agent codex,claude --dry-run

# Preview setup for Pi, OpenCode, Copilot, and Gemini CLI.
gha agent install --agent pi,opencode,copilot,gemini --dry-run

# Remove GHA's managed guidance block and its installed skill.
gha agent uninstall --agent codex --dry-run

# Use the local build while developing GHA.
make skill-install
```

Verify the product workflow without touching your agent setup:

```bash
make test-skill-install
```

## Quick start

```bash
# Discover the commands and machine-readable capabilities in this build.
gha --help
gha capabilities --format json
gha version --format json

# Inspect pull requests and local branches in the current repository.
gha prs
gha review 123
gha releases --limit 10
gha branches
gha analyze

# Refresh cached origin refs only after reviewing the plan.
gha branches --refresh-origin --dry-run
gha branches --refresh-origin --confirm-origin

# Inspect an existing local branch before publishing it to origin.
gha branch publish feature/reviews --dry-run

# Review local cleanup candidates without deleting branches.
gha branches cleanup --format json

# Generate read-only release notes from merged pull requests.
gha release create-notes --since 2026-09-01

# Review the release tag and workflow plan before publishing.
gha release publish 1.2.3 --dry-run --format json
gha release publish 1.2.3 --confirm-origin

# Preview the exact pull-request creation plan, then create only after review.
gha pr prepare --title "Improve reviews" --head feature/reviews
gha pr create --title "Improve reviews" --head feature/reviews --dry-run
gha pr create --title "Improve reviews" --head feature/reviews --confirm
```

GHA uses `GHA_GITHUB_TOKEN` when set; otherwise it uses `gh auth token` when the GitHub CLI is installed and authenticated. Without either, public API requests are unauthenticated. For private repositories or higher API rate limits, provide a fine-grained token restricted to the repositories GHA will access. The full command surface uses `Contents: read`, `Pull requests: write`, `Checks: read`, `Actions: read`, and `Issues: read`; `Pull requests: write` is required for `gha pr create --confirm` and includes pull-request read access. Git branch publication uses the checkout remote's authentication, not the GitHub API token. A classic token needs the `repo` scope for private repositories.

## Documentation

- [Getting started](docs/getting-started.md) — installation, authentication, and first commands
- [Command guide](docs/command-guide.md) — output contracts and examples for every available workflow
- [Development](docs/development.md) — build, test, quality, contribution, and repository layout
- [Release preparation](docs/development.md#preparing-a-release) — reviewed notes and tag publication
- [Roadmap](docs/roadmap.md) — delivered work, prioritized next steps, and future phases
- [Architecture](ARCHITECTURE.md) — current implementation architecture
- [Design](DESIGN.md) — long-term vision and design principles
- [Architecture decision records](docs/adr/) — decisions behind the project structure

## License

GHA is distributed under the [MIT License](LICENSE).
