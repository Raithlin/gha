# Agent efficiency evaluation

This is a local, paired pilot for testing whether GHA helps a coding agent
answer repository workflow questions accurately with fewer tokens. It runs the
same Codex model on a pinned commit in a fresh clone for each task. One run is
instructed to use Git and source inspection without GHA; the other may use a
built GHA executable. Both use read-only sandboxes. The answers are checked
against fixed, reviewable expectations. The current
[`workflow_tasks.json`](../tools/evals/workflow_tasks.json) is pinned to commit
`16df66c472d5e63208f4a7786aa136d5c6fb4b79` and covers branch
reachability and cached upstream divergence. The initial diagnostic also
included largest tracked file lookup.
The harness pins Codex reasoning to low and alternates which condition runs
first. It replaces the clone's project agent instructions with identical
neutral guidance in both conditions.

## Run the pilot

Build from the pinned commit, then record its SHA. Use a model available
to your Codex account. The command previews the complete run plan by default;
`--execute` starts agent runs and consumes tokens.

Run `make test-eval` first to validate the local harness without model calls.

```bash
git clone --no-hardlinks . /tmp/gha-eval-source
git -C /tmp/gha-eval-source checkout --detach 16df66c472d5e63208f4a7786aa136d5c6fb4b79
make -C /tmp/gha-eval-source build
python tools/evals/agent_efficiency.py run \
  --tasks tools/evals/workflow_tasks.json \
  --repo /tmp/gha-eval-source \
  --revision 16df66c472d5e63208f4a7786aa136d5c6fb4b79 \
  --gha /tmp/gha-eval-source/bin/gha --model YOUR_MODEL --repeats 2 \
  --out /tmp/gha-agent-eval-1

# Run after reviewing the plan.
python tools/evals/agent_efficiency.py run \
  --tasks tools/evals/workflow_tasks.json \
  --repo /tmp/gha-eval-source \
  --revision 16df66c472d5e63208f4a7786aa136d5c6fb4b79 \
  --gha /tmp/gha-eval-source/bin/gha --model YOUR_MODEL --repeats 2 \
  --out /tmp/gha-agent-eval-1 --execute

python tools/evals/agent_efficiency.py report \
  --input /tmp/gha-agent-eval-1/runs.jsonl \
  --public-output /tmp/gha-eval-public-results.json
```

The output directory contains `plan.json`, `runs.jsonl`, and one raw Codex
event trace per attempt. Treat traces as private: they may contain prompts,
tool output, and repository content. Review them before sharing. No agent run
publishes code or contacts a Git remote as part of the benchmark prompt.
`--public-output` exports only whitelisted metrics, with no answers, stderr,
trace text, or private output paths. Review even this export before publishing.

## Why the initial result did not show GHA efficiency

On 27 September 2026, Codex CLI 0.156.1 ran `gpt-6-luna` with low reasoning.
The GHA binary SHA-256 was
`8f6d3786e3c53dc12261801d1d492bbe3d3104b4ec69b67f5649ddad898882f7`.
Each of three tasks ran twice per condition, alternating the order of
conditions. The initial harness accepted any GHA invocation as use of GHA.
Trace review showed that five of six GHA-enabled attempts ran
`gha capabilities --format json` and then used Git for the task. The sixth ran
`gha analyze` and repeated the lookup with Git. The capability inventory
returned 7,516 characters on every GHA-enabled attempt. This adds a model
turn and context without replacing a Git lookup. The initial token totals are
therefore a diagnostic of the setup, not a measurement of GHA task workflow
efficiency.

| Measure | Control | With GHA |
| --- | ---: | ---: |
| Correct attempts | 5/6 | 6/6 |
| Total input and output tokens | 181,544 | 382,061 |
| Tokens per correct attempt | 36,309 | 63,677 |
| Mean elapsed time per attempt | 12.4 s | 18.4 s |

The only correctness difference was one branch reachability attempt where the
control returned empty answers. It ran no inspection command, so this cannot
be attributed to GHA. The [sanitized per-attempt results](evaluation-results/2026-09-27-workflows.json)
are retained as diagnostic evidence. Traces and complete run records were
kept privately in `/tmp/gha-agent-eval-workflows` for review.

A stricter follow-up required the GHA task command. The agent used
`gha branches` in both cached upstream attempts and answered correctly. Those
runs consumed 65,258 and 48,257 tokens, versus 29,049 and 43,602 for their
Git controls. GHA returned about 3,500 characters of branch data after the
7,516-character inventory; the Git controls retrieved under 900 characters of
branch data. In the reachability attempts, some agents answered without any
repository inspection. The harness now marks those attempts invalid. This
follow-up is still too small and inconsistent to support an efficiency claim.

## Corrected branch workflow rerun

On 27 September 2026, the corrected harness ran the two current tasks twice
per condition with `gpt-6-sol` at low reasoning. All eight attempts inspected
the repository, all four GHA attempts used the selected GHA task command, and
all answers were correct. The [sanitized per-attempt results](evaluation-results/2026-09-27-branch-workflows-sol.json)
show:

| Measure | Git control | With GHA |
| --- | ---: | ---: |
| Correct attempts | 4/4 | 4/4 |
| Total input and output tokens | 146,454 | 227,309 |
| Tokens per correct attempt | 36,614 | 56,827 |
| Mean elapsed time per attempt | 18.9 s | 18.4 s |

In both branch reachability runs, the GHA agent read the 7,516-character
capability inventory, read 729 characters of command help, then received a
1,542-character cleanup result. The Git control used one shell command with
under 300 characters of output. In the upstream runs, the GHA agent read the
inventory and a roughly 3,500-character branch listing, while the Git control
used short focused queries. These output sizes and additional model turns
explain the higher observed token use; they do not prove how much each step
cost in isolation. Both conditions were equally accurate, and the sample is
too small to infer a general effect on quality or time.

## Developer terminal effort

The agent trace also provides a command-length proxy. After replacing the
absolute GHA binary path with `gha` and the temporary checkout path with `.`,
the four Git-control attempts issued six shell commands totalling 1,011
characters. The four GHA attempts issued ten commands totalling 346 characters,
including capability discovery and help. Their four task commands alone
totalled 172 characters. This is an accounting of agent-generated command
strings; it is not a human typing study.

A developer who knows the commands can use shorter Git paths than the agent
did. For the fixture, `gha branches cleanup --base eval/base` is 37 characters
and one command, while `git branch --merged eval/base` plus
`git branch --no-merged eval/base` total 61 characters and two commands. For
upstream inspection, `gha branches` is 12 characters and `git branch -vv` is
14. This suggests a terminal convenience for cleanup, but little typing
advantage for a simple branch listing. A human study would also measure time
to identify the right branch, use of shell history or completion, and mistakes
or rework after reading the output.

## What the numbers mean

`correct` requires every expected answer field to match. `valid_runs` excludes
failed Codex invocations, missing answers, runs without a successful inspection
command, control runs that invoked GHA, and GHA runs that skipped the selected
task workflow.
Success rate counts every scheduled run, including invalid and timed-out runs.
`total_tokens` sums Codex-reported input and output tokens over completed turns,
including retries. Cached input tokens are shown separately and are already
part of input tokens. `tokens_per_correct` divides all measured tokens from
scheduled runs, including wrong answers, by their correct answer count. If any
run lacks usage data, the metric is unavailable. Wall time includes the
entire agent invocation.

The current two tasks test offline repository lookup using a generated
branch-history fixture. They are an initial measurement pilot, not
sufficient evidence of a general quality or token advantage. Expand the
held-out set to realistic PR review,
branch safety, and release preparation tasks before making a product claim.
Score code or judgment tasks with predeclared checks and blinded human review;
inspect failures where the checks and a valid solution disagree. Publish the
task list, pinned model and revision, repeated paired results, missing usage,
and limitations with any README summary.

## Skill guidance ablation

The [three-arm task set](../tools/evals/skill_ablation_tasks.json) tests how
much guidance the agent needs to discover GHA. Every arm has the same GHA
binary on `PATH` and the same pinned repository fixture. The `binary_only`
arm has neutral read-only instructions; `short_guide` adds a brief GHA usage
hint to `AGENTS.md`; `full_skill` installs the repository's complete
`skills/gha/SKILL.md` in the clone's discoverable `.agents/skills/gha`
directory and adds its bundled `AGENT-GUIDANCE.md` block to `AGENTS.md`.
The task prompt itself does not name GHA or forbid direct Git.
Codex runs with an isolated home and read-only sandbox. Each task runs twice
per arm, with arm order rotated on the second pass.

The set covers combined branch reachability and cached upstream analysis,
branch creation and deletion previews, and a tag publication preview. The
write-intent tasks instruct the agent to use only dry-run commands. The
harness also rejects a GHA write command without `--dry-run` and checks that
Git refs are unchanged after every attempt. It records correctness, actual
GHA use, observed skill-file reads, tokens, elapsed time, and a proxy for
terminal effort: command count and normalized command-string characters.
These command strings are agent generated, not human keystrokes.

Preview the plan, then add `--execute` to run model calls:

```bash
python tools/evals/skill_ablation.py run \
  --tasks tools/evals/skill_ablation_tasks.json \
  --revision 16df66c472d5e63208f4a7786aa136d5c6fb4b79 \
  --gha bin/gha --skill skills/gha/SKILL.md \
  --model gpt-6-sol --repeats 2 \
  --out /tmp/gha-skill-ablation-1

python tools/evals/skill_ablation.py report \
  --input /tmp/gha-skill-ablation-1/runs.jsonl \
  --tasks tools/evals/skill_ablation_tasks.json \
  --public-output /tmp/gha-skill-ablation-public.json
```

The report export contains only whitelisted metrics. Keep the raw traces in
`/tmp` private; they can include repository contents and model prompts.
Interpret skill-loading counts as observed reads in the trace, since Codex
may expose a skill through discovery before a visible file-read command.

### 27 September 2026 run

The expanded run used `gpt-6-sol` at low reasoning, pinned commit
`16df66c472d5e63208f4a7786aa136d5c6fb4b79`, two repetitions per task
and arm, and GHA binary SHA-256
`8f6d3786e3c53dc12261801d1d492bbe3d3104b4ec69b67f5649ddad898882f7`.
All 18 attempts were valid, reported token usage, and left local refs
unchanged. No trace contained a GHA write command without `--dry-run`.
The [sanitized run record](evaluation-results/2026-09-27-skill-ablation-sol.json)
contains per-attempt metrics and no answers or raw trace text.

| Measure | Binary only | Short guide | Full skill |
| --- | ---: | ---: | ---: |
| Correct | 5/6 | 6/6 | 6/6 |
| Used GHA | 3/6 | 4/6 | 6/6 |
| Observed skill-file read | 0/6 | 0/6 | 6/6 |
| Input and output tokens | 424,554 | 325,434 | 379,920 |
| Tokens per correct answer | 84,911 | 54,239 | 63,320 |
| Mean seconds | 28.1 | 23.9 | 28.5 |
| Agent shell commands | 23 | 21 | 21 |
| Normalized command characters | 2,946 | 1,458 | 1,276 |

The full skill reliably triggered GHA, but this did not yield the lowest
token use. It ran GHA for both branch-analysis attempts, where focused Git
queries were shorter: that task consumed 123,530 tokens with the full skill,
versus 57,375 with the short guide. All four write-preview attempts in both
guided arms used GHA and were correct. On branch create/delete previews,
the full skill consumed 136,812 tokens versus 143,379 with the short guide.
On tag previews it consumed 119,578 versus 124,680. These task totals include
all turns and discovery overhead.

The initial exact-string grader marked one short-guide branch-analysis answer
wrong because it included a space after a comma in an otherwise correct
branch list. The grader now ignores spacing around comma separators, and the
saved answers were regraded without rerunning the model. The remaining wrong
binary-only answer came from a `git push --dry-run` tag preview that reported
the local effect as `unchanged` and origin as `create`; it did not provide
GHA's requested preflight effects. Both attempts remain in token and time
totals. Command characters strip Codex's shell-launch wrapper and count the
agent-generated shell body; they do not measure a person's typing time,
completion use, or error recovery.

Six attempts per arm cannot establish a general quality or token advantage.
The task set uses a local Git fixture, so it does not test live PR review,
CI, provider availability, release preflights, or human terminal work. The
observed pattern supports trying compact install guidance for common tasks
while retaining the full skill for discoverability and workflows where its
extra instructions improve correctness or safety. A larger held-out set and
human terminal study are needed before changing the default installer.

## All implemented capabilities: 27 September 2026

The follow-up used GHA's own `capabilities --format json` inventory as the
scope: 22 available entries, comprising 21 operational commands and the
inventory command itself. `dashboard` was reported unavailable and excluded.
The [direct capability probe](../tools/evals/capability_probe.py) exercised
all 22 against disposable local fixtures and public Raithlin/gha data. Every
write-capable probe used `--dry-run`. Twenty-one exited successfully. The
`update --dry-run` probe failed closed because the currently fetched published
skill does not declare `gha-required-capabilities`; it changed no state. All
22 probes left checkout refs, files, temporary home, and GHA configuration
unchanged. See the [sanitized direct results](evaluation-results/2026-09-27-capability-probe.json).

The [eight agent tasks](../tools/evals/all_capability_tasks.json) group the
same 22 entries into agent setup, repository diagnosis, branch plans, tag
publication, PR inspection, PR creation preflight, release research, and
release publication preflight. This is command coverage by the direct probe
and task coverage by the agent comparison: a correct agent may use a different
command than the one mapped to its task. Each task ran twice per setup with
`gpt-6-sol` at low reasoning on pinned commit
`16df66c472d5e63208f4a7786aa136d5c6fb4b79`. The three setups were
binary only, short guide, and full installed skill plus guidance. Provider
tasks allowed unauthenticated reads of public GitHub data; agents had isolated
homes and disposable checkouts. All 48 attempts completed, used no
write-capable command without a dry run, and left refs and files unchanged.
The [public PR and release facts](evaluation-results/2026-09-27-provider-snapshot.json)
were checked again after the run.

| Measure | Binary only | Short guide | Full skill |
| --- | ---: | ---: | ---: |
| Strict correct answers | 13/16 | 14/16 | 15/16 |
| Correct after count ambiguity review | 13/16 | 16/16 | 16/16 |
| Attempts using GHA | 13/16 | 16/16 | 16/16 |
| Total input and output tokens | 1,861,023 | 1,111,159 | 1,476,567 |
| Mean elapsed seconds | 36.5 | 26.6 | 29.6 |
| Agent shell commands | 116 | 99 | 111 |
| Normalized command characters | 16,059 | 4,723 | 5,175 |

The strict grader expected `22` for available capabilities. Three setup
answers instead gave `21`, counting operational commands but excluding the
inventory command. Their other fields were correct. The public run file
retains the strict score; the reviewed row above accepts either interpretation
of that ambiguous prompt. The binary-only arm's three remaining wrong answers
were substantive: it marked a bounded PR list truncated in both repetitions
when it was complete, and marked a bounded release-notes inventory complete
when GHA reported truncation. The
[sanitized per-attempt results](evaluation-results/2026-09-27-all-capabilities-sol.json)
retain every token, timing, validity, and safety result. Two full-skill PR
preflight answers were initially flagged by a trace parser that treated help
text and a multiline `--body` as writes; the corrected parser rechecked the
saved traces and found both dry-run only. No model attempt was rerun for these
grading corrections.

| Scenario, two attempts per setup | Binary tokens | Short-guide tokens | Full-skill tokens |
| --- | ---: | ---: | ---: |
| Agent setup and update preview | 167,413 | **113,191** | 150,488 |
| Repository diagnosis | 159,268 | **105,607** | 174,354 |
| Branch change plans | 197,190 | **129,072** | 138,311 |
| Tag publication plan | 219,003 | **92,169** | 135,986 |
| PR inspection | 342,657 | **188,792** | 225,322 |
| PR creation plan | 193,334 | **151,267** | 243,166 |
| Release research | 248,106 | **134,323** | 190,717 |
| Release publication plan | 334,052 | **196,738** | 218,223 |

The clearest **quality** wins were bounded provider listings: explicit
`truncated` fields prevented the binary-only agent's three wrong completeness
claims. The largest **token** difference was the tag preview, where all arms
were correct but the short guide used 58% fewer tokens than binary only. The
short guide used the fewest tokens in every scenario and 40% fewer overall
than binary only, while matching the full skill's reviewed correctness. It
also used 25% fewer tokens than the full skill. On these complex tasks the
short guide already triggered GHA in all 16 attempts, so the full skill's
reliable invocation did not add a correctness advantage in this sample. The
binary-only arm used GHA in 13 attempts; this is a guidance comparison, not
an enforced GHA-versus-no-GHA comparison.

The sample has one repository, two repetitions per scenario, task prompts
derived from GHA's command surface, and live public provider reads. Results
do not establish general quality, token, or developer keystroke savings.
Command characters are agent-generated shell text with Codex's launcher
removed; no human typing or completion behavior was measured. The direct
probe and [resumable agent harness](../tools/evals/skill_ablation.py) preserve
the exact task and binary hashes for later replication; private traces are
kept under the ignored `.eval-private/` directory.

To repeat the sweep, first run `make test-eval` and build `bin/gha` from the
same source revision. Preview the agent plan by omitting `--execute`; adding
it starts model calls. If interrupted, run the same command with `--resume`.

```bash
python tools/evals/capability_probe.py \
  --revision 16df66c472d5e63208f4a7786aa136d5c6fb4b79 \
  --gha bin/gha --out docs/evaluation-results/capability-probe-new.json

python tools/evals/skill_ablation.py run \
  --tasks tools/evals/all_capability_tasks.json \
  --revision 16df66c472d5e63208f4a7786aa136d5c6fb4b79 \
  --gha bin/gha --skill skills/gha/SKILL.md --model gpt-6-sol \
  --repeats 2 --out .eval-private/all-capabilities-new --execute

python tools/evals/skill_ablation.py report \
  --input .eval-private/all-capabilities-new/runs.jsonl \
  --tasks tools/evals/all_capability_tasks.json \
  --public-output docs/evaluation-results/all-capabilities-new.json
```

## Git/gh-only control: 28 September 2026

The earlier `binary_only` arm was not a raw control: GHA remained on `PATH`
and was used in 13 of 16 attempts. This follow-up compared the saved
`short_guide` GHA attempts with a separate control that had no GHA guidance,
skill, or executable on `PATH`. The control invoked no GHA command. It used
Git and source inspection for local tasks, and authenticated `gh` read calls
for public provider tasks. A preliminary unauthenticated run fell back to
direct public HTTP and is retained as [diagnostic data](evaluation-results/2026-09-28-raw-control-sol.json),
not counted in the git/gh-only provider comparison.

The seven control tasks are exact copies of the corresponding questions and
expected answers from the [eight-scenario task set](../tools/evals/all_capability_tasks.json).
The GHA-only agent setup scenario was excluded: `capabilities`, `version`,
agent installation, and guidance update have no git/gh counterpart. Both arms
used `gpt-6-sol` at low reasoning, commit
`16df66c472d5e63208f4a7786aa136d5c6fb4b79`, and two repetitions per
scenario. The control ran on 28 September; the reused GHA arm ran on 27
September. All 28 selected attempts were valid, answered correctly, left refs
and files unchanged, and had complete token usage.

| Scenario, two attempts per arm | Git/gh tokens | Short-guide GHA tokens | GHA token change |
| --- | ---: | ---: | ---: |
| Repository diagnosis | 164,501 | 105,607 | −36% |
| Branch change plans | 257,572 | 129,072 | −50% |
| Tag publication plan | 241,314 | 92,169 | −62% |
| PR inspection | **151,130** | 188,792 | +25% |
| PR creation plan | 179,515 | 151,267 | −16% |
| Release research | 202,392 | 134,323 | −34% |
| Release publication plan | 238,274 | 196,738 | −17% |
| **Seven-scenario total** | 1,434,698 | **997,968** | **−30%** |

Both arms scored 14/14. The GHA arm averaged 27.5 seconds per attempt versus
38.7 for git/gh. Agent-generated command strings totaled 4,250 characters
with GHA versus 13,802 for git/gh; these are not measured human keystrokes.
For the three questions less tied to GHA's output vocabulary—repository
diagnosis, PR inspection, and release research—the totals were 428,722 GHA
tokens and 518,023 git/gh tokens (17% fewer with GHA), with both arms 6/6
correct. PR inspection favored `gh`, while the large tag-plan difference
favored GHA. The tag question asked for `planned` effect words from GHA's
dry-run contract, so that difference is not evidence of the same advantage on
an independently designed tag task.

This is an exploratory comparison. The questions were designed around GHA's
implemented commands and several answer fields mirror its structured output.
The GHA attempts were reused rather than run concurrently with the control;
public GitHub state, service conditions, and model behavior could have changed.
The control had a token for authenticated `gh` reads while the earlier GHA
arm used unauthenticated public reads. The sample has one repository and two
repetitions per scenario. Equal correctness here does not establish a quality
benefit over git/gh, and the token difference does not establish a general
benefit across repositories or independently written tasks.

The [sanitized combined comparison](evaluation-results/2026-09-28-git-gh-comparison.json)
contains per-scenario tokens, correctness, time, command count, and command
characters. Its sources are the [saved GHA attempts](evaluation-results/2026-09-27-all-capabilities-sol.json),
[local control attempts](evaluation-results/2026-09-28-raw-control-sol.json),
and [strict gh provider attempts](evaluation-results/2026-09-28-strict-gh-control-sol.json).
The [comparison script](../tools/evals/compare_raw_control.py) checks matching
task definitions, model, revision, repetitions, and control safety signals.
Raw Codex traces remain in the ignored `.eval-private/` directory. The strict
provider traces were checked for actual `gh` calls, direct HTTP clients,
GHA invocation, and accidental token disclosure.

## Compact guidance check: 28 September 2026

Before changing the default installer guidance, a separate small Git
repository was generated from a pinned, single-commit source. Its
[source generator](../tools/evals/make_held_out_source.py) recreates commit
`e801cb80ec060f64fe899f970e921446ef37e438`; the harness adds a new local
branch graph in each disposable clone. The two retained
[held-out questions](../tools/evals/held_out_complex_tasks.json) ask which
topic tips are included in `stable` and whether a tracked topic is actually
in sync with its local origin-tracking ref. Neither prompt names GHA or its
commands. Every setup ran each task twice with `gpt-6-sol` at low reasoning.

| Setup | Correct | Used GHA | Total tokens |
| --- | ---: | ---: | ---: |
| Short guide only | 4/4 | 0/4 | 142,254 |
| Former long guidance + detailed skill | 4/4 | 4/4 | 293,992 |
| Compact guidance + detailed skill | 4/4 | 0/4 | 142,581 |

The third setup is the proposed default: it keeps the detailed skill
installed but uses the compact guide in the always-present instruction file.
It behaved like the short-guide-only arm on the local branch questions. These
questions were answerable with focused Git commands, so GHA non-use is not a
failure. A separate [PR and release check](../tools/evals/hybrid_provider_tasks.json)
with the third setup used GHA in all 4/4 attempts and answered all four
correctly, consuming 395,986 tokens. Those provider prompts were taken from
the earlier GHA-shaped sweep and test discovery, not independent quality.

The original [three-task held-out file](../tools/evals/held_out_guidance_tasks.json)
also asked whether the working tree was clean. The harness itself adds an
untracked `AGENTS.md`, making its predeclared `yes` answer wrong. All four
agents answered `no`; the public file retains those strict failures. The
simple task is excluded from the table above because its expected answer was
incorrect. This correction applies equally to both setups and was made after
the run; no model answers were changed or rerun. The comparison remains a
small local pilot, with two branch tasks and two repetitions, rather than
proof that compact guidance will invoke GHA appropriately across every
repository and harness.

The [short/full run data](evaluation-results/2026-09-28-held-out-guidance-sol.json),
[compact-plus-skill data](evaluation-results/2026-09-28-held-out-hybrid-sol.json),
and [provider check data](evaluation-results/2026-09-28-hybrid-provider-sol.json)
contain sanitized per-attempt tokens and safety metrics. The source generator
and task definitions allow replication; raw traces remain private in
`.eval-private/`. Based on this check, the installer still copies the detailed
skill for explicit use, while its bundled `AGENT-GUIDANCE.md` becomes a short
default instruction block. Existing user-modified skill files retain the
installer's preservation rules.
