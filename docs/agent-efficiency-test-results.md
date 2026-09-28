# Agent efficiency test results

A paired Codex evaluation ran two offline branch tasks twice each with
`gpt-6-sol` at low reasoning. Every attempt inspected the repository, and each
GHA attempt used its selected GHA workflow.

| Measure | Git control | With GHA |
| --- | ---: | ---: |
| Correct attempts | 4/4 | 4/4 |
| Total input and output tokens | 146,454 | 227,309 |
| Mean elapsed time per attempt | 18.9 s | 18.4 s |

GHA did not reduce tokens on these short tasks. Each fresh GHA session read the
capability inventory before its task command, adding context and a model turn.
This small test says nothing conclusive about larger workflows where GHA may
replace several Git, GitHub, and CI lookups. See the [method and trace diagnosis](agent-efficiency-evaluation.md)
and [per-attempt results](evaluation-results/2026-09-27-branch-workflows-sol.json).

For terminal use, the goal is different. On the branch fixture,
`gha branches cleanup --base eval/base` is one 37-character command; the
corresponding `git branch --merged eval/base` and
`git branch --no-merged eval/base` total 61 characters across two commands.
For upstream inspection, `gha branches` (12 characters) and `git branch -vv`
(14 characters) are similar. These are command-length examples, not measured
human keystrokes or time savings.

## Skill guidance experiment

An expanded offline evaluation used the same GHA binary and pinned repository
in three setups: binary only, a short GHA guide, and the installed GHA skill
with its guidance block. Six attempts per setup covered combined branch
analysis, branch create/delete previews, and a tag publication preview.
All write-intent GHA commands used `--dry-run`; no Git refs changed.

| Measure | Binary only | Short guide | Full skill |
| --- | ---: | ---: | ---: |
| Correct attempts | 5/6 | 6/6 | 6/6 |
| Attempts using GHA | 3/6 | 4/6 | 6/6 |
| Total input and output tokens | 424,554 | 325,434 | 379,920 |
| Mean elapsed time per attempt | 28.1 s | 23.9 s | 28.5 s |
| Agent command characters | 2,946 | 1,458 | 1,276 |

The full skill reliably led the agent to use GHA, while the short guide had
the lowest token use with the same correctness in this small sample. The
short guide used GHA on all four write-preview attempts and used Git for the
two branch-analysis attempts. The full skill saved another 182 agent command
characters but consumed 54,486 more tokens than the short guide. The one
binary-only error came from a tag preview attempted with `git push --dry-run`
that did not answer the requested preflight fields. Command characters are a
proxy from agent traces, not human keystrokes. See the [method and task-level
analysis](agent-efficiency-evaluation.md) and [sanitized run
results](evaluation-results/2026-09-27-skill-ablation-sol.json).

## All implemented capabilities

A follow-up covered all 22 entries reported available by `gha capabilities`
(21 operational commands plus the inventory command) in eight workflow
scenarios. The unavailable `dashboard` was excluded. A direct probe exercised
each command; write-capable commands used `--dry-run`. Twenty-one probes
succeeded. `gha update --dry-run` refused a published skill that lacked its
required capability declaration. No probe changed checkout or temporary
agent state.

The same eight scenarios ran twice with each guidance setup using `gpt-6-sol`
at low reasoning. Public PR and release tasks used unauthenticated GitHub
reads; agents worked in disposable checkouts. Every attempt completed without
changing refs or files.

| Measure, 16 attempts per setup | Binary only | Short guide | Full skill |
| --- | ---: | ---: | ---: |
| Strict correct answers | 13 | 14 | 15 |
| Correct after resolving a count ambiguity | 13 | 16 | 16 |
| Attempts using GHA | 13 | 16 | 16 |
| Total input and output tokens | 1,861,023 | **1,111,159** | 1,476,567 |
| Agent command characters | 16,059 | **4,723** | 5,175 |

Three guided setup answers counted 21 operational commands, while the strict
expected count was 22 including `capabilities`. The binary-only arm made
three substantive errors: it misread bounded-result truncation twice for PRs
and once for release notes. The short guide used 40% fewer tokens than binary
only and 25% fewer than the full skill, with the same reviewed correctness as
the full skill. Its clearest quality benefit appeared in PR and release
listings, where GHA explicitly reports whether results were truncated. Agent
command characters are a proxy, not measured developer keystrokes.

See the [eight-scenario analysis](agent-efficiency-evaluation.md#all-implemented-capabilities-27-september-2026),
[per-attempt results](evaluation-results/2026-09-27-all-capabilities-sol.json),
and [direct command probe](evaluation-results/2026-09-27-capability-probe.json).

## Git/gh-only control

A follow-up ran the seven scenarios with a git/gh-only control and compared
them with the saved short-guide GHA attempts. GHA was absent from the control's
`PATH` and never invoked. Agent setup was excluded because GHA's inventory,
installer, and updater have no git/gh equivalent. Provider tasks used
authenticated, read-only `gh` calls; both arms used the same task questions,
model, pinned checkout, and two repetitions.

| Measure, 14 attempts per arm | Git/gh only | Short guide + GHA |
| --- | ---: | ---: |
| Correct answers | 14 | 14 |
| Total input and output tokens | 1,434,698 | **997,968** |
| Mean elapsed seconds | 38.7 | 27.5 |
| Agent command characters | 13,802 | 4,250 |

The GHA arm used **30% fewer tokens** with equal correctness in this sample.
For the three less GHA-specific questions (repository diagnosis, PR inspection,
and release research), it used **17% fewer**. Git/gh used **25% fewer tokens on
PR inspection**, while GHA's largest win was the tag preview, whose requested
effect words mirror GHA's output. The tasks were derived from GHA's command
surface, and the saved GHA arm ran a day earlier with unauthenticated public
reads; these results are exploratory, not a general efficiency claim. Command
characters are agent-generated text, not developer keystrokes. See the
[per-scenario comparison](agent-efficiency-evaluation.md#gitgh-only-control-28-september-2026)
and [sanitized result](evaluation-results/2026-09-28-git-gh-comparison.json).

## Compact guidance check

A held-out local repository tested two independently written branch questions
twice per setup. The short guide alone, detailed skill with its former long
guidance, and compact guidance with the detailed skill installed each answered
4/4 correctly. The first and third setups used Git for these simple local
checks; the former full setup used GHA on all four attempts. Their token totals
were 142,254, 293,992, and 142,581 respectively. In a separate PR and release
check, compact guidance with the skill installed used GHA on all four attempts
and answered 4/4 correctly. This supports keeping the detailed skill
available while making the compact block the installer default. See the
[held-out method and limitations](agent-efficiency-evaluation.md#compact-guidance-check-28-september-2026).
