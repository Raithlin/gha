"""Exercise generated casks in disposable macOS CI, using the built archive bytes."""

import functools
import http.server
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import threading


def local_cask(cask, base_url):
    result, count = re.subn(
        r"https://github\.com/Raithlin/gha/releases/download/[^/]+/",
        base_url,
        cask,
    )
    if not count:
        raise ValueError("generated cask has no expected GHA release URL")
    return result


def run(*args):
    subprocess.run(args, check=True)


def check_digest_rejection(result):
    message = result.stdout + result.stderr
    assert result.returncode != 0, "Homebrew accepted an incorrect digest"
    assert "checksum" in message.lower() and "0" * 64 in message, message


def check_binary(version):
    identity = json.loads(subprocess.check_output(["gha", "version", "--format", "json"]))
    assert identity["version"] == version, identity
    capabilities = json.loads(subprocess.check_output(["gha", "capabilities", "--format", "json"]))
    assert capabilities["commands"]
    run("gha", "version")


def main(packages):
    if sys.platform != "darwin" or os.environ.get("GHA_DISPOSABLE_PACKAGE_TEST") != "1":
        raise SystemExit("Run only in disposable macOS package CI")
    packages = Path(packages).resolve()
    handler = functools.partial(http.server.SimpleHTTPRequestHandler, directory=str(packages))
    server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), handler)
    threading.Thread(target=server.serve_forever, daemon=True).start()
    base = f"http://127.0.0.1:{server.server_port}/"
    tap_name = "gha-ci/packages"
    guidance = [Path.home() / ".codex/AGENTS.md", Path.home() / ".agents/skills/gha/SKILL.md"]
    for path in guidance:
        assert not path.exists(), f"Refuse to replace existing CI guidance: {path}"
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text("Personal customized guidance\n")
    before = [path.read_bytes() for path in guidance]
    try:
        with tempfile.TemporaryDirectory() as directory:
            run("git", "init", directory)
            run("git", "-C", directory, "-c", "user.name=Package CI", "-c", "user.email=ci@example.invalid", "commit", "--allow-empty", "-m", "Initialize local test tap")
            run("brew", "tap", tap_name, directory)
            tap = Path(subprocess.check_output(["brew", "--repository", tap_name], text=True).strip())
            cask_path = tap / "Casks/gha.rb"
            cask_path.parent.mkdir(exist_ok=True)
            for index, version in enumerate(("0.0.1-test", "0.0.2-test")):
                cask = local_cask((packages / version / "homebrew/Casks/gha.rb").read_text(), base + version + "/")
                if index == 0:
                    cask_path.write_text(re.sub(r'sha256 "[a-f0-9]{64}"', 'sha256 "' + "0" * 64 + '"', cask))
                    rejected = subprocess.run(["brew", "fetch", "--cask", "--force", tap_name + "/gha"], capture_output=True, text=True)
                    check_digest_rejection(rejected)
                cask_path.write_text(cask)
                operation = "install" if index == 0 else "upgrade"
                run("brew", operation, "--cask", tap_name + "/gha")
                check_binary(version)
                assert [path.read_bytes() for path in guidance] == before
                assert not (Path.home() / ".config/gha").exists()
            run("brew", "uninstall", "--cask", tap_name + "/gha")
            assert [path.read_bytes() for path in guidance] == before
            run("brew", "untap", tap_name)
    finally:
        server.shutdown()


if __name__ == "__main__":
    main(sys.argv[1])
