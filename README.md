# GHA - GitHub Assistant

![GitHub](https://img.shields.io/badge/go-1.26.5-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

GHA is an agent-first developer tool written in Go. It turns GitHub, Git, CI,
and local-repository signals into bounded, versioned, decision-ready workflows
for coding agents, with readable terminal output for developers.

It complements rather than replaces `git` and `gh`: a GHA command should add
context, safety, or workflow value beyond a raw provider invocation.

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

## Quick start

```bash
# Discover the commands and machine-readable capabilities in this build.
gha --help
gha capabilities --format json

# Inspect pull requests and local branches in the current repository.
gha prs
gha review 123
gha branches
gha analyze
```

For private repositories, or to avoid unauthenticated GitHub API limits, set
`GHA_GITHUB_TOKEN` to a token that can read the repository.

## Documentation

- [Getting started](docs/getting-started.md) — installation, authentication, and first commands
- [Command guide](docs/command-guide.md) — output contracts and examples for every available workflow
- [Development](docs/development.md) — build, test, quality, contribution, and repository layout
- [Roadmap](docs/roadmap.md) — current scope, planned command migrations, and future phases
- [Architecture](ARCHITECTURE.md) — current implementation architecture
- [Design](DESIGN.md) — long-term vision and design principles
- [Architecture decision records](docs/adr/) — decisions behind the project structure

## License

GHA is distributed under the [MIT License](LICENSE).
