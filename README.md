# GHA - GitHub Assistant

![GitHub](https://img.shields.io/badge/go-1.26.5-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

GHA is a command-line assistant for developers working with GitHub and local repositories. It brings GitHub, Git, CI, and local-repository signals into clear workflows for tasks such as reviewing pull requests, inspecting branches, and preparing releases. The commands are useful directly in a terminal, and structured output lets coding agents use those same workflows when you want them to.

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

## Quick start

Use GHA directly in your terminal. Text output is intended for people; add `--format json` when you need structured results for a script or agent.

```bash
# Discover the commands and capabilities in this build.
gha --help
gha capabilities --format json
gha version --format json

# Inspect pull requests and local repository state.
gha prs
gha review 123
gha releases --limit 10
gha branches
gha analyze
gha branches cleanup --format json

# Prepare release notes from merged pull requests.
gha release create-notes --since 2026-09-01

# Preview workflows that can make changes.
gha branch publish feature/reviews --dry-run
gha pr prepare --title "Improve reviews" --head feature/reviews
gha pr create --title "Improve reviews" --head feature/reviews --dry-run
gha release publish 1.2.3 --dry-run --format json
```

## Use GHA with coding agents

GHA can also provide its structured GitHub and local-repository workflows to coding agents. It bundles a portable `gha` skill that explains how agents can use those workflows. To detect configured harnesses and install or refresh the guidance, run:

```bash
gha agent install
```

The command detects configured Codex, Claude Code, Pi, OpenCode, GitHub Copilot, Gemini CLI, Cursor, Hermes Agent, and OpenClaw setup directories and configures those harnesses without requiring their executables. Use `--agent` with comma-separated names to choose explicitly, or `--binary-only` to skip harness setup. If none are detected, GHA reports how to configure one later. It copies the bundled detailed skill into a discoverable global skill directory and idempotently adds a **compact** GHA guidance block where a documented global instruction file exists. Agents can use the detailed skill explicitly when a workflow needs it; the default block leaves simple Git tasks to Git. Existing skills are not overwritten, and modified GHA skills are preserved by update and uninstall. `gha agent list` shows recorded harnesses and current file presence.

```bash
# Preview paths without writing files.
gha agent install --agent codex,claude --dry-run

# Preview setup for Pi, OpenCode, Copilot, Gemini CLI, Cursor, Hermes, and OpenClaw.
gha agent install --agent pi,opencode,copilot,gemini,cursor,hermes,openclaw --dry-run

# Remove GHA's managed guidance block and its installed skill.
gha agent uninstall --agent codex --dry-run

# Use the local build while developing GHA.
make skill-install
```

Preview and refresh installed agent guidance with:

```bash
gha update --dry-run
gha update
```

Verify the product workflow without touching your agent setup:

```bash
make test-skill-install
```

GHA uses `GHA_GITHUB_TOKEN` when set; otherwise it uses `gh auth token` when the GitHub CLI is installed and authenticated. Without either, public API requests are unauthenticated. For private repositories or higher API rate limits, provide a fine-grained token restricted to the repositories GHA will access. The full command surface uses `Contents: read`, `Pull requests: write`, `Checks: read`, `Actions: read`, and `Issues: read`; `Pull requests: write` is required for `gha pr create` and includes pull-request read access. Git branch publication uses the checkout remote's authentication, not the GitHub API token. A classic token needs the `repo` scope for private repositories.

## Agent efficiency tests

Benchmarks of agent correctness, token use, and command effort are in the
[test results](docs/agent-efficiency-test-results.md). They include a
git/gh-only control and a check of compact default guidance. See the
[evaluation method and limitations](docs/agent-efficiency-evaluation.md)
for how the runs were scored.

## Documentation

- [Getting started](docs/getting-started.md) — installation, authentication, and first commands
- [Command guide](docs/command-guide.md) — output contracts and examples for every available workflow
- [Development](docs/development.md) — build, test, quality, contribution, and repository layout
- [Agent efficiency test results](docs/agent-efficiency-test-results.md) — benchmark tables and findings
- [Agent efficiency evaluation](docs/agent-efficiency-evaluation.md) — paired agent benchmark and reporting method
- [Release preparation](docs/development.md#preparing-a-release) — reviewed notes and tag publication
- [Roadmap](docs/roadmap.md) — delivered work, prioritized next steps, and future phases
- [Architecture](ARCHITECTURE.md) — current implementation architecture
- [Design](DESIGN.md) — long-term vision and design principles
- [Architecture decision records](docs/adr/) — decisions behind the project structure

## License

GHA is distributed under the [MIT License](LICENSE).
