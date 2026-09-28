import unittest

import compare_raw_control as comparison


class CompareRawControlTests(unittest.TestCase):
    def test_duplicate_attempt_is_not_silently_overwritten(self):
        run = {"task_id": "repository-diagnosis", "repetition": 0,
               "condition": "raw_control"}
        with self.assertRaisesRegex(ValueError, "duplicate attempt"):
            comparison.selected({"runs": [run, run.copy()]}, "raw_control",
                                ("repository-diagnosis",))

    def test_missing_usage_is_not_reported_as_zero(self):
        run = {"valid": True, "correct": True, "seconds": 1,
               "input_tokens": None, "cached_input_tokens": None,
               "output_tokens": None, "total_tokens": None,
               "command_count": 2, "command_chars": 10}
        totals = comparison.totals([run])
        self.assertIsNone(totals["total_tokens"])
        self.assertEqual(totals["correct"], 1)


if __name__ == "__main__":
    unittest.main()
