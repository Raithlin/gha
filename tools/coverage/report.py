"""Report Go's measured total without replacing the CI coverage gate."""

import argparse
from decimal import Decimal
import json
from pathlib import Path
import re
import subprocess
import sys


def parse_total(output):
    totals = re.findall(r"^total:\s+\(statements\)\s+(\d+(?:\.\d+)?)%\s*$", output, re.MULTILINE)
    if len(totals) != 1 or not 0 <= Decimal(totals[0]) <= 100:
        raise ValueError("expected one valid total statement coverage from go tool cover")
    return totals[0]


def badge_data(total, revision):
    return {
        "schemaVersion": 1,
        "label": "coverage",
        "message": f"{total}%",
        "color": "brightgreen" if Decimal(total) >= 95 else "red",
        "commit": revision,
    }


def publish(profile, summary, badge, revision):
    result = subprocess.run(
        ["go", "tool", "cover", f"-func={profile}"],
        capture_output=True, text=True, check=False,
    )
    if result.returncode:
        raise ValueError(f"cannot measure coverage: {result.stderr.strip()}")
    total = parse_total(result.stdout)
    if summary:
        with summary.open("a") as output:
            output.write(
                f"\n## Statement coverage\n\n"
                f"Measured total: **{total}%** (CI minimum: **95%**).\n\n"
                f"Revision: `{revision}`. Source: `go tool cover -func` on the test profile.\n"
            )
    if badge:
        badge.parent.mkdir(parents=True, exist_ok=True)
        badge.write_text(json.dumps(badge_data(total, revision), indent=2) + "\n")
    print(f"Total statement coverage: {total}% (revision {revision})")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--profile", type=Path, required=True)
    parser.add_argument("--summary", type=Path)
    parser.add_argument("--badge", type=Path)
    parser.add_argument("--revision", required=True)
    args = parser.parse_args()
    try:
        publish(args.profile, args.summary, args.badge, args.revision)
    except (OSError, ValueError) as error:
        print(f"coverage report: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
