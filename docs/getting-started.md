# Getting started

Install GHA, connect it to GitHub, then inspect a repository. Coding-agent
setup is optional. For a short tour, start with the [README](../README.md).

## Requirements

- Git for local repository workflows and branch or tag pushes.
- Linux or macOS on x86-64 or ARM64 for the published binaries.
- Optional: [GitHub CLI (`gh`)](https://cli.github.com/) to reuse its login.
  GHA also accepts `GHA_GITHUB_TOKEN` and continues without authentication if
  neither source supplies a token.
- GitHub authentication for private repositories and authenticated workflows.

Release binaries do not require Go. Building from source requires Go 1.26.5
or later; Make is optional. Published archives currently cover Linux and macOS,
not Windows.

GHA publishes prereleases. These docs describe the current source, which
may have commands an older binary lacks. Check `gha --help` and, when available,
`gha capabilities --format json` for what your installation supports. Build
from source if you need features that have not reached a release.

## Install a release

For Homebrew on macOS or native Linux packages, follow
[package installation and upgrades](package-distribution.md). The steps below
cover manual archive installation.

### Choose a download

Open [GitHub Releases](https://github.com/Raithlin/gha/releases) and choose a
release. Download its archive and `checksums.txt` into a separate directory.
Choose the archive suffix for your machine:

| Machine | Archive suffix |
| --- | --- |
| Linux, Intel or AMD x86-64 | `linux_amd64.tar.gz` |
| Linux, ARM64 | `linux_arm64.tar.gz` |
| macOS, Intel | `darwin_amd64.tar.gz` |
| macOS, Apple Silicon | `darwin_arm64.tar.gz` |

Run `uname -m` if you are unsure of the CPU architecture. `x86_64` means
`amd64`; `aarch64` or `arm64` means `arm64`.

The commands below use the published `0.4.0-beta.1` Linux x86-64 archive as an
example. Substitute the exact filename you downloaded throughout. Release
archives contain a `gha` executable, README, and license; they do not install
coding-agent guidance automatically.

### Verify the download

On Linux, from the directory containing the archive and checksum file, run:

```sh
sha256sum --check --ignore-missing checksums.txt
```

Your archive should appear with `OK`. You do not need the other release assets.
Stop if its checksum fails.

On macOS, select the line for your exact archive and check it:

```sh
grep ' gha_0.4.0-beta.1_darwin_arm64.tar.gz$' checksums.txt > selected-checksum.txt
shasum -a 256 --check selected-checksum.txt
```

Expect `gha_0.4.0-beta.1_darwin_arm64.tar.gz: OK`. Use the `darwin_amd64` filename
for an Intel Mac. An empty selection is not a successful check. This verifies
the download against the release's checksum file.

### Install for your user

Extract the verified archive and copy the executable into `~/.local/bin`.
On Linux x86-64, for example:

```sh
tar -xzf gha_0.4.0-beta.1_linux_amd64.tar.gz
mkdir -p "$HOME/.local/bin"
install -m 755 gha "$HOME/.local/bin/gha"
```

This installs GHA for your user without sudo. On macOS, use your downloaded
`darwin` archive in the `tar` command.

Add that directory to your current shell's PATH:

```sh
export PATH="$HOME/.local/bin:$PATH"
gha --help
```

You should see GHA's commands and options. To keep the PATH change in new
terminals, add the export line once to your shell configuration, such as
`~/.bashrc` for Bash or `~/.zshrc` for Zsh.

If you see `command not found`, try `~/.local/bin/gha --help`. If that works,
check your PATH and open a new terminal after saving the shell configuration.
Run `command -v gha` to check which binary your shell selects if you have more
than one installation.

## Build from source

Install Git and Go 1.26.5 or later. With Make available, run:

```sh
git clone https://github.com/Raithlin/gha.git
cd gha
make build
./bin/gha --help
```

This builds the current development checkout into `bin/gha`. The help output
lists its commands. To install the same source through Go, run:

```sh
make install
```

Go puts `gha` in `GOBIN`, or `$(go env GOPATH)/bin` when `GOBIN` is unset. Add
that directory to your PATH. Without Make, run `go install ./cmd/gha` from the
checkout for the same Go installation.

Building does not configure coding agents. See [agent setup](#set-up-a-coding-agent)
if you want that integration, or [Development](development.md) to work on GHA.

## Authentication

If you use the GitHub CLI, sign in and check your account:

```sh
gh auth login
gh auth status
```

The status should identify the GitHub account you intend to use. GHA uses
`GHA_GITHUB_TOKEN` when set; otherwise it tries `gh auth token`. With neither,
public API requests remain unauthenticated and have lower rate limits. Local
commands such as `gha analyze` work without GitHub credentials.

If you do not use `gh`, supply a fine-grained token through `GHA_GITHUB_TOKEN`
using your shell or secret manager. Restrict it to the repositories GHA needs.
Read-only inspection needs only the relevant read permissions; for the full
current command surface, grant:

- `Contents: read` for branch metadata, comparisons, and releases.
- `Pull requests: write` for `gha pr create`; this includes PR read access.
- `Checks: read` for review CI status.
- `Actions: read` to observe release workflows.
- `Issues: read` for `gha prs --assigned`.

Git branch and tag pushes use your checkout remote's Git authentication,
separately from the API token. A classic token needs the `repo` scope for
private repositories. Do not put tokens in repository files or issue reports.

If GitHub access fails, check the selected repository, authenticated account,
and token permissions. An explicitly set `GHA_GITHUB_TOKEN` takes precedence
over your `gh` login.

## Select a repository

Run GHA inside your local checkout, or select a GitHub repository explicitly:

```sh
gha prs --repo Raithlin/gha --limit 10
```

This lists up to ten open PRs from that repository. An empty list is a valid
result. Repository-aware commands select their provider target in this order:

1. `--repo owner/repo`.
2. The `origin` remote in an explicit `--path /path/to/checkout`.
3. `GHA_REPOSITORY`.
4. The current Git repository's `origin` remote.

`--path` selects an existing local checkout; resolving it does not clone or
fetch. Local workflows need a checkout. Commands that combine a provider target
and local changes can also require the two to match; see their help.

## First commands

Inside your GitHub checkout, list open pull requests:

```sh
gha prs --limit 10
```

Choose a PR number from the list, then replace `123` below:

```sh
gha review 123
```

The summary reports merge and CI readiness, outstanding reviewers and threads,
risks, and suggested next actions. Optional signals that cannot be obtained
remain unavailable.

Inspect local branches and worktrees:

```sh
gha branches
gha worktrees
```

These show local state. Branch output marks origin information as cached;
neither command fetches implicitly. Use `gha analyze` for offline history,
storage, and large-file analysis.

For agent or script input, request the same result as JSON:

```sh
gha review 123 --format json
```

Stdout contains the structured result; diagnostics go to stderr. Consult the
[command guide](command-guide.md) for schemas, filters, limits, and freshness.

Commands that change state execute by default. Use `--dry-run` to preview
supported writes before applying them. For the local branch `feature/reviews`:

```sh
gha branch publish feature/reviews --dry-run
```

This shows the target and safety checks without pushing. Follow the
[PR worktree lifecycle guide](pr-worktree-lifecycle.md) for isolated development,
existing PR checkout and resume, fork updates, and safe cleanup.

## Set up a coding agent

GHA includes a skill and compact guidance for supported coding agents. Preview
setup before writing to your agent configuration:

```sh
gha agent install --dry-run
```

The preview lists the files it would write for detected setups. Run
`gha agent install` to apply that selection. If none are detected, GHA explains
how to choose one explicitly. For example, to configure Codex and Claude Code:

```sh
gha agent install --agent codex,claude --dry-run
gha agent install --agent codex,claude
```

The second command installs the bundled skill and managed guidance at the
previewed paths. Current source supports Codex, Claude Code, Pi, OpenCode,
GitHub Copilot, Gemini CLI, Cursor, Hermes Agent, and OpenClaw. Detection uses
setup directories and does not require agent executables. Use comma-separated
names with `--agent` to choose explicitly.

Check GHA's recorded installations with:

```sh
gha agent list
```

This reports recorded destinations and file presence, not every agent installed
on your machine. Existing skills are preserved; modified GHA skills are also
preserved during updates and removal. See the
[agent command details](command-guide.md#agent-guidance-installation) for ownership
and shared destinations.

## Update

For a release binary, download and verify a newer archive, then repeat the
installation steps to replace your `gha` executable. GitHub downloads do not
add a package-manager repository or automatic binary updates.

For a source installation, update your checkout to the source you want and
repeat `make install`, or `make build` if you run `bin/gha` directly.

After updating the executable, preview agent-guidance updates:

```sh
gha update --dry-run
```

The preview checks the latest published guidance against your binary's
capabilities. Run `gha update` to apply compatible guidance to recorded agent
installations. It preserves incompatible skills and user content outside the
managed guidance blocks. **It does not update the GHA executable.** Published
guidance may lag a development build.

## Uninstall

If you configured agents, remove their GHA guidance before deleting the binary.
Use the names shown by `gha agent list`, for example:

```sh
gha agent uninstall --agent codex,claude --dry-run
gha agent uninstall --agent codex,claude
```

The first command previews removal; the second removes managed guidance and
unmodified managed skills. It preserves modified skills and files still shared
by another recorded agent.

For the per-user release installation in this guide, remove the executable:

```sh
rm "$HOME/.local/bin/gha"
```

For `make install` or `go install`, remove `gha` from the `GOBIN` or GOPATH
binary directory used at installation. Check `command -v gha` first to confirm
the location. A local build lives at `bin/gha` inside your source checkout.
Removing the executable leaves your repositories and GitHub data in place.
