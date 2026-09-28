import tempfile
import unittest
from pathlib import Path

import make_held_out_source


class HeldOutSourceTests(unittest.TestCase):
    def test_source_revision_is_reproducible_and_existing_path_is_preserved(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "source"
            self.assertEqual(make_held_out_source.create_source(path),
                             make_held_out_source.REVISION)
            with self.assertRaises(FileExistsError):
                make_held_out_source.create_source(path)


if __name__ == "__main__":
    unittest.main()
