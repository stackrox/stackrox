#!/usr/bin/env python3
"""Preserve test helpers; use the corrected Part C checker for focused cases."""

import os
from pathlib import Path
import shutil

import lib

CORRECTED_CASES = {"healthy", "N3c", "N4-go-zero"}
VALIDATOR = ".github/workflows/scripts/scanner-updater-validate-import.py"


def main():
    repo = Path.cwd()
    dest = Path("/tmp/t1-tests")
    evidence = Path(os.environ["T1_EVIDENCE"])
    case = os.environ["T1_CASE"]
    stream = os.environ["STREAM"]
    if dest.exists() or evidence.exists():
        raise RuntimeError("Refusing to overwrite another test preparation")
    shutil.copytree(Path(__file__).resolve().parent, dest)
    evidence.mkdir(parents=True)
    validator_sha256 = None
    configured_sources = None
    if case in CORRECTED_CASES:
        validator = repo / VALIDATOR
        if not validator.is_file():
            raise RuntimeError("The reviewed corrected Part C validator is missing")
        shutil.copy2(validator, dest / "scanner-updater-validate-import.py")
        validator_sha256 = lib.digest(dest / "scanner-updater-validate-import.py")
        configured_sources = lib.sources(repo, stream, os.environ.copy())
    else:
        # Keep the original frozen workflow blocks only for the legacy W tests.
        # The healthy/N3c/N4 cases must exercise the corrected Part C validator.
        lib.extract(repo, dest)
        shutil.copy2(dest / "original-blocks.json", evidence)
    lib.write_json(evidence / "run.json", {
        "implementation_commit": lib.BASE,
        "test_commit": os.environ["GITHUB_SHA"],
        "case": case,
        "force": os.environ["T1_FORCE"],
        "dry_run": os.environ["T1_DRY_RUN"],
        "stream": stream,
        "run_id": os.environ["GITHUB_RUN_ID"],
        "run_attempt": os.environ["GITHUB_RUN_ATTEMPT"],
        "source_config_sha256": lib.digest(repo / lib.CONFIG),
        "part_c_checker": (
            "scanner-updater-validate-import.py" if case in CORRECTED_CASES
            else "frozen-original-block"
        ),
        "corrected_validator_sha256": validator_sha256,
        "configured_sources": configured_sources,
    })


if __name__ == "__main__":
    main()
