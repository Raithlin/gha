#!/usr/bin/env python3
"""Local paired Codex evaluation for GHA's read-only agent workflows."""

import argparse
import hashlib
import json
import os
import re
import statistics
import subprocess
import tempfile
import time
from pathlib import Path


def load_tasks(path):
    tasks = json.loads(Path(path).read_text())
    if not isinstance(tasks, list) or not tasks:
        raise ValueError("tasks must be a nonempty JSON array")
    seen = set()
    for task in tasks:
        if not isinstance(task, dict) or not isinstance(task.get("id"), str) or not task["id"]:
            raise ValueError("each task needs an id")
        if task["id"] in seen:
            raise ValueError("duplicate task id: " + task["id"])
        seen.add(task["id"])
        if not isinstance(task.get("prompt"), str) or not task["prompt"]:
            raise ValueError("each task needs a prompt")
        expected = task.get("expected")
        if not isinstance(expected, dict) or not expected or any(
            not isinstance(k, str) or not isinstance(v, str) for k, v in expected.items()
        ):
            raise ValueError("each task needs nonempty string expected values")
        if not isinstance(task.get("gha_workflow"), str) or not task["gha_workflow"]:
            raise ValueError("each task needs a gha_workflow command")
    return tasks


def usage_from_events(events):
    totals = [0, 0, 0]
    found = False
    for event in events:
        if event.get("type") != "turn.completed" or not isinstance(event.get("usage"), dict):
            continue
        usage = event["usage"]
        if not all(isinstance(usage.get(key), int) for key in ("input_tokens", "output_tokens")):
            continue
        found = True
        for index, key in enumerate(("input_tokens", "cached_input_tokens", "output_tokens")):
            totals[index] += usage.get(key, 0)
    return tuple(totals) if found else None


def used_gha(events):
    for event in events:
        item = event.get("item", {})
        if item.get("type") == "command_execution" and re.search(
            r"(?<![\w/])(?:[\w./-]*/)?gha(?:\s|$)", item.get("command", "")
        ):
            return True
    return False


def used_gha_workflow(events, binary, workflow):
    pattern = re.compile(re.escape(str(binary)) + r"\s+" +
                         r"\s+".join(re.escape(part) for part in workflow.split()) + r"(?:\s|$)")
    for event in events:
        item = event.get("item", {})
        if event.get("type") != "item.completed" or item.get("type") != "command_execution" or item.get("exit_code") != 0:
            continue
        command = item.get("command", "")
        match = pattern.search(command)
        if match and not (workflow == "branches" and
                          re.match(r"cleanup\b", command[match.end():].lstrip())):
            return True
    return False


def inspected_repository(events):
    return any(event.get("type") == "item.completed" and
               event.get("item", {}).get("type") == "command_execution" and
               event["item"].get("exit_code") == 0 for event in events)


def grade(message, expected):
    try:
        answers = json.loads(message)["answers"]
    except (ValueError, KeyError, TypeError):
        return False
    return isinstance(answers, dict) and all(
        str(answers.get(key, "")).strip().casefold() == value.strip().casefold()
        for key, value in expected.items()
    )


def summarize(records):
    result = {}
    for condition in ("control", "gha"):
        runs = [record for record in records if record["condition"] == condition]
        valid = [record for record in runs if record["valid"]]
        correct = sum(bool(record["correct"]) for record in runs)
        measured = [record for record in runs if record["total_tokens"] is not None]
        tokens = sum(record["total_tokens"] for record in measured)
        result[condition] = {
            "runs": len(runs), "valid_runs": len(valid), "correct": correct,
            "success_rate": correct / len(runs) if runs else None,
            "gha_used_runs": sum(bool(record.get("gha_used")) for record in runs),
            "gha_workflow_used_runs": sum(bool(record.get("gha_workflow_used")) for record in runs),
            "usage_coverage": len(measured),
            "total_tokens": tokens if measured else None,
            "tokens_per_correct": tokens / correct if correct and len(measured) == len(runs) else None,
            "mean_seconds": sum(record["seconds"] for record in runs) / len(runs) if runs else None,
        }
    pairs = {}
    for record in records:
        if "task_id" in record and "repetition" in record:
            pairs.setdefault((record["task_id"], record["repetition"]), {})[record["condition"]] = record
    jointly_correct = [pair for pair in pairs.values() if
                       pair.get("control", {}).get("correct") and pair.get("gha", {}).get("correct")]
    result["paired"] = {
        "both_correct": len(jointly_correct),
        "median_tokens_when_both_correct": {
            condition: statistics.median(pair[condition]["total_tokens"] for pair in jointly_correct)
            if jointly_correct and all(pair[condition]["total_tokens"] is not None for pair in jointly_correct)
            else None for condition in ("control", "gha")
        },
    }
    return result


def output_schema(expected):
    return {
        "type": "object", "additionalProperties": False, "required": ["answers"],
        "properties": {"answers": {"type": "object", "additionalProperties": False,
                                   "required": list(expected),
                                   "properties": {key: {"type": "string"} for key in expected}}},
    }


def public_results(plan, records):
    plan_fields = ("tasks", "revision", "model", "repeats", "gha_sha256")
    run_fields = ("task_id", "repetition", "condition", "valid", "correct", "gha_used",
                  "gha_workflow_used",
                  "inspected_repository",
                  "seconds", "input_tokens", "cached_input_tokens", "output_tokens", "total_tokens")
    return {
        "schema_version": "v1",
        "plan": {key: plan.get(key) for key in plan_fields},
        "summary": summarize(records),
        "runs": [{key: record.get(key) for key in run_fields} for record in records],
    }


def make_fixture(checkout, fixture):
    if fixture is None:
        return
    if fixture != "branch_cleanup":
        raise ValueError("unknown fixture: " + fixture)
    def git(*args):
        subprocess.run(["git", "-C", str(checkout), *args], check=True, capture_output=True)
    git("branch", "eval/base", "HEAD")
    git("branch", "feature/merged", "HEAD~1")
    for branch in ("feature/active", "feature/old-unmerged"):
        git("checkout", "--quiet", "-b", branch, "eval/base~1")
        git("-c", "user.name=GHA Eval", "-c", "user.email=eval@example.invalid",
            "commit", "--quiet", "--allow-empty", "-m", branch)
    git("checkout", "--quiet", "--detach", "eval/base")
    git("update-ref", "refs/remotes/origin/feature/active", "eval/base~1")
    git("branch", "--set-upstream-to", "origin/feature/active", "feature/active")


def run_one(task, condition, source, revision, gha_binary, model, timeout, trace_path):
    with tempfile.TemporaryDirectory(prefix="gha-eval-") as directory:
        root = Path(directory)
        checkout = root / "repo"
        subprocess.run(["git", "clone", "--quiet", "--no-hardlinks", str(source), str(checkout)], check=True)
        subprocess.run(["git", "-C", str(checkout), "checkout", "--quiet", "--detach", revision], check=True)
        make_fixture(checkout, task.get("fixture"))
        # Both arms see identical neutral project guidance; otherwise the
        # repository's GHA-specific AGENTS.md biases the control arm.
        (checkout / "AGENTS.md").write_text("Follow the evaluation task prompt. Do not modify files.\n")
        schema = root / "schema.json"
        schema.write_text(json.dumps(output_schema(task["expected"])))
        answer = root / "answer.json"
        codex_home = root / "codex-home"
        codex_home.mkdir(mode=0o700)
        auth_source = Path(os.environ.get("CODEX_HOME", str(Path.home() / ".codex"))) / "auth.json"
        if auth_source.is_file():
            (codex_home / "auth.json").symlink_to(auth_source)
        access = (f"Use GHA's {task['gha_workflow']} workflow for this task. Its executable is {gha_binary}. "
                  f"Discover supported commands with {gha_binary} capabilities --format json. "
                  if condition == "gha" else
                  "For this control run, use git, gh, and source inspection only. Do not run GHA. ")
        prompt = ("This is a read-only repository evaluation. Inspect actual repository state with commands; "
                  "do not guess or infer from branch names. Do not change any files or contact a remote. "
                  + access + task["prompt"] + " Return only the requested JSON answers.")
        command = ["codex", "exec", "--json", "--ephemeral", "--ignore-user-config", "-s", "read-only",
                   "-c", "model_reasoning_effort=low",
                   "-m", model, "-C", str(checkout), "--output-schema", str(schema),
                   "-o", str(answer), prompt]
        start = time.monotonic()
        try:
            completed = subprocess.run(command, capture_output=True, text=True, timeout=timeout,
                                       check=False, env={**os.environ, "CODEX_HOME": str(codex_home)})
            elapsed = time.monotonic() - start
            trace_path.write_text(completed.stdout)
            os.chmod(trace_path, 0o600)
            events = []
            for line in completed.stdout.splitlines():
                try:
                    events.append(json.loads(line))
                except ValueError:
                    pass
            usage = usage_from_events(events)
            gha_used = used_gha(events)
            workflow_used = used_gha_workflow(events, gha_binary, task["gha_workflow"])
            inspected = inspected_repository(events)
            violation = condition == "control" and gha_used
            message = answer.read_text() if answer.exists() else ""
            valid = completed.returncode == 0 and bool(message) and inspected and not violation and (
                condition != "gha" or workflow_used)
            return {"valid": valid, "correct": valid and grade(message, task["expected"]),
                    "seconds": round(elapsed, 3), "exit_code": completed.returncode,
                    "gha_used": gha_used, "gha_workflow_used": workflow_used,
                    "inspected_repository": inspected,
                    "control_used_gha": violation,
                    "input_tokens": usage[0] if usage else None,
                    "cached_input_tokens": usage[1] if usage else None,
                    "output_tokens": usage[2] if usage else None,
                    "total_tokens": usage[0] + usage[2] if usage else None,
                    "answers": message, "stderr_tail": completed.stderr[-1000:]}
        except subprocess.TimeoutExpired:
            return {"valid": False, "correct": False, "seconds": round(time.monotonic() - start, 3),
                    "exit_code": None, "gha_used": False, "gha_workflow_used": False,
                    "inspected_repository": False,
                    "control_used_gha": False,
                    "input_tokens": None,
                    "cached_input_tokens": None, "output_tokens": None, "total_tokens": None,
                    "answers": "", "stderr_tail": "timeout"}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="action", required=True)
    run = sub.add_parser("run")
    run.add_argument("--tasks", type=Path, required=True)
    run.add_argument("--repo", type=Path, default=Path(__file__).resolve().parents[2])
    run.add_argument("--revision", required=True, help="Pinned commit SHA")
    run.add_argument("--gha", type=Path, required=True, help="Built GHA executable")
    run.add_argument("--model", required=True)
    run.add_argument("--repeats", type=int, default=2)
    run.add_argument("--timeout", type=int, default=600)
    run.add_argument("--out", type=Path, default=Path("/tmp/gha-agent-eval"))
    run.add_argument("--execute", action="store_true", help="Run agents and spend tokens")
    report = sub.add_parser("report")
    report.add_argument("--input", type=Path, required=True)
    report.add_argument("--public-output", type=Path,
                        help="Write a sanitized, shareable JSON result without traces or answers")
    args = parser.parse_args()
    if args.action == "report":
        records = [json.loads(line) for line in args.input.read_text().splitlines() if line.strip()]
        summary = summarize(records)
        if args.public_output:
            plan = json.loads((args.input.parent / "plan.json").read_text())
            args.public_output.parent.mkdir(parents=True, exist_ok=True)
            args.public_output.write_text(json.dumps(public_results(plan, records), indent=2) + "\n")
        print(json.dumps(summary, indent=2))
        return
    tasks = load_tasks(args.tasks)
    if args.repeats < 2 or args.repeats % 2 or args.timeout < 1:
        parser.error("repeats must be a positive even number (at least 2); timeout must be positive")
    source = args.repo.resolve()
    revision = subprocess.check_output(["git", "-C", str(source), "rev-parse", "--verify",
                                        args.revision + "^{commit}"], text=True).strip()
    for task in tasks:
        if task.get("revision") and task["revision"] != revision:
            parser.error(f"task {task['id']} requires revision {task['revision']}")
    gha_binary = args.gha.resolve()
    if not gha_binary.is_file():
        parser.error("GHA executable does not exist; run make build")
    plan = {"tasks": [task["id"] for task in tasks], "revision": revision, "model": args.model,
            "conditions": ["control", "gha"], "repeats": args.repeats,
            "total_runs": len(tasks) * 2 * args.repeats, "output": str(args.out.resolve()),
            "gha_sha256": hashlib.sha256(gha_binary.read_bytes()).hexdigest()}
    print(json.dumps(plan, indent=2))
    if not args.execute:
        return
    args.out.mkdir(parents=True, exist_ok=True)
    os.chmod(args.out, 0o700)
    (args.out / "plan.json").write_text(json.dumps(plan, indent=2) + "\n")
    records_path = args.out / "runs.jsonl"
    if records_path.exists():
        parser.error("output already contains runs.jsonl; choose a new output directory")
    records = []
    with records_path.open("w") as stream:
        for repetition in range(args.repeats):
            for task in tasks:
                for condition in (("control", "gha") if repetition % 2 == 0 else ("gha", "control")):
                    trace = args.out / f"{task['id']}-{repetition}-{condition}.jsonl"
                    record = {"task_id": task["id"], "repetition": repetition, "condition": condition,
                              "trace": trace.name}
                    record.update(run_one(task, condition, source, revision, gha_binary, args.model,
                                          args.timeout, trace))
                    stream.write(json.dumps(record) + "\n")
                    stream.flush()
                    records.append(record)
                    print(f"{task['id']} {repetition} {condition}: "
                          f"{'correct' if record['correct'] else 'failed'}")
    print(json.dumps(summarize(records), indent=2))


if __name__ == "__main__":
    main()
