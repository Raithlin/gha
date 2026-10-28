# Development

## Prerequisites

- Go 1.26.5 or later
- Make (optional, for using `makefile`)
- Git
- Python 3 for repository PR/architecture tooling and the optional evaluation harness
- [golangci-lint](https://golangci-lint.run/docs/welcome/install/) for local linting

Backlog.md and Groma are **optional, highly recommended** contributor tools.
They provide task history and architecture context. You can build and run the Go
quality gate without installing either. The repository PR entrypoint uses Groma,
and CI checks committed architecture freshness for every PR. See
[Tasks and architecture](#tasks-and-architecture) for the workflow.

## Commands

```bash
# Build the application
make build          # Creates bin/gha
make run            # Runs with go run
go build ./...      # Alternative build command

# Testing
make test           # Run all tests
go test ./...       # Alternative test command
make check          # Mirror CI: dependencies, tidy check, lint, race tests, and coverage
make test-eval      # Test the local evaluation harness without model calls
make test-coverage-report # Test coverage summary generation

# Code quality
make fmt            # Format code and tidy dependencies
make lint           # Run golangci-lint with .golangci.yml
go fmt ./...        # Alternative formatting
go vet ./...        # Alternative vet

# Cleanup
make clean          # Remove bin/ directory

# Build the app and choose one or more supported coding-agent harnesses.
make skill-install

# Verify the product installer is repeatable.
make test-skill-install
```

## Repository layout

```text
.
├── README.md           # Project entry point
├── .backlog/           # Backlog.md tasks and project configuration
├── .groma/             # Groma architecture records and scanner configuration
├── docs/               # Architecture, design, guides, and decision records
├── cmd/gha/            # Application entry point
├── internal/
│   ├── commands/       # CLI command implementations
│   ├── branch/         # Branch inventory and safety inspection workflows
│   ├── config/         # Startup configuration
│   ├── git/            # Local Git repository resolution and inspection
│   ├── github/         # GitHub provider
│   ├── interfaces/     # Provider boundary
│   ├── output/         # Text, JSON, and YAML rendering
│   └── review/         # Pull-request review workflows
├── pkg/model/          # Shared data models
└── makefile            # Build automation
```

See [Architecture](architecture.md) for the current implementation and
[Design](design.md) for long-term principles.

## Contributing

See the [contributor guide](CONTRIBUTING.md) for bug reports, feature proposals,
and pull-request expectations.

Contributions are welcome.

1. Fork the repository.
2. Create a feature branch: `git checkout -b feature/amazing-feature`.
3. Commit your changes: `git commit -m 'Add amazing feature'`.
4. Push the branch: `git push origin feature/amazing-feature`.
5. Open a pull request.

Before opening a pull request, format Go code, include relevant tests, and
update documentation when a user-facing workflow or contract changes.

## Tasks and architecture

Use Backlog.md to understand planned work and preserve decisions and verification
results. Use Groma to explore the current architecture and find which component
owns a source file. Both tools store readable records in Git, so you can also
read [.backlog/tasks](../.backlog/tasks/) and the [architecture map](../.groma/index.md)
without their CLIs. These records provide context; check the current source when
making implementation decisions.

The workflow below is recommended when using these tools. The build and test
commands above work independently of them. Check each CLI's `--help` for the
installed version. Coding agents can read [AGENTS.md](../AGENTS.md) and the
tools' instruction commands for detailed repository guidance.

### Track work with Backlog.md

Start from the repository root and look for an existing task before creating
one. Small mechanical edits may not need a task.

```bash
backlog instructions overview
backlog search "release CI" --plain
backlog task view TASK-010 --plain
```

The overview explains the task lifecycle. Search returns matching records;
`task view` shows the description, status, criteria, plan, and notes. This
repository enables remote operations and active-branch checks, so some Backlog
commands may refresh Git refs. Treat a failed refresh as incomplete branch
context, rather than proof that no matching task exists.

For new work, read `backlog instructions task-creation`, then create a focused
task with a reason and verifiable acceptance criteria. `backlog task create
--help` lists the fields; creation prints the new task ID and file path.

Before starting an existing task, read `backlog instructions task-execution`.
Use the actual task ID and your own assignee name in place of these examples:

```bash
backlog task edit TASK-010 --status "In Progress" --assignee @your-name
backlog task edit TASK-010 --plan 'Inspect the release checks, add regression tests, implement the fix, and verify the result.'
backlog task edit TASK-010 --append-notes 'Recorded findings and verification evidence here.'
```

Each edit confirms the task update. Keep the plan current and record evidence
as work progresses. For code changes, follow the project's test-driven process:
write a failing test, implement the behavior, then refactor with tests passing.

When the work is ready, read `backlog instructions task-finalization`. Verify
each acceptance criterion and Definition of Done item before checking it.
Use `backlog task edit --help` for `--check-ac`, `--check-dod`, and
`--final-summary`, then set the status to `Done`. Leave completed tasks on the
board until cleanup. Use the Backlog CLI to change its tasks and other managed
records; direct Markdown edits can break metadata and history.

### Explore architecture with Groma

These commands read the stored architecture without scanning or changing it:

```bash
groma view --plain
groma view gha-cli --plain
groma view internal/commands/branch.go
```

The first command lists actors, systems, external systems, relationships, and
flows. The second shows the CLI container and its components. The source-file
query reports its owner (`branch-commands`) and incoming and outgoing file
relationships. Use `groma view branch-commands` to read that component's full
record. A file with no owner may be excluded or outside scanner coverage;
check the reported reason before drawing conclusions.

Run `groma agent-instructions` for the guide index. The `inspect`, `structure`,
`describe`, and `relationships` guides explain how to inspect and curate the
map. For human-oriented guidance, use `groma instructions`.

`groma scan` refreshes source evidence and can write architecture records.
`groma web` and interactive `groma view` without a target also scan before
opening their maps. Review the resulting Git diff. A first scan supplies
evidence and initial placement; curation adds responsibility, boundaries, and
relationships grounded in the source. If an agent reports a first scan, decide
whether to ask it to curate the map before it changes those meanings.

Maintain architecture Markdown through Groma's CLI. Keep useful descriptions
and stable source ownership when refreshing the map. Routine code edits do not
need a full curation pass. When changing structure, follow the structure guide's
checks, including two scans that preserve established ownership and curation.

### Prepare pull requests with current architecture

This repository has a local PR entrypoint that refreshes Groma before creating
a pull request. It does not change the public behavior of `gha pr create`.
Install Python 3 and Groma 0.7.7 for this workflow. The Go-only `make check`
continues to work independently of Groma.

During development, refresh the map from the current worktree:

```bash
make architecture-refresh
```

Review the `.groma` diff and the affected components with `groma view <owner>`
or the interactive map. Check descriptions, ownership, relationships, and flows
against the changed source; a clean scan cannot prove that their meaning is
accurate. Use Groma's CLI for any correction. Commit the reviewed records
together in one architecture update, run `make architecture-check` on the
committed result, then push the branch explicitly.

For branches published directly to `Raithlin/gha`, create the PR through this
entrypoint:

```bash
make pr-dry-run PR_TITLE="Describe the change" PR_BODY_FILE=/tmp/pr-body.md
make pr PR_TITLE="Describe the change" PR_BODY_FILE=/tmp/pr-body.md
# PR_BASE optionally overrides the provider default base branch.
```

The entrypoint requires a clean worktree and uses the current branch. It runs
a scan, stops if architecture records need review and committing, verifies the
local commit against the live origin branch, and then delegates to `bin/gha pr
create`. It never commits, pushes, or changes branch automatically. Scanner
messages go to stderr, leaving stdout for GHA's JSON result. Dry-run scans a
disposable copy of committed `HEAD` and does not modify local architecture.
Missing tools, scanner errors, stale records, and unpublished commits stop the
workflow with guidance. If the origin lookup fails, fix Git authentication
before retrying; cached remote refs do not establish publication.

The helper accepts an `origin` configured with the standard GitHub HTTPS,
`git@github.com:` SSH, or `ssh://git@github.com/` URL for `Raithlin/gha`, and
explicitly targets that repository. It rejects fork and other origins before
scanning or creating a PR. It does not support custom SSH host aliases.

For fork contributions, run the same architecture refresh, review, commit, and
check sequence above, then push to your fork. Open the PR against `Raithlin/gha`
through GitHub or `gh`, selecting your fork's head explicitly. For example,
replace `CONTRIBUTOR` and the branch name with your own:

```bash
gh pr create --repo Raithlin/gha --head CONTRIBUTOR:feature/amazing-feature --base master
```

Fork contributions do not use `make pr` or `make pr-dry-run`. Their PRs run the
same architecture freshness check in CI.

`make architecture-check` also checks a disposable copy of committed `HEAD`.
It detects both modified and newly generated `.groma` records while preserving
the author checkout. Commit local map corrections before running it. The
required **Build and Test** CI job runs the same tests and freshness check on
PRs, including PRs opened directly through GitHub or `gh`. It does not commit to
the branch or publish previews. GitHub Pages still publishes only after changes
reach `master`.

### Publish the architecture map

The [interactive architecture site](https://raithlin.github.io/gha/) lets readers
explore the committed map, flows, and source without installing Groma.
The [Publish architecture workflow](../.github/workflows/architecture.yml)
exports each push to `master` and deploys it through GitHub Pages. You can also
run the workflow manually from GitHub Actions. The deploy job accepts only
`master`.

The workflow pins Groma 0.7.7 and exports `HEAD`. It reads the committed `.groma`
records and source without scanning or changing them. Refresh and review the
stored architecture locally when necessary, then commit it with the code.
Static exports include source inspection and exclude Backlog tasks and editing
controls.

To generate the same snapshot locally with Groma installed:

```bash
groma export /tmp/gha-architecture-site --revision HEAD \
  --url https://raithlin.github.io/gha/
```

The command reports the export directory and writes the static site there.
Serve that directory with a local HTTP server to preview it. Generated files
stay outside the repository; Actions uploads its temporary export as a Pages
artifact. Repository settings must select **GitHub Actions** as the Pages build
source.

### Coverage summaries and project badges

The **Build and Test** job appends total statement coverage to its GitHub Actions
job summary from `coverage.out`, including when the 95% gate subsequently fails.
It uses `go tool cover -func`, so this is the statement-weighted total rather
than an average of package percentages. The existing 95% CI gate remains in
place. A missing or invalid report fails reporting instead of inventing a value.

The README CI badge follows pushes to `master`. Its release badge uses Shields'
GitHub release source with `include_prereleases` and `sort=date`, so alpha
releases appear even before a stable release exists. No version or percentage
is maintained by hand; the existing license badge is preserved.

Coverage is reported in CI; the README CI badge reflects the job that enforces
the coverage threshold. No separate coverage badge or Pages endpoint is needed.

To reproduce the reporting locally after `make check`:

```bash
python3 -B tools/coverage/report.py --profile coverage.out \
  --summary /tmp/gha-coverage-summary.md \
  --revision "$(git rev-parse HEAD)"
```

### Link tasks to the architecture

When using both tools, read `groma agent-instructions backlog`. Add planned
source files or element IDs as task references. Record changed files immediately
after each edit, before editing another file:

```bash
backlog task edit TASK-010 --add-ref internal/release/service.go
backlog task edit TASK-010 --modified-file internal/release/service.go
backlog task edit TASK-010 \
  --modified-file internal/release/service.go \
  --modified-file internal/release/service_test.go
```

These commands update the task's references and modified-file list.
`--modified-file` replaces the complete list, so preserve earlier entries and
append each new path in order. Supported source files link through their current
Groma owners; a reference alone does not create ownership. After a structural
Groma command, record every path it reports as created, changed, or removed and
update any replaced element IDs as the guide describes.

Include relevant task records and architecture changes in the same review as
the implementation so their context stays with the code.

## Preparing a release

The tag workflow publishes a reviewed Markdown file from the tagged commit.
Copy [the template](releases/TEMPLATE.md) to `docs/releases/<tag>.md`, using the
exact tag name (for example, `docs/releases/v0.4.0-alpha.md`).

1. Run `gha release create-notes --since <previous-release-time> --format json`
   to inventory merged pull requests. Review commits in the release range too:
   the GHA command reports merged PRs, not every direct commit.
2. Write the release summary and link changes that matter to users. Exclude
   administrative PRs from the notes, and mention human contributors with
   their GitHub `@` handles. The [v0.3.0-alpha notes](releases/v0.3.0-alpha.md)
   show the agreed format.
3. Replace every `REPLACE_ME` token, review the file, and merge it into the
   default branch before tagging. The release workflow fails on a missing,
   empty, or unfinished notes file.
4. From the current default-branch commit, run
   `gha release publish <tag> --dry-run --format json`. Verify that
   `release_notes.state` is `available` and its path is the reviewed file.
   After reviewing the plan, run the command without `--dry-run` to publish the tag. Wait for
   the Actions release run, then verify the GitHub Release, notes, prerelease
   status, and downloads. A tag push alone does not prove publication.

The workflow passes this versioned file to GoReleaser with `--release-notes`.
The bundled GHA preflight reads the committed workflow's
`GHA_RELEASE_NOTES_DIR` setting and checks the matching file at `HEAD`, so a
missing file blocks the tag before the workflow runs. Other repositories
without that setting retain the existing release-publication behavior.
