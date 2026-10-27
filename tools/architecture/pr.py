#!/usr/bin/env python3
"""Repository-only Groma refresh, freshness check, and guarded PR entrypoint."""
import argparse
import contextlib
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

GROMA_VERSION = "0.7.7"
PR_REPOSITORY = "Raithlin/gha"
ROOT = Path(__file__).resolve().parents[2]


class WorkflowError(Exception):
    """A failed prerequisite or architecture check requiring author action."""


def run(args, root, *, diagnostics=False):
    try:
        result = subprocess.run(
            [str(arg) for arg in args], cwd=root, text=True, check=True,
            stdout=sys.stderr if diagnostics else subprocess.PIPE,
            stderr=sys.stderr if diagnostics else subprocess.PIPE,
        )
    except FileNotFoundError as exc:
        raise WorkflowError(f"Required executable is unavailable: {args[0]}") from exc
    except subprocess.CalledProcessError as exc:
        detail = (exc.stderr or "").strip()
        raise WorkflowError(f"{args[0]} failed (exit {exc.returncode}). {detail}") from exc
    return result.stdout or ""


def git(root, *args):
    return run(["git", *args], root).strip()


def scan(root):
    if not (root / ".groma").is_dir():
        raise WorkflowError("This repository needs its committed .groma architecture records.")
    if shutil.which("groma") is None:
        raise WorkflowError(f"Groma {GROMA_VERSION} is required; install groma.md@{GROMA_VERSION}.")
    version = run(["groma", "--version"], root).strip().removeprefix("v")
    if version != GROMA_VERSION:
        raise WorkflowError(f"Groma {GROMA_VERSION} is required; found {version}.")
    run(["groma", "scan"], root, diagnostics=True)


def architecture_changes(root):
    return git(root, "status", "--porcelain=v1", "--untracked-files=all", "--", ".groma")


def refresh(root):
    scan(root)
    changes = architecture_changes(root)
    if changes:
        print("Architecture refreshed. Review ownership, descriptions, and relationships, "
              "then commit the changes together:\n" + changes, file=sys.stderr)
    else:
        print("Architecture scan produced no record changes. Review the affected map "
              "when responsibilities or relationships changed.", file=sys.stderr)


@contextlib.contextmanager
def committed_snapshot(root):
    revision = git(root, "rev-parse", "HEAD")
    with tempfile.TemporaryDirectory(prefix="gha-architecture-") as temporary:
        snapshot = Path(temporary) / "checkout"
        # Clone only local Git data; do not fetch, copy worktree edits, or run setup hooks.
        run(["git", "-c", "init.templateDir=", "clone", "--quiet", "--no-checkout",
             "--no-local", root, snapshot], root)
        git(snapshot, "checkout", "--quiet", "--detach", revision)
        yield snapshot


def check(root):
    with committed_snapshot(root) as snapshot:
        scan(snapshot)
        changes = architecture_changes(snapshot)
        if changes:
            print(changes, file=sys.stderr)
            raise WorkflowError("Committed architecture evidence is stale. Run make "
                                "architecture-refresh, review the map, commit the records, "
                                "then retry. This check did not change your checkout.")
    print("Committed architecture evidence is current; semantic review remains required.",
          file=sys.stderr)


def create(root, *, title=None, body_file=None, base=None, dry_run=False):
    if git(root, "status", "--porcelain=v1", "--untracked-files=all"):
        raise WorkflowError("Review and commit local changes before PR creation. "
                            "Use make architecture-refresh before committing source changes.")
    # Read the configured URL, before Git rewrites, as GHA's resolver does.
    origin = git(root, "config", "--get", "remote.origin.url").lower().rstrip("/").removesuffix(".git")
    repository = PR_REPOSITORY.lower()
    if origin not in {f"https://github.com/{repository}",
                      f"git@github.com:{repository}",
                      f"ssh://git@github.com/{repository}"}:
        raise WorkflowError("This helper supports same-repository branches in Raithlin/gha; "
                            "for fork contributions, refresh and review architecture, then "
                            "open the upstream PR through GitHub or gh. See docs/development.md.")
    branch = git(root, "symbolic-ref", "--quiet", "--short", "HEAD")
    if dry_run:
        check(root)
    else:
        refresh(root)
        if architecture_changes(root):
            raise WorkflowError("Review and commit the refreshed architecture, push the "
                                "branch, then retry make pr. No PR was created.")
    try:
        published = git(root, "ls-remote", "--exit-code", "origin", f"refs/heads/{branch}")
    except WorkflowError as exc:
        raise WorkflowError("Cannot verify the published origin head. Publish this branch "
                            "and resolve Git authentication before retrying make pr. "
                            f"{exc}") from exc
    if published.split()[0] != git(root, "rev-parse", "HEAD"):
        raise WorkflowError("Commit and push this exact branch before creating its PR; "
                            "the origin head differs from the checked architecture.")
    args = [root / "bin/gha", "pr", "create", "--repo", PR_REPOSITORY,
            "--path", root, "--head", branch,
            "--format", "json"]
    if title:
        args += ["--title", title]
    if body_file:
        args += ["--body", Path(body_file).read_text()]
    if base:
        args += ["--base", base]
    if dry_run:
        args += ["--dry-run"]
    # Inherit stdout only for GHA's structured result; scan diagnostics go to stderr.
    try:
        result = subprocess.run([str(arg) for arg in args], cwd=root, check=False)
    except FileNotFoundError as exc:
        raise WorkflowError("Build the repository-local GHA binary with make build.") from exc
    if result.returncode:
        raise WorkflowError(f"GHA PR creation failed (exit {result.returncode}).")


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=["refresh", "check", "create", "version"])
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--title", default=os.environ.get("PR_TITLE"))
    parser.add_argument("--body-file", default=os.environ.get("PR_BODY_FILE"))
    parser.add_argument("--base", default=os.environ.get("PR_BASE"))
    args = parser.parse_args(argv)
    try:
        if args.command == "version":
            print(GROMA_VERSION)
        elif args.command == "refresh":
            if args.dry_run:
                check(ROOT)
            else:
                refresh(ROOT)
        elif args.command == "check":
            check(ROOT)
        else:
            create(ROOT, title=args.title, body_file=args.body_file, base=args.base,
                   dry_run=args.dry_run)
    except (WorkflowError, OSError) as exc:
        print(f"Architecture workflow blocked: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
