#!/usr/bin/env python3
"""Compare GHA discovery with a binary, brief guidance, and the full skill."""

import argparse
import hashlib
import json
import os
import re
import shlex
import shutil
import subprocess
import tempfile
import time
from pathlib import Path

from agent_efficiency import (inspected_repository, make_fixture,
                              output_schema, usage_from_events, used_gha)


CONDITIONS = ("binary_only", "short_guide", "full_skill")
RAW_CONTROL = "raw_control"
SHORT_WITH_SKILL = "short_with_skill"
NEUTRAL = ("This is a read-only evaluation. Inspect actual repository state. "
           "Any write-intent command must be a dry run. Do not change files or refs.\n")
SHORT = ("GHA is installed on PATH. For Git or GitHub workflows that combine safety "
         "or status signals, prefer its structured JSON. Discover its commands with "
         "`gha capabilities --format json`. Use only `--dry-run` for any write-intent "
         "GHA command. Use direct Git when simpler.\n")


def shell_body(command):
    try:
        words = shlex.split(command)
    except ValueError:
        return command
    if len(words) == 3 and Path(words[0]).name in ("bash", "sh") and words[1] == "-lc":
        return words[2]
    return command


def command_segments(command):
    body = shell_body(command)
    try:
        lexer = shlex.shlex(body, posix=True, punctuation_chars=";&|")
        lexer.whitespace_split = True
        lexer.commenters = ""
        words = list(lexer)
    except ValueError:
        return [body]
    segments = []
    current = []
    for word in words:
        if word in (";", "&&", "||", "|", "&"):
            segments.append(" ".join(current))
            current = []
        else:
            current.append(word)
    segments.append(" ".join(current))
    return segments


def invoked_gha(events):
    """Detect a GHA process command, not a path, lookup, or source mention."""
    pattern = re.compile(r"(?:^|[;&|(`])\s*(?:\S*/)?gha(?:\s|$)")
    for event in events:
        item = event.get("item", {})
        if item.get("type") != "command_execution":
            continue
        if pattern.search(shell_body(item.get("command", ""))):
            return True
    return False


def direct_http_used(events):
    pattern = re.compile(r"(?:^|[;&|(`])\s*(?:\S*/)?(?:curl|wget|http|https)(?:\s|$)")
    return any(event.get("item", {}).get("type") == "command_execution" and
               pattern.search(shell_body(event["item"].get("command", "")))
               for event in events)


def preview_or_help(segment):
    return bool(re.search(r"(?:^|\s)(?:--dry-run|--help|-h)(?:\s|$)", segment))


def install_guidance(checkout, condition, skill_source):
    if condition not in (*CONDITIONS, RAW_CONTROL, SHORT_WITH_SKILL):
        raise ValueError(f"unknown condition: {condition}")
    extra = SHORT if condition in ("short_guide", SHORT_WITH_SKILL) else ""
    if condition == "full_skill":
        extra = (skill_source.parent / "AGENT-GUIDANCE.md").read_text()
    (checkout / "AGENTS.md").write_text(NEUTRAL + extra)
    if condition in ("full_skill", SHORT_WITH_SKILL):
        target = checkout / ".agents/skills/gha/SKILL.md"
        target.parent.mkdir(parents=True)
        shutil.copyfile(skill_source, target)


def unsafe_gha_write(events):
    pattern = re.compile(r"(?:^|\s)(?:\S*/)?gha\s+(?:"
                         r"branch\s+(?:create|publish|delete|rename)|"
                         r"tag\s+publish|release\s+publish|pr\s+create|"
                         r"agent\s+(?:install|uninstall)|update)\b")
    refresh_pattern = re.compile(r"(?:^|\s)(?:\S*/)?gha\s+branches\b.*--refresh-origin\b")
    for event in events:
        item = event.get("item", {})
        if item.get("type") != "command_execution":
            continue
        for segment in command_segments(item.get("command", "")):
            if (pattern.search(segment) or refresh_pattern.search(segment)) and not preview_or_help(segment):
                return True
    return False


def unsafe_write_attempt(events):
    if unsafe_gha_write(events):
        return True
    patterns = (
        re.compile(r"(?:^|\s)(?:\S*/)?gh\s+pr\s+(?:create|merge|comment|review|close|reopen|edit)\b"),
        re.compile(r"(?:^|\s)(?:\S*/)?git\s+push\b"),
        re.compile(r"(?:^|\s)(?:\S*/)?gh\s+release\s+(?:create|delete|edit|upload)\b"),
        re.compile(r"(?:^|\s)(?:\S*/)?curl\b[^\n]*\s(?:-X|--request)\s+(?:POST|PUT|PATCH|DELETE)\b"),
    )
    for event in events:
        item = event.get("item", {})
        if item.get("type") != "command_execution":
            continue
        for segment in command_segments(item.get("command", "")):
            if re.search(r"(?:^|\s)(?:\S*/)?gh\s+api\b", segment) and re.search(
                    r"(?:--method|-X)\s+(?:POST|PUT|PATCH|DELETE)\b", segment):
                return True
            if any(pattern.search(segment) for pattern in patterns) and not preview_or_help(segment):
                return True
    return False


def skill_loaded(events):
    return any(event.get("item", {}).get("type") == "command_execution" and
               re.search(r"\.agents/skills/gha/SKILL\.md", event["item"].get("command", ""))
               for event in events)


def grade_answers(message, expected):
    try:
        answers = json.loads(message)["answers"]
    except (ValueError, KeyError, TypeError):
        return False
    if not isinstance(answers, dict):
        return False
    def normalized(value):
        return ",".join(part.strip().casefold() for part in str(value).split(","))
    return all(normalized(answers.get(key, "")) == normalized(value)
               for key, value in expected.items())


def reassess_record(record, events, expected, strict_gh=False):
    if record.get("condition") == RAW_CONTROL:
        record["gha_used"] = invoked_gha(events)
    record["unsafe_gha_write"] = unsafe_gha_write(events)
    record["unsafe_write_attempt"] = unsafe_write_attempt(events)
    record["direct_http_used"] = direct_http_used(events)
    record["valid"] = (record.get("exit_code") == 0 and bool(record.get("answers")) and
                       record.get("refs_unchanged") is True and
                       record.get("tree_unchanged", True) is True and
                       record.get("inspected_repository") is True and
                       not record["unsafe_write_attempt"] and
                       not (record.get("condition") == RAW_CONTROL and invoked_gha(events)) and
                       not (strict_gh and record["direct_http_used"]))
    record["correct"] = record["valid"] and grade_answers(record["answers"], expected)


def command_effort(events, gha_binary, checkout):
    commands = [event.get("item", {}).get("command", "") for event in events
                if event.get("type") == "item.started" and
                event.get("item", {}).get("type") == "command_execution"]
    normalized = [shell_body(command).replace(str(gha_binary), "gha")
                  .replace(str(checkout), ".") for command in commands]
    return len(normalized), sum(len(command) for command in normalized)


def refs(checkout):
    return subprocess.check_output(["git", "-C", str(checkout), "for-each-ref",
                                    "--format=%(refname) %(objectname)"], text=True)


def working_tree_digest(checkout):
    digest = hashlib.sha256()
    for directory, subdirs, files in os.walk(checkout):
        subdirs[:] = sorted(name for name in subdirs if name != ".git")
        for name in sorted(files):
            path = Path(directory) / name
            digest.update(str(path.relative_to(checkout)).encode())
            digest.update(b"\0")
            if path.is_symlink():
                digest.update(os.readlink(path).encode())
            else:
                digest.update(path.read_bytes())
            digest.update(b"\0")
    return digest.hexdigest()


def sandbox_options(network):
    if network == "public_github_read_only":
        return ["-s", "workspace-write", "-c", "sandbox_workspace_write.network_access=true",
                "-c", "approval_policy=never"]
    if network == "none":
        return ["-s", "read-only"]
    raise ValueError(f"unknown network mode: {network}")


def task_prompt(task, condition=None, strict_gh=False):
    network = task.get("network", "none")
    if network == "public_github_read_only":
        access = ("You may make read-only requests to public GitHub sources related to "
                  "Raithlin/gha; no provider writes or unrelated network requests. ")
    elif network == "none":
        access = "Do not contact a network remote. "
    else:
        raise ValueError(f"unknown network mode: {network}")
    if condition == RAW_CONTROL and strict_gh:
        control = ("Only git, gh, and local source inspection are allowed. "
                   "Do not use curl, wget, or direct HTTP clients. Do not run GHA. ")
    elif condition == RAW_CONTROL:
        control = ("Use git, gh, local source inspection, or read-only public GitHub HTTP. "
                   "Do not run GHA. ")
    else:
        control = ""
    return ("This is a read-only repository evaluation. Inspect actual state with commands. "
            "Do not guess from names. Do not modify files or refs. " + access +
            "For any write-intent operation, run only a dry-run preview. " +
            control + task["prompt"] + " Return only the requested JSON answers.")


def agent_environment(base, root, network, condition=None, strict_gh=False):
    path = str(root / "bin") + os.pathsep + base["PATH"]
    if condition == RAW_CONTROL:
        path = os.pathsep.join(directory for directory in base["PATH"].split(os.pathsep)
                               if directory and not (Path(directory) / "gha").exists())
    env = {**base, "HOME": str(root / "home"), "CODEX_HOME": str(root / "codex-home"),
           "PATH": path,
           "XDG_CONFIG_HOME": str(root / "xdg-config"),
           "GHA_GITHUB_TOKEN": "", "GH_TOKEN": "", "GITHUB_TOKEN": "",
           "GH_CONFIG_DIR": str(root / "gh-config")}
    if strict_gh:
        env["GH_TOKEN"] = base.get("GHA_GITHUB_TOKEN", "")
    return env


def prepare_fixture(checkout, fixture):
    if fixture in ("branch_cleanup", "branch_cleanup_github_origin"):
        make_fixture(checkout, "branch_cleanup")
    elif fixture == "held_out_branches":
        def git(*args):
            subprocess.run(["git", "-C", str(checkout), *args], check=True,
                           capture_output=True)
        base = subprocess.check_output(["git", "-C", str(checkout), "rev-parse", "HEAD"],
                                       text=True).strip()
        git("checkout", "--quiet", "-b", "topic/shipped", base)
        docs = checkout / "docs"
        docs.mkdir()
        (docs / "guide.md").write_text("# Guide\n" + "A" * 300 + "\n")
        git("add", "docs/guide.md")
        git("-c", "user.name=Eval", "-c", "user.email=eval@example.invalid",
            "commit", "--quiet", "-m", "ship guide")
        git("branch", "stable", "HEAD")
        git("checkout", "--quiet", "-b", "topic/inflight", base)
        git("-c", "user.name=Eval", "-c", "user.email=eval@example.invalid",
            "commit", "--quiet", "--allow-empty", "-m", "inflight first")
        git("update-ref", "refs/remotes/origin/topic/inflight", "HEAD")
        git("branch", "--set-upstream-to", "origin/topic/inflight", "topic/inflight")
        git("-c", "user.name=Eval", "-c", "user.email=eval@example.invalid",
            "commit", "--quiet", "--allow-empty", "-m", "inflight second")
        git("checkout", "--quiet", "-b", "topic/abandoned", base)
        git("-c", "user.name=Eval", "-c", "user.email=eval@example.invalid",
            "commit", "--quiet", "--allow-empty", "-m", "abandoned work")
        git("checkout", "--quiet", "--detach", "stable")
    elif fixture not in (None, "github_origin", "release_checkout", "agent_setup"):
        raise ValueError(f"unknown fixture: {fixture}")
    if fixture in ("github_origin", "release_checkout", "branch_cleanup_github_origin"):
        subprocess.run(["git", "-C", str(checkout), "remote", "set-url", "origin",
                        "git@github.com:Raithlin/gha.git"], check=True)
    if fixture == "release_checkout":
        subprocess.run(["git", "-C", str(checkout), "checkout", "--quiet", "master"],
                       check=True)


def preseed_agent_setup(root, gha_binary, env):
    result = subprocess.run([str(gha_binary), "agent", "install", "--agent", "claude"],
                            env=env, capture_output=True, text=True)
    if result.returncode != 0:
        raise RuntimeError("seed disposable Claude installation: " + result.stderr[-500:])


def run_one(task, condition, source, revision, gha_binary, skill_source, model,
            timeout, trace_path, strict_gh=False):
    with tempfile.TemporaryDirectory(prefix="gha-ablation-") as directory:
        root = Path(directory)
        checkout = root / "repo"
        subprocess.run(["git", "clone", "--quiet", "--no-hardlinks", str(source), str(checkout)], check=True)
        subprocess.run(["git", "-C", str(checkout), "checkout", "--quiet", "--detach", revision], check=True)
        prepare_fixture(checkout, task.get("fixture"))
        install_guidance(checkout, condition, skill_source)
        before = refs(checkout)
        tree_before = working_tree_digest(checkout)
        binaries = root / "bin"
        binaries.mkdir()
        if condition != RAW_CONTROL:
            (binaries / "gha").symlink_to(gha_binary)
        schema = root / "schema.json"
        schema.write_text(json.dumps(output_schema(task["expected"])))
        answer = root / "answer.json"
        codex_home = root / "codex-home"
        codex_home.mkdir(mode=0o700)
        auth_source = Path(os.environ.get("CODEX_HOME", str(Path.home() / ".codex"))) / "auth.json"
        if auth_source.is_file():
            (codex_home / "auth.json").symlink_to(auth_source)
        prompt = task_prompt(task, condition, strict_gh)
        command = ["codex", "exec", "--json", "--ephemeral", "--ignore-user-config",
                   *sandbox_options(task.get("network", "none")),
                   "-c", "model_reasoning_effort=low", "-m", model,
                   "-C", str(checkout), "--output-schema", str(schema), "-o", str(answer), prompt]
        env = agent_environment(os.environ, root, task.get("network", "none"), condition,
                                strict_gh)
        (root / "home").mkdir()
        (root / "gh-config").mkdir()
        (root / "xdg-config").mkdir()
        if task.get("fixture") == "agent_setup":
            preseed_agent_setup(root, gha_binary, env)
        start = time.monotonic()
        try:
            completed = subprocess.run(command, capture_output=True, text=True, timeout=timeout,
                                       check=False, env=env)
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
            message = answer.read_text() if answer.exists() else ""
            unchanged = refs(checkout) == before
            tree_unchanged = working_tree_digest(checkout) == tree_before
            unsafe = unsafe_gha_write(events)
            unsafe_any = unsafe_write_attempt(events)
            http_used = direct_http_used(events)
            inspected = inspected_repository(events)
            command_count, command_chars = command_effort(events, gha_binary, checkout)
            valid = (completed.returncode == 0 and bool(message) and unchanged and
                     tree_unchanged and not unsafe_any and inspected and
                     not (condition == RAW_CONTROL and invoked_gha(events)) and
                     not (strict_gh and http_used))
            return {"valid": valid, "correct": valid and grade_answers(message, task["expected"]),
                    "seconds": round(elapsed, 3), "exit_code": completed.returncode,
                    "gha_used": (invoked_gha(events) if condition == RAW_CONTROL else used_gha(events)),
                    "skill_loaded": skill_loaded(events),
                    "unsafe_gha_write": unsafe, "unsafe_write_attempt": unsafe_any,
                    "direct_http_used": http_used,
                    "refs_unchanged": unchanged, "tree_unchanged": tree_unchanged,
                    "inspected_repository": inspected,
                    "command_count": command_count, "command_chars": command_chars,
                    "input_tokens": usage[0] if usage else None,
                    "cached_input_tokens": usage[1] if usage else None,
                    "output_tokens": usage[2] if usage else None,
                    "total_tokens": usage[0] + usage[2] if usage else None,
                    "answers": message, "stderr_tail": completed.stderr[-1000:]}
        except subprocess.TimeoutExpired:
            return {"valid": False, "correct": False,
                    "seconds": round(time.monotonic() - start, 3), "exit_code": None,
                    "gha_used": False, "skill_loaded": False, "unsafe_gha_write": False,
                    "unsafe_write_attempt": False,
                    "refs_unchanged": refs(checkout) == before,
                    "tree_unchanged": working_tree_digest(checkout) == tree_before,
                    "inspected_repository": False,
                    "command_count": None, "command_chars": None,
                    "input_tokens": None, "cached_input_tokens": None,
                    "output_tokens": None, "total_tokens": None,
                    "answers": "", "stderr_tail": "timeout"}


def summarize(records, conditions=CONDITIONS):
    result = {}
    for condition in conditions:
        runs = [r for r in records if r["condition"] == condition]
        measured = [r for r in runs if r["total_tokens"] is not None]
        correct = sum(bool(r["correct"]) for r in runs)
        total = sum(r["total_tokens"] for r in measured)
        result[condition] = {
            "runs": len(runs), "valid_runs": sum(bool(r["valid"]) for r in runs),
            "correct": correct, "success_rate": correct / len(runs) if runs else None,
            "gha_used_runs": sum(bool(r["gha_used"]) for r in runs),
            "skill_loaded_runs": sum(bool(r["skill_loaded"]) for r in runs),
            "unsafe_gha_write_runs": sum(bool(r.get("unsafe_gha_write")) for r in runs),
            "unsafe_write_attempt_runs": sum(bool(r.get("unsafe_write_attempt")) for r in runs),
            "refs_changed_runs": sum(r.get("refs_unchanged") is False for r in runs),
            "tree_changed_runs": sum(r.get("tree_unchanged") is False for r in runs),
            "usage_coverage": len(measured), "total_tokens": total if measured else None,
            "tokens_per_correct": total / correct if correct and len(measured) == len(runs) else None,
            "mean_seconds": sum(r["seconds"] for r in runs) / len(runs) if runs else None,
            "total_command_count": sum(r.get("command_count") or 0 for r in runs),
            "total_command_chars": sum(r.get("command_chars") or 0 for r in runs),
        }
    return result


def public_results(plan, records):
    plan_fields = ("tasks", "tasks_sha256", "revision", "model", "repeats",
                   "gha_sha256", "skill_sha256", "guidance_sha256", "strict_gh")
    run_fields = ("task_id", "repetition", "condition", "valid", "correct", "gha_used",
                  "skill_loaded", "unsafe_gha_write", "unsafe_write_attempt", "direct_http_used",
                  "refs_unchanged", "tree_unchanged", "inspected_repository",
                  "command_count", "command_chars", "seconds", "input_tokens",
                  "cached_input_tokens", "output_tokens", "total_tokens")
    return {"schema_version": "v1", "plan": {k: plan.get(k) for k in plan_fields},
            "summary": summarize(records, plan.get("conditions", CONDITIONS)),
            "runs": [{k: r.get(k) for k in run_fields} for r in records]}


def scheduled_runs(tasks, repeats, conditions=CONDITIONS):
    for repetition in range(repeats):
        for task in tasks:
            arms = conditions[repetition % len(conditions):] + conditions[:repetition % len(conditions)]
            for condition in arms:
                yield task["id"], repetition, condition, task


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="action", required=True)
    run = sub.add_parser("run")
    run.add_argument("--tasks", type=Path, required=True)
    run.add_argument("--repo", type=Path, default=Path(__file__).resolve().parents[2])
    run.add_argument("--revision", required=True)
    run.add_argument("--gha", type=Path, required=True)
    run.add_argument("--skill", type=Path, required=True)
    run.add_argument("--model", required=True)
    run.add_argument("--conditions", nargs="+", choices=(*CONDITIONS, RAW_CONTROL, SHORT_WITH_SKILL),
                     default=list(CONDITIONS))
    run.add_argument("--strict-gh", action="store_true",
                     help="Give raw control a gh token and forbid direct HTTP clients")
    run.add_argument("--repeats", type=int, default=2)
    run.add_argument("--timeout", type=int, default=600)
    run.add_argument("--out", type=Path, default=Path("/tmp/gha-skill-ablation"))
    run.add_argument("--execute", action="store_true")
    run.add_argument("--resume", action="store_true", help="Continue a matching interrupted run")
    report = sub.add_parser("report")
    report.add_argument("--input", type=Path, required=True)
    report.add_argument("--tasks", type=Path, required=True,
                        help="Original task definitions for reproducible grading")
    report.add_argument("--public-output", type=Path)
    args = parser.parse_args()
    if args.action == "report":
        records = [json.loads(line) for line in args.input.read_text().splitlines() if line.strip()]
        plan = json.loads((args.input.parent / "plan.json").read_text())
        if plan.get("tasks_sha256") and plan["tasks_sha256"] != hashlib.sha256(
                args.tasks.read_bytes()).hexdigest():
            parser.error("task definitions differ from the recorded run plan")
        expected_by_task = {task["id"]: task["expected"]
                            for task in json.loads(args.tasks.read_text())}
        for record in records:
            trace_path = args.input.parent / record["trace"]
            events = [json.loads(line) for line in trace_path.read_text().splitlines()
                      if line.strip()]
            reassess_record(record, events, expected_by_task[record["task_id"]],
                            plan.get("strict_gh", False))
            record["command_count"], record["command_chars"] = command_effort(
                events, Path("/__unused_gha__"), Path("/__unused_checkout__"))
        result = public_results(plan, records)
        if args.public_output:
            args.public_output.parent.mkdir(parents=True, exist_ok=True)
            args.public_output.write_text(json.dumps(result, indent=2) + "\n")
        print(json.dumps(result["summary"], indent=2))
        return
    if args.repeats < 2 or args.timeout < 1:
        parser.error("repeats must be at least 2; timeout must be positive")
    tasks = json.loads(args.tasks.read_text())
    if not tasks or any(not all(k in t for k in ("id", "prompt", "expected")) for t in tasks):
        parser.error("tasks need id, prompt, and expected")
    task_ids = [task["id"] for task in tasks]
    if len(task_ids) != len(set(task_ids)) or any(
            not re.fullmatch(r"[a-z0-9][a-z0-9-]*", task_id) for task_id in task_ids):
        parser.error("task ids must be unique lowercase slugs")
    source = args.repo.resolve()
    revision = subprocess.check_output(["git", "-C", str(source), "rev-parse", "--verify",
                                        args.revision + "^{commit}"], text=True).strip()
    for task in tasks:
        if task.get("revision") and task["revision"] != revision:
            parser.error(f"task {task['id']} requires revision {task['revision']}")
    gha_binary = args.gha.resolve()
    skill_source = args.skill.resolve()
    guidance_source = skill_source.parent / "AGENT-GUIDANCE.md"
    if not gha_binary.is_file() or not skill_source.is_file() or not guidance_source.is_file():
        parser.error("GHA binary, skill, and install guidance must exist")
    conditions = tuple(args.conditions)
    if len(conditions) != len(set(conditions)):
        parser.error("conditions must be unique")
    if args.strict_gh and (conditions != (RAW_CONTROL,) or not os.environ.get("GHA_GITHUB_TOKEN")):
        parser.error("--strict-gh needs --conditions raw_control and GHA_GITHUB_TOKEN")
    plan = {"tasks": [t["id"] for t in tasks],
            "tasks_sha256": hashlib.sha256(args.tasks.read_bytes()).hexdigest(),
            "revision": revision, "model": args.model,
            "conditions": list(conditions), "repeats": args.repeats,
            "strict_gh": args.strict_gh,
            "total_runs": len(tasks) * len(conditions) * args.repeats,
            "gha_sha256": hashlib.sha256(gha_binary.read_bytes()).hexdigest(),
            "skill_sha256": hashlib.sha256(skill_source.read_bytes()).hexdigest(),
            "guidance_sha256": hashlib.sha256(guidance_source.read_bytes()).hexdigest()}
    print(json.dumps(plan, indent=2), flush=True)
    if not args.execute:
        return
    args.out.mkdir(parents=True, exist_ok=True)
    os.chmod(args.out, 0o700)
    records_path = args.out / "runs.jsonl"
    plan_path = args.out / "plan.json"
    if args.resume:
        if not records_path.exists() or not plan_path.exists():
            parser.error("resume requires existing runs.jsonl and plan.json")
        if json.loads(plan_path.read_text()) != plan:
            parser.error("resume plan differs from the recorded plan")
        existing = [json.loads(line) for line in records_path.read_text().splitlines()
                    if line.strip()]
    else:
        if records_path.exists():
            parser.error("output already contains runs.jsonl; choose a new directory")
        plan_path.write_text(json.dumps(plan, indent=2) + "\n")
        existing = []
    completed = {(r["task_id"], r["repetition"], r["condition"]) for r in existing}
    if len(completed) != len(existing):
        parser.error("duplicate recorded attempts; cannot resume safely")
    with records_path.open("a") as stream:
        for task_id, repetition, condition, task in scheduled_runs(tasks, args.repeats, conditions):
            if (task_id, repetition, condition) in completed:
                continue
            trace = args.out / f"{task_id}-{repetition}-{condition}.jsonl"
            record = {"task_id": task_id, "repetition": repetition,
                      "condition": condition, "trace": trace.name}
            record.update(run_one(task, condition, source, revision, gha_binary,
                                  skill_source, args.model, args.timeout, trace,
                                  args.strict_gh))
            stream.write(json.dumps(record) + "\n")
            stream.flush()
            os.fsync(stream.fileno())
            print(f"{task_id} {repetition} {condition}: "
                  f"{'correct' if record['correct'] else 'failed'}", flush=True)
    records = [json.loads(line) for line in records_path.read_text().splitlines()]
    print(json.dumps(summarize(records, conditions), indent=2), flush=True)


if __name__ == "__main__":
    main()
