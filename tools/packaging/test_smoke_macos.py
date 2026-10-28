import unittest
from types import SimpleNamespace

import smoke_macos


class CaskFixtureTests(unittest.TestCase):
    def test_local_download_keeps_version_and_integrity_verification(self):
        cask = '''version "0.0.1-test"
sha256 "abc123"
url "https://github.com/Raithlin/gha/releases/download/v0.3.0-alpha/gha_#{version}_darwin_arm64.tar.gz"
'''
        local = smoke_macos.local_cask(cask, "http://127.0.0.1:8000/0.0.1-test/")
        self.assertIn('sha256 "abc123"', local)
        self.assertIn('version "0.0.1-test"', local)
        self.assertIn('url "http://127.0.0.1:8000/0.0.1-test/gha_#{version}_darwin_arm64.tar.gz"', local)

    def test_unknown_download_source_fails_instead_of_testing_wrong_artifact(self):
        with self.assertRaisesRegex(ValueError, "release URL"):
            smoke_macos.local_cask('url "https://example.com/file"', "http://127.0.0.1/")

    def test_current_homebrew_checksum_rejection_is_recognized(self):
        result = SimpleNamespace(returncode=1, stdout="SHA-256 checksum of downloaded file: abc", stderr="Cask reports different checksum: " + "0" * 64)
        smoke_macos.check_digest_rejection(result)

    def test_network_failure_is_not_evidence_of_checksum_rejection(self):
        result = SimpleNamespace(returncode=1, stdout="", stderr="Connection refused")
        with self.assertRaises(AssertionError):
            smoke_macos.check_digest_rejection(result)


if __name__ == "__main__":
    unittest.main()
