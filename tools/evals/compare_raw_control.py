#!/usr/bin/env python3
"""Combine saved no-GHA runs with the earlier short-guide GHA sweep."""

import argparse
import json
from pathlib import Path


LOCAL = ("repository-diagnosis", "branch-change-plans", "tag-publication-plan")
PROVIDER = ("pull-request-inspection", "pull-request-creation-plan",
            "release-research", "release-publication-plan")
NEUTRAL_SUBSET = ("repository-diagnosis", "pull-request-inspection", "release-research")


def selected(data, condition, task_ids):
    result = {}
    for run in data["runs"]:
        if run["condition"] != condition or run["task_id"] not in task_ids:
            continue
        key = (run["task_id"], run["repetition"])
        if key in result:
            raise ValueError(f"duplicate attempt: {key}")
        result[key] = run
    return result


def totals(runs):
    fields = ("input_tokens", "cached_input_tokens", "output_tokens", "total_tokens",
              "command_count", "command_chars")
    result = {field: sum(run[field] for run in runs) if all(
              isinstance(run.get(field), int) for run in runs) else None for field in fields}
    result.update(runs=len(runs), valid=sum(bool(run["valid"]) for run in runs),
                  correct=sum(bool(run["correct"]) for run in runs),
                  mean_seconds=round(sum(run["seconds"] for run in runs) / len(runs), 3))
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--all-tasks", type=Path, required=True)
    parser.add_argument("--control-tasks", type=Path, required=True)
    parser.add_argument("--gha", type=Path, required=True)
    parser.add_argument("--local-control", type=Path, required=True)
    parser.add_argument("--provider-control", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()

    all_tasks = json.loads(args.all_tasks.read_text())
    control_tasks = json.loads(args.control_tasks.read_text())
    task_ids = LOCAL + PROVIDER
    if control_tasks != [task for task in all_tasks if task["id"] in task_ids]:
        parser.error("control tasks differ from original GHA tasks")
    gha = json.loads(args.gha.read_text())
    local = json.loads(args.local_control.read_text())
    provider = json.loads(args.provider_control.read_text())
    plans = (gha["plan"], local["plan"], provider["plan"])
    for key in ("revision", "model", "repeats", "gha_sha256"):
        if len({json.dumps(plan[key]) for plan in plans}) != 1:
            parser.error(f"plans differ on {key}")
    if provider["plan"].get("strict_gh") is not True:
        parser.error("provider control was not strict git/gh")

    raw = selected(local, "raw_control", LOCAL) | selected(provider, "raw_control", PROVIDER)
    guided = selected(gha, "short_guide", task_ids)
    expected_keys = {(task_id, repetition) for task_id in task_ids
                     for repetition in range(gha["plan"]["repeats"])}
    if set(raw) != expected_keys or set(guided) != expected_keys:
        parser.error("missing or duplicate scenario repetitions")
    for key, run in raw.items():
        if (run["gha_used"] or run.get("direct_http_used") or
                run["unsafe_write_attempt"] or not run["refs_unchanged"] or
                not run["tree_unchanged"]):
            parser.error(f"control policy violation in {key}")

    scenarios = {}
    for task_id in task_ids:
        keys = [(task_id, repetition) for repetition in range(gha["plan"]["repeats"])]
        scenarios[task_id] = {"git_gh": totals([raw[key] for key in keys]),
                              "short_guide": totals([guided[key] for key in keys])}
    result = {
        "schema_version": "v1",
        "revision": gha["plan"]["revision"],
        "model": gha["plan"]["model"],
        "repeats": gha["plan"]["repeats"],
        "task_ids": list(task_ids),
        "neutral_subset": list(NEUTRAL_SUBSET),
        "sources": {"short_guide": args.gha.name,
                    "local_control": args.local_control.name,
                    "provider_control": args.provider_control.name},
        "overall": {"git_gh": totals(list(raw.values())),
                    "short_guide": totals(list(guided.values()))},
        "neutral_subset_totals": {
            "git_gh": totals([run for key, run in raw.items() if key[0] in NEUTRAL_SUBSET]),
            "short_guide": totals([run for key, run in guided.items()
                                   if key[0] in NEUTRAL_SUBSET])},
        "scenarios": scenarios,
    }
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(result, indent=2) + "\n")
    print(json.dumps(result["overall"], indent=2))


if __name__ == "__main__":
    main()
