import importlib.util
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


SPEC = importlib.util.spec_from_file_location("report", Path(__file__).with_name("report.py"))
report = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(report)


class CoverageReportingTests(unittest.TestCase):
    def test_reports_real_go_total_and_preserves_existing_summary(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / "example.go"
            source.write_text("package example\nfunc Example() int {\n x := 1\n return x\n}\n")
            profile = root / "coverage.out"
            profile.write_text(f"mode: atomic\n{source}:2.20,3.8 19 1\n{source}:4.2,4.10 1 0\n")
            summary = root / "summary.md"
            summary.write_text("Earlier summary\n")
            report.publish(profile, summary, "abc123")
            text = summary.read_text()
            self.assertTrue(text.startswith("Earlier summary\n"))
            self.assertIn("95.0%", text)
            self.assertIn("95%", text)
            self.assertIn("abc123", text)

    def test_missing_or_invalid_totals_never_become_zero(self):
        for value in ("", "example.go:2: Example 95.0%", "total: (statements) NaN%",
                      "total: (statements) 101.0%", "total: (statements) -1.0%",
                      "total: (statements) 95.0%\ntotal: (statements) 96.0%"):
            with self.subTest(value=value), self.assertRaises(ValueError):
                report.parse_total(value)

    def test_total_not_individual_function_or_package_average(self):
        self.assertEqual(report.parse_total("example.go:2: Example 0.0%\ntotal: (statements) 95.2%\n"), "95.2")

    def test_missing_profile_fails_without_publishing(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with self.assertRaises((OSError, ValueError)):
                report.publish(root / "missing.out", root / "summary", "abc")
            self.assertFalse((root / "summary").exists())

    def test_cli_failure_is_actionable_and_nonzero(self):
        with tempfile.TemporaryDirectory() as directory:
            result = subprocess.run(
                [sys.executable, "-B", str(Path(__file__).with_name("report.py")),
                 "--profile", str(Path(directory) / "missing.out"), "--revision", "abc"],
                capture_output=True, text=True,
            )
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(result.stdout, "")
            self.assertIn("cannot measure coverage", result.stderr)
            self.assertNotIn("Traceback", result.stderr)


if __name__ == "__main__":
    unittest.main()
