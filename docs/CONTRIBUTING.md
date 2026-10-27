# Contributing to GHA

Thanks for taking an interest in GHA. You can help by reporting a reproducible
bug, testing a release, improving the documentation, or contributing a fix.
You do not need to start with a new command.

GHA is an agent-first developer tool. Its commands should give coding agents
stable, structured results and give developers clear terminal output. It
complements `git` and `gh`. Keep that purpose in mind when proposing a change.

## Report a bug

Search the [existing issues](https://github.com/Raithlin/gha/issues) first. If
there is no matching report, [open a bug report](https://github.com/Raithlin/gha/issues/new?template=bug_report.md)
with enough detail for someone else to reproduce it:

- The GHA version and installation method, operating system, and relevant Git
  or GitHub CLI versions.
- The exact command, expected result, and actual result, including the exit
  code and whether the message appeared on stdout or stderr.
- A small reproduction or failing test. For repository-dependent bugs, a
  disposable repository that demonstrates the problem is especially useful.

Use `gha version --format json` to identify the build. Remove tokens,
credentials, and private repository data from reports and logs.

If you can reproduce an existing report, add your findings there. A smaller
reproduction or confirmation on another platform also helps.

## Propose a feature

Check the [roadmap](roadmap.md) and existing issues before starting a substantial
change. Open a [feature request](https://github.com/Raithlin/gha/issues/new?template=feature_request.md)
describing the workflow, the difficulty you encounter, and how the change
would help. Include an example of the commands and result
you would want to use.

Explain what GHA adds beyond running `git` or `gh` directly. The
[Command Value Test](design.md#command-value-test) asks whether a command
combines useful signals, provides a stable contract, makes uncertainty clear,
or guides a safer workflow. Discuss changes to command behavior or public JSON
fields before building a large patch.

## Improve the documentation

Corrections, clearer examples, and reports of instructions that do not work
are welcome. Keep examples consistent with the current command help and
`gha capabilities --format json`. Include the command and result you checked
when an example depends on executable behavior.

See the [documentation index](README.md) to find the relevant guide. Small,
focused documentation fixes can go straight to a pull request.

## Contribute a patch

For a bug fix, start with a regression that fails before the fix. Keep each
pull request focused on one problem, and include the affected help or
documentation when behavior changes.

The [development guide](development.md) covers prerequisites, repository
layout, build commands, and the complete pull-request workflow. A typical
local check is:

```bash
make build
bin/gha capabilities --format json
make check
```

The build creates `bin/gha`; the next command prints its capability inventory.
`make check` runs lint, race-enabled tests, and the 95% statement-coverage gate.
Report the checks you ran and any limitations. Documentation-only changes need
link and example checks appropriate to the edit.

When changing commands, preserve the agent-facing contract:

- Return data on stdout and diagnostics on stderr. Support JSON and readable
  text from the same result.
- Keep unavailable evidence explicit. Cached refs and a successful tag push
  cannot stand in for fresh provider evidence or a completed release.
- Preserve existing JSON fields where possible and document compatibility
  decisions when a change cannot be additive.
- For mutations, retain explicit targets, safety checks, and a `--dry-run`
  preview. Commands execute by default; do not add `--confirm` flags.

Read the [task and architecture workflow](development.md#tasks-and-architecture)
when working with Backlog.md or Groma. They are recommended contributor aids;
you can build and run the Go checks without them. Architecture changes need the
refresh and review described in the development guide before publication.
Coding agents should also follow [AGENTS.md](../AGENTS.md).

## Open a pull request

Push your branch to your fork and open the PR against `Raithlin/gha`. Follow the
[documented fork workflow](development.md#prepare-pull-requests-with-current-architecture);
branches published directly to this repository use its `make pr` entrypoint.

Use the PR template to explain the problem, the resulting behavior, and how
you verified it. Link a related issue when one exists. Keep unrelated
formatting and refactoring out of the patch so reviewers can assess the change.
You remain responsible for the changes and verification evidence, including
work produced with a coding agent.

Respond to review questions and update the same PR as you address feedback.
If you cannot finish a change, leave a brief note describing what remains.

## Work with others

Keep discussion respectful and specific. Explain the behavior you want to
change, give evidence for technical claims, and review the work without
attacking the person who contributed it. Questions, reproduction steps, and
careful reviews are useful contributions too.
