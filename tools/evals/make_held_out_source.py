#!/usr/bin/env python3
"""Create the pinned small repository used by the held-out guidance evaluation."""

import argparse
import os
import subprocess
from pathlib import Path


REVISION = "e801cb80ec060f64fe899f970e921446ef37e438"
COMMIT_DATE = "2026-09-28T16:56:10+02:00"
README = "# Tiny planning repo\n\nLocal fixture for a held-out agent workflow evaluation.\n"


def create_source(path):
    if path.exists():
        raise FileExistsError(f"refuse to replace existing source: {path}")
    path.mkdir(parents=True)
    subprocess.run(["git", "init", "-q", str(path)], check=True)
    (path / "README.md").write_text(README)
    subprocess.run(["git", "-C", str(path), "add", "README.md"], check=True)
    env = {**os.environ, "GIT_AUTHOR_DATE": COMMIT_DATE, "GIT_COMMITTER_DATE": COMMIT_DATE}
    subprocess.run(["git", "-C", str(path), "-c", "user.name=Eval", "-c",
                    "user.email=eval@example.invalid", "commit", "-qm", "initial"],
                   check=True, env=env)
    revision = subprocess.check_output(["git", "-C", str(path), "rev-parse", "HEAD"],
                                     text=True).strip()
    if revision != REVISION:
        raise RuntimeError(f"source revision differs: {revision}")
    return revision


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()
    print(create_source(args.out.resolve()))


if __name__ == "__main__":
    main()
