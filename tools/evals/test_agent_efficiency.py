import json
import subprocess
import tempfile
import unittest
from pathlib import Path

import agent_efficiency as bench


class AgentEfficiencyTests(unittest.TestCase):
    def test_load_tasks_rejects_duplicate_ids_and_empty_expectations(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "tasks.json"
            path.write_text(json.dumps([{"id": "same", "prompt": "P", "expected": {"answer": "yes"},
                                         "gha_workflow": "analyze"},
                                        {"id": "same", "prompt": "P", "expected": {"answer": "no"},
                                         "gha_workflow": "analyze"}]))
            with self.assertRaisesRegex(ValueError, "duplicate"):
                bench.load_tasks(path)
            path.write_text(json.dumps([{"id": "one", "prompt": "P", "expected": {},
                                         "gha_workflow": "analyze"}]))
            with self.assertRaisesRegex(ValueError, "expected"):
                bench.load_tasks(path)

    def test_usage_sums_completed_turns_and_detects_control_violation(self):
        events = [
            {"type": "turn.completed", "usage": {"input_tokens": 100, "cached_input_tokens": 30,
                                                   "output_tokens": 20}},
            {"type": "item.started", "item": {"type": "command_execution", "command": "gha branches"}},
            {"type": "turn.completed", "usage": {"input_tokens": 80, "cached_input_tokens": 10,
                                                   "output_tokens": 15}},
        ]
        self.assertEqual(bench.usage_from_events(events), (180, 40, 35))
        self.assertTrue(bench.used_gha(events))
        self.assertFalse(bench.used_gha([{"type": "item.started", "item": {"type": "command_execution",
                                                             "command": "git status"}}]))
        self.assertFalse(bench.used_gha_workflow(events, "/tmp/bin/gha", "branches cleanup"))
        commands = [{"type": "item.completed", "item": {"type": "command_execution",
                     "command": "/tmp/bin/gha branches cleanup --format json", "exit_code": 0}}]
        self.assertTrue(bench.used_gha_workflow(commands, "/tmp/bin/gha", "branches cleanup"))
        self.assertFalse(bench.used_gha_workflow(commands, "/tmp/bin/gha", "branches"))
        self.assertTrue(bench.inspected_repository(commands))
        self.assertFalse(bench.inspected_repository(events))

    def test_grade_requires_all_exact_fields(self):
        expected = {"status": "unavailable", "read_only": "true"}
        self.assertTrue(bench.grade(json.dumps({"answers": expected}), expected))
        self.assertFalse(bench.grade(json.dumps({"answers": {"status": "available"}}), expected))
        self.assertFalse(bench.grade("not json", expected))

    def test_summary_counts_failures_and_excludes_missing_usage_from_token_metric(self):
        records = [
            {"condition": "control", "valid": True, "correct": True, "total_tokens": 100, "seconds": 2},
            {"condition": "control", "valid": True, "correct": False, "total_tokens": 50, "seconds": 1},
            {"condition": "gha", "valid": True, "correct": True, "total_tokens": 60, "seconds": 1},
            {"condition": "gha", "valid": False, "correct": False, "total_tokens": None, "seconds": 1},
        ]
        summary = bench.summarize(records)
        self.assertEqual(summary["control"]["correct"], 1)
        self.assertEqual(summary["control"]["tokens_per_correct"], 150)
        self.assertEqual(summary["gha"]["valid_runs"], 1)
        self.assertIsNone(summary["gha"]["tokens_per_correct"])
        self.assertEqual(summary["gha"]["usage_coverage"], 1)
        self.assertEqual(summary["gha"]["success_rate"], 0.5)

    def test_fixture_has_one_reachable_feature_branch(self):
        with tempfile.TemporaryDirectory() as directory:
            repo = Path(directory)
            subprocess.run(["git", "init", "-q", str(repo)], check=True)
            subprocess.run(["git", "-C", str(repo), "remote", "add", "origin", str(repo)],
                           check=True)
            for number in (1, 2):
                (repo / "file").write_text(str(number))
                subprocess.run(["git", "-C", str(repo), "add", "file"], check=True)
                subprocess.run(["git", "-C", str(repo), "-c", "user.name=Test",
                                "-c", "user.email=test@example.invalid", "commit", "-qm", str(number)],
                               check=True)
            bench.make_fixture(repo, "branch_cleanup")
            for branch, reachable in (("feature/merged", True), ("feature/active", False),
                                      ("feature/old-unmerged", False)):
                result = subprocess.run(["git", "-C", str(repo), "merge-base", "--is-ancestor",
                                         branch, "eval/base"], check=False)
                self.assertEqual(result.returncode == 0, reachable)

    def test_paired_summary_uses_only_jointly_correct_attempts(self):
        records = [
            {"task_id": "a", "repetition": 0, "condition": "control", "valid": True,
             "correct": True, "total_tokens": 20, "seconds": 1},
            {"task_id": "a", "repetition": 0, "condition": "gha", "valid": True,
             "correct": True, "total_tokens": 30, "seconds": 1, "gha_used": True},
            {"task_id": "b", "repetition": 0, "condition": "control", "valid": True,
             "correct": False, "total_tokens": 5, "seconds": 1},
            {"task_id": "b", "repetition": 0, "condition": "gha", "valid": True,
             "correct": True, "total_tokens": 50, "seconds": 1, "gha_used": True},
        ]
        result = bench.summarize(records)
        self.assertEqual(result["paired"]["both_correct"], 1)
        self.assertEqual(result["paired"]["median_tokens_when_both_correct"],
                         {"control": 20, "gha": 30})
        self.assertEqual(result["gha"]["gha_used_runs"], 2)

    def test_public_results_exclude_private_trace_and_answer_fields(self):
        plan = {"tasks": ["a"], "revision": "sha", "model": "model", "repeats": 2,
                "gha_sha256": "hash", "output": "/tmp/private"}
        record = {"task_id": "a", "repetition": 0, "condition": "control", "valid": True,
                  "correct": True, "total_tokens": 20, "seconds": 1, "trace": "private.jsonl",
                  "answers": "secret", "stderr_tail": "secret"}
        result = json.dumps(bench.public_results(plan, [record]))
        self.assertNotIn("secret", result)
        self.assertNotIn("private", result)
        self.assertIn('"total_tokens": 20', result)


if __name__ == "__main__":
    unittest.main()
