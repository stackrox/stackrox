#!/usr/bin/env python3
"""Set file mtimes from git blob SHAs to preserve Go build cache across CI runs.

Git checkout always sets file mtime to the current time, which invalidates
Go's stat cache (which maps (path, size, mtime) -> content_hash). This forces
Go to re-hash all source files on every CI run even when nothing changed.

By deriving mtime from the git blob SHA (which IS the content hash), files get
an identical mtime on any runner for the same content. The GOCACHE stat cache
then hits on all unchanged files, avoiding the re-hash step entirely.

Performance: reads the git index in one subprocess call, sets mtimes in-process.
For ~12k files this runs in ~1-2s vs Go's ~40s re-hashing on CI runners.

Usage:
    python3 scripts/ci/set-file-mtimes-from-git.py
    python3 scripts/ci/set-file-mtimes-from-git.py --verify
"""

import argparse
import os
import subprocess
import sys
import time


def blob_sha_to_mtime(blob_sha: str) -> int:
    """Derive a deterministic mtime from a git blob SHA.

    Uses the first 8 hex characters as an integer, mapped into the
    range [2001-01-01, 2021-09-09] (epoch 978307200 to 1631188735).
    This avoids year-1970 timestamps (which confuse some tools) and
    ensures all mtimes are well in the past so Go's test cache never
    rejects them as "too new".
    """
    return int(blob_sha[:8], 16) % 652_881_536 + 978_307_200


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--verify",
        action="store_true",
        help="Check current mtimes match expected values (do not modify)",
    )
    parser.add_argument(
        "--verbose", "-v",
        action="store_true",
        help="Print each file being processed",
    )
    args = parser.parse_args()

    start = time.monotonic()

    # Get repo root
    root = subprocess.run(
        ["git", "rev-parse", "--show-toplevel"],
        capture_output=True, text=True, check=True,
    ).stdout.strip()

    # Read the git index in one shot: blob SHA + path, NUL-separated
    # --format uses the newer porcelain format (git 2.11+)
    # -z uses NUL as record separator to handle filenames with spaces/newlines
    result = subprocess.run(
        ["git", "ls-files", "-z", "--format=%(objectname) %(path)"],
        capture_output=True, text=True, check=True,
        cwd=root,
    )

    count = 0
    mismatches = 0
    errors = 0

    for record in result.stdout.split("\0"):
        record = record.strip()
        if not record:
            continue
        try:
            blob_sha, path = record.split(" ", 1)
        except ValueError:
            continue

        expected_mtime = blob_sha_to_mtime(blob_sha)
        full_path = os.path.join(root, path)

        if args.verify:
            try:
                actual_mtime = int(os.stat(full_path).st_mtime)
                if actual_mtime != expected_mtime:
                    mismatches += 1
                    if args.verbose:
                        print(f"MISMATCH: {path} expected={expected_mtime} actual={actual_mtime}")
            except OSError:
                errors += 1
        else:
            try:
                os.utime(full_path, (expected_mtime, expected_mtime))
                count += 1
                if args.verbose:
                    print(f"SET: {path} mtime={expected_mtime}")
            except OSError as e:
                errors += 1
                if errors <= 5:
                    print(f"Warning: could not set mtime for {path}: {e}", file=sys.stderr)

    elapsed = time.monotonic() - start

    if args.verify:
        status = "OK" if mismatches == 0 else f"{mismatches} MISMATCHES"
        print(f"Verified {count + mismatches} files: {status} ({errors} errors) in {elapsed:.2f}s")
        return 1 if mismatches > 0 else 0
    else:
        print(f"Set mtimes for {count} files ({errors} errors) in {elapsed:.2f}s")
        return 0


if __name__ == "__main__":
    sys.exit(main())
