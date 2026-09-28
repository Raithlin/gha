#!/usr/bin/env python3
"""Safely exercise every available GHA command in disposable fixtures."""

import argparse
import datetime
import hashlib
import json
import os
import subprocess
import tempfile
from pathlib import Path

from skill_ablation import prepare_fixture, preseed_agent_setup, refs, working_tree_digest


def probe(capability, fixture, args, write_intent=False):
    return {"capability": capability, "fixture": fixture, "args": args,
            "write_intent": write_intent}


PROBES = (
    probe("capabilities", "local", ["capabilities", "--format", "json"]),
    probe("version", "local", ["version", "--format", "json"]),
    probe("agent list", "local", ["agent", "list", "--format", "json"]),
    probe("agent install", "local", ["agent", "install", "--agent", "gemini", "--dry-run"], True),
    probe("agent uninstall", "local", ["agent", "uninstall", "--agent", "claude", "--dry-run"], True),
    probe("update", "local", ["update", "--dry-run", "--format", "json"], True),
    probe("analyze", "local", ["analyze", "--limit", "5", "--format", "json"]),
    probe("branches", "local", ["branches", "--limit", "10", "--format", "json"]),
    probe("branches cleanup", "local", ["branches", "cleanup", "--base", "eval/base", "--format", "json"]),
    probe("branch show <name>", "local", ["branch", "show", "feature/active", "--format", "json"]),
    probe("branch create <name>", "local", ["branch", "create", "feature/new", "--from", "eval/base", "--dry-run", "--format", "json"], True),
    probe("branch publish <name>", "public", ["branch", "publish", "feature/old-unmerged", "--dry-run", "--format", "json"], True),
    probe("branch rename <old> <new>", "local", ["branch", "rename", "feature/active", "feature/renamed", "--dry-run", "--format", "json"], True),
    probe("branch delete <name>", "local", ["branch", "delete", "feature/merged", "--local", "--dry-run", "--format", "json"], True),
    probe("tag publish <name>", "local", ["tag", "publish", "eval-demo", "--commit", "HEAD", "--dry-run", "--format", "json"], True),
    probe("prs", "public", ["prs", "--repo", "Raithlin/gha", "--state", "closed", "--head", "Raithlin:codex/guarded-release-publication", "--limit", "2", "--format", "json"]),
    probe("review <number>", "public", ["review", "9", "--repo", "Raithlin/gha", "--format", "json"]),
    probe("pr prepare", "public", ["pr", "prepare", "--repo", "Raithlin/gha", "--head", "feature/active", "--base", "master", "--format", "json"]),
    probe("pr create", "public", ["pr", "create", "--repo", "Raithlin/gha", "--head", "feature/active", "--base", "master", "--dry-run", "--format", "json"], True),
    probe("releases", "public", ["releases", "--repo", "Raithlin/gha", "--limit", "3", "--format", "json"]),
    probe("release create-notes --since <timestamp>", "public", ["release", "create-notes", "--repo", "Raithlin/gha", "--since", "2026-09-25T00:00:00Z", "--limit", "3", "--format", "json"]),
    probe("release publish <version>", "clean", ["release", "publish", "99.0.0-eval", "--repo", "Raithlin/gha", "--dry-run", "--format", "json"], True),
)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, default=Path(__file__).resolve().parents[2])
    parser.add_argument("--revision", required=True)
    parser.add_argument("--gha", type=Path, required=True)
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()
    source = args.repo.resolve()
    binary = args.gha.resolve()
    inventory = json.loads(subprocess.check_output([str(binary), "capabilities", "--format", "json"], text=True))
    available = {item["command"] for item in inventory["commands"] if item["status"] == "available"}
    mapped = {item["capability"] for item in PROBES}
    if mapped != available:
        parser.error(f"probe inventory mismatch: missing={sorted(available-mapped)}, extra={sorted(mapped-available)}")
    with tempfile.TemporaryDirectory(prefix="gha-capability-probe-") as directory:
        root = Path(directory)
        for name in ("home", "codex-home", "gh-config", "xdg-config"):
            (root / name).mkdir()
        env = {**os.environ, "HOME": str(root / "home"), "CODEX_HOME": str(root / "codex-home"),
               "GH_CONFIG_DIR": str(root / "gh-config"), "XDG_CONFIG_HOME": str(root / "xdg-config"),
               "GHA_GITHUB_TOKEN": "", "GH_TOKEN": "", "GITHUB_TOKEN": ""}
        repositories = {}
        for label, fixture in (("local", "branch_cleanup"),
                               ("public", "branch_cleanup_github_origin"),
                               ("clean", "release_checkout")):
            checkout = root / label
            subprocess.run(["git", "clone", "--quiet", "--no-hardlinks", str(source),
                            str(checkout)], check=True)
            subprocess.run(["git", "-C", str(checkout), "checkout", "--quiet", "--detach",
                            args.revision], check=True)
            prepare_fixture(checkout, fixture)
            repositories[label] = checkout
        preseed_agent_setup(root, binary, env)
        records = []
        for item in PROBES:
            checkout = repositories[item["fixture"]]
            before_refs = refs(checkout)
            before_tree = working_tree_digest(checkout)
            before_home = working_tree_digest(root / "home")
            before_config = working_tree_digest(root / "xdg-config")
            try:
                result = subprocess.run([str(binary), *item["args"]], cwd=checkout,
                                        env=env, capture_output=True, text=True, timeout=60)
                exit_code = result.returncode
                try:
                    output = json.loads(result.stdout)
                except ValueError:
                    output = {}
                error_tail = result.stderr[-300:]
            except subprocess.TimeoutExpired:
                exit_code, output, error_tail = None, {}, "timeout"
            record = {"capability": item["capability"], "fixture": item["fixture"],
                      "write_intent": item["write_intent"], "dry_run": "--dry-run" in item["args"],
                      "exit_code": exit_code, "json_output": bool(output),
                      "error_code": output.get("code"),
                      "refs_unchanged": refs(checkout) == before_refs,
                      "tree_unchanged": working_tree_digest(checkout) == before_tree,
                      "home_unchanged": working_tree_digest(root / "home") == before_home,
                      "config_unchanged": working_tree_digest(root / "xdg-config") == before_config}
            if item["capability"] == "update" and "does not declare gha-required-capabilities" in error_tail:
                record["failure_reason"] = "published_skill_missing_capability_manifest"
            records.append(record)
            print(item["capability"], "exit", exit_code,
                  "safe", all(record[key] for key in ("refs_unchanged", "tree_unchanged",
                                                        "home_unchanged", "config_unchanged")),
                  "error", record["error_code"] or error_tail[:100], flush=True)
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(json.dumps({"schema_version": "v1",
                                     "captured_at_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
                                     "revision": args.revision,
                                     "gha_sha256": hashlib.sha256(binary.read_bytes()).hexdigest(),
                                     "available_count": len(available),
                                     "runs": records}, indent=2) + "\n")


if __name__ == "__main__":
    main()
