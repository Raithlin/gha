import unittest
import json
from pathlib import Path

import capability_probe as probe


class CapabilityProbeTests(unittest.TestCase):
    def test_all_available_commands_have_safe_probe(self):
        self.assertEqual(len(probe.PROBES), 22)
        names = [item["capability"] for item in probe.PROBES]
        self.assertEqual(len(names), len(set(names)))
        for item in probe.PROBES:
            if item["write_intent"]:
                self.assertIn("--dry-run", item["args"], item["capability"])

    def test_agent_scenarios_cover_every_probe_capability(self):
        tasks = json.loads((Path(__file__).parent / "all_capability_tasks.json").read_text())
        covered = [name for task in tasks for name in task["capabilities"]]
        self.assertEqual(set(covered), {item["capability"] for item in probe.PROBES})
        self.assertEqual(len(covered), len(set(covered)))


if __name__ == "__main__":
    unittest.main()
