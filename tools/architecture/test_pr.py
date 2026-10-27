"""Behavior checks for the repository's architecture preparation workflow."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import pr


class ArchitectureWorkflowTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name) / "checkout with spaces"
        self.root.mkdir()
        self.bin = Path(self.temp.name) / "bin"
        self.bin.mkdir()
        (self.root / ".groma").mkdir()
        (self.root / ".groma" / "record.md").write_text("curated architecture\n")
        (self.root / ".gitignore").write_text("bin/\n")
        self.git("init", "-q", "-b", "feature/example")
        self.git("config", "user.name", "Fixture")
        self.git("config", "user.email", "fixture@example.invalid")
        self.git("add", ".")
        self.git("commit", "-qm", "fixture")
        origin = Path(self.temp.name) / "origin.git"
        subprocess.run(["git", "init", "--bare", "-q", str(origin)], check=True)
        self.git("config", f"url.{origin.as_uri()}.insteadOf", "https://github.com/Raithlin/gha.git")
        self.git("remote", "add", "origin", "https://github.com/Raithlin/gha.git")
        self.git("push", "-qu", "origin", "HEAD")
        self.calls = Path(self.temp.name) / "gha.json"
        scanner = self.bin / "groma"
        scanner.write_text("""#!/usr/bin/env python3
import os,sys
from pathlib import Path
if sys.argv[1:] == ['--version']:
    print(os.environ.get('FAKE_VERSION', '0.7.7'))
    raise SystemExit(0)
assert sys.argv[1:] == ['scan']
mode=os.environ.get('FAKE_SCAN', 'clean')
if mode == 'fail': raise SystemExit(9)
if mode == 'changed': Path('.groma/record.md').write_text('refreshed evidence\\n')
if mode == 'new': Path('.groma/new record.md').write_text('new evidence\\n')
""")
        scanner.chmod(0o755)
        (self.root / "bin").mkdir()
        gha = self.root / "bin" / "gha"
        gha.write_text("""#!/usr/bin/env python3
import os,sys,json
from pathlib import Path
Path(os.environ['FAKE_GHA_LOG']).write_text(json.dumps(sys.argv[1:]))
""")
        gha.chmod(0o755)
        self.env = patch.dict(os.environ, {
            "PATH": str(self.bin) + os.pathsep + os.environ["PATH"],
            "FAKE_SCAN": "clean", "FAKE_VERSION": "0.7.7",
            "FAKE_GHA_LOG": str(self.calls),
        })
        self.env.start()
        self.addCleanup(self.env.stop)

    def git(self, *args):
        return subprocess.check_output(["git", "-C", str(self.root), *args], text=True).strip()

    def test_check_clean_committed_snapshot(self):
        pr.check(self.root)
        self.assertEqual("curated architecture\n", (self.root / ".groma/record.md").read_text())

    def test_check_detects_changes_and_new_records_without_touching_checkout(self):
        for mode in ("changed", "new"):
            with self.subTest(mode=mode), patch.dict(os.environ, {"FAKE_SCAN": mode}):
                with self.assertRaisesRegex(pr.WorkflowError, "architecture-refresh"):
                    pr.check(self.root)
                self.assertEqual("curated architecture\n", (self.root / ".groma/record.md").read_text())
                self.assertFalse((self.root / ".groma/new record.md").exists())

    def test_check_handles_ci_shallow_checkout(self):
        shallow = Path(self.temp.name) / "shallow"
        subprocess.run(["git", "clone", "-q", "--depth=1", self.root.as_uri(), str(shallow)],
                       check=True)
        self.assertEqual("true", subprocess.check_output(
            ["git", "-C", str(shallow), "rev-parse", "--is-shallow-repository"],
            text=True).strip())
        pr.check(shallow)

    def test_refresh_leaves_evidence_available_for_review(self):
        with patch.dict(os.environ, {"FAKE_SCAN": "new"}):
            pr.refresh(self.root)
        self.assertTrue((self.root / ".groma/new record.md").exists())

    def test_scanner_failure_and_wrong_version_block(self):
        for settings in ({"FAKE_SCAN": "fail"}, {"FAKE_VERSION": "99.0.0"}):
            with self.subTest(settings=settings), patch.dict(os.environ, settings):
                with self.assertRaises(pr.WorkflowError):
                    pr.check(self.root)

    def test_missing_scanner_is_actionable(self):
        (self.bin / "groma").unlink()
        with patch("pr.shutil.which", return_value=None):
            with self.assertRaisesRegex(pr.WorkflowError, "Groma"):
                pr.refresh(self.root)

    def test_create_stops_before_provider_when_scan_changes_map(self):
        with patch.dict(os.environ, {"FAKE_SCAN": "changed"}):
            with self.assertRaisesRegex(pr.WorkflowError, "commit"):
                pr.create(self.root, title="Example")
        self.assertFalse(self.calls.exists())

    def test_create_rejects_uncommitted_source_before_scanning(self):
        (self.root / "source.go").write_text("package fixture\n")
        with patch.dict(os.environ, {"FAKE_SCAN": "changed"}):
            with self.assertRaisesRegex(pr.WorkflowError, "commit"):
                pr.create(self.root, title="Example")
        self.assertEqual("curated architecture\n", (self.root / ".groma/record.md").read_text())
        self.assertFalse(self.calls.exists())

    def test_create_delegates_explicit_current_branch_and_preserves_arguments(self):
        body = Path(self.temp.name) / "body.md"
        body.write_text("A body with $literal and `backticks`.\n")
        pr.create(self.root, title="Example $title", body_file=body, base="master")
        args = json.loads(self.calls.read_text())
        self.assertEqual(["pr", "create"], args[:2])
        self.assertIn("feature/example", args)
        self.assertIn(str(self.root), args)
        self.assertIn(body.read_text(), args)
        self.assertIn("Example $title", args)
        self.assertEqual("Raithlin/gha", args[args.index("--repo") + 1])

    def test_create_rejects_fork_or_unknown_origin_before_scan(self):
        for origin in ("https://github.com/contributor/gha.git",
                       "https://github.com/Raithlin/other.git",
                       "https://example.invalid/Raithlin/gha.git"):
            for dry_run in (False, True):
                with self.subTest(origin=origin, dry_run=dry_run):
                    self.git("remote", "set-url", "origin", origin)
                    with patch.dict(os.environ, {"FAKE_SCAN": "changed"}):
                        with self.assertRaisesRegex(pr.WorkflowError, "same-repository.*fork"):
                            pr.create(self.root, title="Example", dry_run=dry_run)
                    self.assertFalse(self.calls.exists())
                    self.assertEqual("curated architecture\n",
                                     (self.root / ".groma/record.md").read_text())

    def test_create_accepts_supported_github_origin_forms(self):
        local_origin = (Path(self.temp.name) / "origin.git").as_uri()
        for origin in ("git@github.com:Raithlin/gha.git",
                       "ssh://git@github.com/Raithlin/gha.git",
                       "https://github.com/raithlin/gha"):
            with self.subTest(origin=origin):
                self.git("config", "--add", f"url.{local_origin}.insteadOf", origin)
                self.git("remote", "set-url", "origin", origin)
                pr.create(self.root, title="Example", dry_run=True)
                args = json.loads(self.calls.read_text())
                self.assertEqual("Raithlin/gha", args[args.index("--repo") + 1])

    def test_create_rejects_unpublished_local_commit(self):
        (self.root / "source.go").write_text("package fixture\n")
        self.git("add", "source.go")
        self.git("commit", "-qm", "unpublished")
        with self.assertRaisesRegex(pr.WorkflowError, "push"):
            pr.create(self.root, title="Example")
        self.assertFalse(self.calls.exists())

    def test_create_explains_missing_published_branch(self):
        self.git("push", "-q", "origin", "--delete", "feature/example")
        with self.assertRaisesRegex(pr.WorkflowError, "Publish this branch"):
            pr.create(self.root, title="Example")
        self.assertFalse(self.calls.exists())

    def test_dry_run_checks_snapshot_without_refreshing_worktree(self):
        with patch.dict(os.environ, {"FAKE_SCAN": "changed"}):
            with self.assertRaises(pr.WorkflowError):
                pr.create(self.root, title="Example", dry_run=True)
        self.assertEqual("curated architecture\n", (self.root / ".groma/record.md").read_text())
        self.assertFalse(self.calls.exists())
        with patch.dict(os.environ, {"FAKE_SCAN": "clean"}):
            pr.create(self.root, title="Example", dry_run=True)
        self.assertIn("--dry-run", json.loads(self.calls.read_text()))


if __name__ == "__main__":
    unittest.main()
