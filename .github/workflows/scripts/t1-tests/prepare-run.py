#!/usr/bin/env python3
"""Preserve test helpers and exact implementation blocks across clean checkouts."""

import os
from pathlib import Path
import shutil

import lib


def main():
    repo = Path.cwd()
    dest = Path("/tmp/t1-tests")
    evidence = Path(os.environ["T1_EVIDENCE"])
    if dest.exists() or evidence.exists():
        raise RuntimeError("Refusing to overwrite another test preparation")
    shutil.copytree(Path(__file__).resolve().parent, dest)
    evidence.mkdir(parents=True)
    lib.extract(repo, dest)
    shutil.copy2(dest / "original-blocks.json", evidence)
    lib.write_json(evidence / "run.json", {
        "implementation_commit": lib.BASE,
        "test_commit": os.environ["GITHUB_SHA"],
        "case": os.environ["T1_CASE"],
        "force": os.environ["T1_FORCE"],
        "dry_run": os.environ["T1_DRY_RUN"],
        "stream": os.environ["STREAM"],
        "run_id": os.environ["GITHUB_RUN_ID"],
        "run_attempt": os.environ["GITHUB_RUN_ATTEMPT"],
        "source_config_sha256": lib.digest(repo / lib.CONFIG),
    })


if __name__ == "__main__":
    main()
