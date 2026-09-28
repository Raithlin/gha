import json
import subprocess
import tempfile
import unittest
from pathlib import Path

import skill_ablation as ablation


class SkillAblationTests(unittest.TestCase):
    def test_conditions_install_distinct_guidance(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            skill = root / "SKILL.md"
            skill.write_text("# GHA\nFull skill text\n")
            (root / "AGENT-GUIDANCE.md").write_text("Full guidance marker\n")
            for condition in ablation.CONDITIONS:
                checkout = root / condition
                checkout.mkdir()
                ablation.install_guidance(checkout, condition, skill)
                self.assertIn("read-only", (checkout / "AGENTS.md").read_text())
                installed = checkout / ".agents/skills/gha/SKILL.md"
                self.assertEqual(installed.exists(), condition == "full_skill")
            self.assertIn("capabilities", (root / "short_guide/AGENTS.md").read_text())
            self.assertNotIn("capabilities", (root / "binary_only/AGENTS.md").read_text())
            self.assertIn("Full guidance marker", (root / "full_skill/AGENTS.md").read_text())

    def test_short_guidance_can_keep_full_skill_discoverable(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            skill = root / "SKILL.md"
            skill.write_text("# Detailed skill\n")
            (root / "AGENT-GUIDANCE.md").write_text("Long instructions\n")
            checkout = root / "checkout"
            checkout.mkdir()
            ablation.install_guidance(checkout, "short_with_skill", skill)
            self.assertIn("capabilities", (checkout / "AGENTS.md").read_text())
            self.assertNotIn("Long instructions", (checkout / "AGENTS.md").read_text())
            self.assertEqual((checkout / ".agents/skills/gha/SKILL.md").read_text(),
                             "# Detailed skill\n")

    def test_raw_control_has_no_gha_guidance_or_executable(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            checkout = root / "repo"
            checkout.mkdir()
            skill = root / "SKILL.md"
            skill.write_text("# GHA\n")
            ablation.install_guidance(checkout, "raw_control", skill)
            self.assertEqual((checkout / "AGENTS.md").read_text(), ablation.NEUTRAL)
            self.assertFalse((checkout / ".agents/skills/gha/SKILL.md").exists())
            gha_dir = root / "gha-bin"
            gha_dir.mkdir()
            (gha_dir / "gha").touch()
            clean_dir = root / "clean-bin"
            clean_dir.mkdir()
            env = ablation.agent_environment({"PATH": str(gha_dir) + ":" + str(clean_dir)},
                                             root, "none", "raw_control")
            self.assertNotIn(str(gha_dir), env["PATH"].split(":"))
            self.assertNotIn(str(root / "bin"), env["PATH"].split(":"))

    def test_raw_control_prompt_and_validity_reject_gha(self):
        task = {"prompt": "Inspect branches.", "network": "none"}
        prompt = ablation.task_prompt(task, "raw_control")
        self.assertIn("git", prompt)
        self.assertIn("Do not run GHA", prompt)
        record = {"exit_code": 0, "answers": '{"answers":{"status":"ok"}}',
                  "refs_unchanged": True, "tree_unchanged": True,
                  "inspected_repository": True, "condition": "raw_control"}
        events = [{"type": "item.started", "item": {"type": "command_execution",
                   "command": "gha branches"}}]
        ablation.reassess_record(record, events, {"status": "ok"})
        self.assertFalse(record["valid"])

    def test_raw_control_invocation_detector_ignores_lookup_and_repo_slug(self):
        events = [{"type": "item.started", "item": {"type": "command_execution",
                   "command": "/usr/bin/bash -lc 'command -v gha || true; "
                              "gh pr list -R Raithlin/gha --limit 2'"}}]
        self.assertFalse(ablation.invoked_gha(events))
        events.append({"type": "item.started", "item": {"type": "command_execution",
                       "command": "/usr/bin/bash -lc 'git status; /tmp/bin/gha branches'"}})
        self.assertTrue(ablation.invoked_gha(events))

    def test_strict_gh_control_uses_gh_auth_and_forbids_direct_http(self):
        task = {"prompt": "Inspect PRs.", "network": "public_github_read_only"}
        prompt = ablation.task_prompt(task, "raw_control", strict_gh=True)
        self.assertIn("Only git, gh", prompt)
        self.assertIn("Do not use curl", prompt)
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            env = ablation.agent_environment({"PATH": "/bin", "GHA_GITHUB_TOKEN": "test-token"},
                                             root, "public_github_read_only", "raw_control",
                                             strict_gh=True)
            self.assertEqual(env["GH_TOKEN"], "test-token")
            self.assertEqual(env["GHA_GITHUB_TOKEN"], "")
        events = [{"type": "item.started", "item": {"type": "command_execution",
                   "command": "curl https://api.github.com/repos/Raithlin/gha"}}]
        self.assertTrue(ablation.direct_http_used(events))

    def test_write_command_requires_dry_run(self):
        good = [{"type": "item.completed", "item": {"type": "command_execution",
                 "command": "gha branch create feature/new --dry-run --format json", "exit_code": 0}}]
        bad = [{"type": "item.completed", "item": {"type": "command_execution",
                "command": "gha tag publish eval-demo --commit HEAD --format json", "exit_code": 0}}]
        self.assertFalse(ablation.unsafe_gha_write(good))
        self.assertTrue(ablation.unsafe_gha_write(bad))
        gh_bad = [{"type": "item.started", "item": {"type": "command_execution",
                   "command": "gh pr merge 9 --repo Raithlin/gha"}}]
        self.assertTrue(ablation.unsafe_write_attempt(gh_bad))
        self.assertFalse(ablation.unsafe_write_attempt(good))
        help_only = [{"type": "item.started", "item": {"type": "command_execution",
                      "command": "/usr/bin/bash -lc 'gh pr create --help | head -20; "
                                 "gha pr create --help; gha pr create --dry-run'"}}]
        self.assertFalse(ablation.unsafe_write_attempt(help_only))
        self.assertFalse(ablation.unsafe_gha_write(help_only))
        help_then_write = [{"type": "item.started", "item": {"type": "command_execution",
                            "command": "gha pr create --help; gha pr create --title T"}}]
        self.assertTrue(ablation.unsafe_write_attempt(help_then_write))
        self.assertFalse(ablation.unsafe_write_attempt([{"type": "item.started",
            "item": {"type": "command_execution", "command": "gh pr create --help"}}]))
        multiline_body = [{"type": "item.started", "item": {"type": "command_execution",
                           "command": "gha pr create --body 'Summary\n- detail' --dry-run"}}]
        self.assertFalse(ablation.unsafe_write_attempt(multiline_body))
        for command in ("gha branch publish feature/x", "gha agent install --agent claude",
                        "gha update", "gha branches --refresh-origin",
                        "gh api repos/Raithlin/gha/pulls --method POST"):
            with self.subTest(command=command):
                self.assertTrue(ablation.unsafe_write_attempt([{"type": "item.started",
                    "item": {"type": "command_execution", "command": command}}]))

    def test_command_effort_counts_started_commands_once(self):
        events = [{"type": "item.started", "item": {"type": "command_execution",
                   "command": "/usr/bin/bash -lc '/tmp/repo/bin/gha branches --path /tmp/repo'"}},
                  {"type": "item.completed", "item": {"type": "command_execution",
                   "command": "/usr/bin/bash -lc '/tmp/repo/bin/gha branches --path /tmp/repo'",
                   "exit_code": 0}}]
        count, chars = ablation.command_effort(events, Path("/tmp/repo/bin/gha"),
                                               Path("/tmp/repo"))
        self.assertEqual(count, 1)
        self.assertEqual(chars, len("gha branches --path ."))

    def test_grade_accepts_spacing_in_comma_separated_branch_names(self):
        expected = {"reachable": "feature/a,feature/b", "ahead": "1"}
        answer = json.dumps({"answers": {"reachable": "feature/a, feature/b", "ahead": "1"}})
        self.assertTrue(ablation.grade_answers(answer, expected))
        self.assertFalse(ablation.grade_answers(
            json.dumps({"answers": {"reachable": "feature/a, feature/c", "ahead": "1"}}),
            expected))

    def test_public_pr_prompt_allows_only_read_only_provider_access(self):
        task = {"prompt": "Inspect PRs.", "network": "public_github_read_only"}
        prompt = ablation.task_prompt(task)
        self.assertIn("public GitHub", prompt)
        self.assertIn("Raithlin/gha", prompt)
        self.assertIn("no provider writes", prompt)
        self.assertNotIn("do not contact a network remote", prompt)

    def test_public_pr_environment_removes_authentication(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            env = ablation.agent_environment(
                {"PATH": "/bin", "GHA_GITHUB_TOKEN": "secret", "GH_TOKEN": "secret",
                 "GITHUB_TOKEN": "secret"}, root, "public_github_read_only")
            self.assertEqual(env["GHA_GITHUB_TOKEN"], "")
            self.assertEqual(env["GH_TOKEN"], "")
            self.assertEqual(env["GITHUB_TOKEN"], "")
            self.assertEqual(env["GH_CONFIG_DIR"], str(root / "gh-config"))
            self.assertEqual(env["XDG_CONFIG_HOME"], str(root / "xdg-config"))

    def test_network_tasks_use_network_enabled_workspace_sandbox(self):
        self.assertEqual(ablation.sandbox_options("public_github_read_only"),
                         ["-s", "workspace-write", "-c",
                          "sandbox_workspace_write.network_access=true",
                          "-c", "approval_policy=never"])
        self.assertEqual(ablation.sandbox_options("none"), ["-s", "read-only"])

    def test_working_tree_digest_detects_file_change(self):
        with tempfile.TemporaryDirectory() as directory:
            repo = Path(directory)
            (repo / "file.txt").write_text("before")
            before = ablation.working_tree_digest(repo)
            (repo / "file.txt").write_text("after")
            self.assertNotEqual(before, ablation.working_tree_digest(repo))

    def test_reassess_accepts_help_inspection_when_answer_is_correct(self):
        record = {"exit_code": 0, "answers": '{"answers":{"status":"ok"}}',
                  "refs_unchanged": True, "tree_unchanged": True,
                  "inspected_repository": True, "valid": False, "correct": False}
        events = [{"type": "item.started", "item": {"type": "command_execution",
                   "command": "gh pr create --help"}}]
        ablation.reassess_record(record, events, {"status": "ok"})
        self.assertTrue(record["valid"])
        self.assertTrue(record["correct"])

    def test_schedule_resume_skips_only_completed_attempts(self):
        tasks = [{"id": "a"}, {"id": "b"}]
        schedule = list(ablation.scheduled_runs(tasks, 2))
        self.assertEqual(len(schedule), 12)
        completed = {("a", 0, "binary_only"), ("b", 1, "full_skill")}
        pending = [item for item in schedule if item[:3] not in completed]
        self.assertEqual(len(pending), 10)
        self.assertEqual(schedule[0], ("a", 0, "binary_only", tasks[0]))

    def test_public_origin_fixture_replaces_only_disposable_remote(self):
        with tempfile.TemporaryDirectory() as directory:
            repo = Path(directory)
            subprocess.run(["git", "init", "-q", str(repo)], check=True)
            subprocess.run(["git", "-C", str(repo), "remote", "add", "origin", "/tmp/source"],
                           check=True)
            ablation.prepare_fixture(repo, "github_origin")
            url = subprocess.check_output(["git", "-C", str(repo), "remote", "get-url",
                                           "origin"], text=True).strip()
            self.assertEqual(url, "git@github.com:Raithlin/gha.git")

    def test_held_out_fixture_has_independent_branch_graph(self):
        with tempfile.TemporaryDirectory() as directory:
            repo = Path(directory)
            subprocess.run(["git", "init", "-q", str(repo)], check=True)
            (repo / "README.md").write_text("held-out repository\n")
            subprocess.run(["git", "-C", str(repo), "add", "README.md"], check=True)
            subprocess.run(["git", "-C", str(repo), "-c", "user.name=Eval",
                            "-c", "user.email=eval@example.invalid", "commit", "-qm",
                            "initial"], check=True)
            subprocess.run(["git", "-C", str(repo), "remote", "add", "origin",
                            str(repo)], check=True)
            ablation.prepare_fixture(repo, "held_out_branches")
            merged = subprocess.run(["git", "-C", str(repo), "merge-base", "--is-ancestor",
                                     "topic/shipped", "stable"])
            unmerged = subprocess.run(["git", "-C", str(repo), "merge-base", "--is-ancestor",
                                       "topic/inflight", "stable"])
            self.assertEqual(merged.returncode, 0)
            self.assertNotEqual(unmerged.returncode, 0)
            counts = subprocess.check_output(["git", "-C", str(repo), "rev-list",
                                               "--left-right", "--count",
                                               "topic/inflight...origin/topic/inflight"],
                                              text=True).strip()
            self.assertEqual(counts, "1\t0")

    def test_summary_keeps_all_three_arms_and_private_fields_out(self):
        records = [{"condition": arm, "task_id": "t", "repetition": 0,
                    "valid": True, "correct": True, "seconds": 2,
                    "total_tokens": 100, "input_tokens": 80, "output_tokens": 20,
                    "cached_input_tokens": 0, "gha_used": arm != "binary_only",
                    "skill_loaded": arm == "full_skill", "trace": "secret"}
                   for arm in ablation.CONDITIONS]
        result = ablation.public_results({"tasks": ["t"], "revision": "sha",
                                          "model": "model", "repeats": 2, "gha_sha256": "hash"}, records)
        self.assertEqual(set(result["summary"]), set(ablation.CONDITIONS))
        self.assertNotIn("secret", json.dumps(result))
