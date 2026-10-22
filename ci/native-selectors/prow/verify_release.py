#!/usr/bin/env python3
"""Validate the selector patches against the pinned external source configuration."""

import argparse
import hashlib
import json
from pathlib import Path
import subprocess

import yaml

from generate_patch import render


def main():
    """Check patch applicability and preserve every field except presubmit selectors."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--release-root", required=True, type=Path)
    args = parser.parse_args()
    directory = Path(__file__).resolve().parent
    policy = json.loads((directory / "selectors.json").read_text())
    fields = {"always_run", "run_if_changed", "skip_if_only_changed"}
    for expanded, patch, expected_count in [(False, "keep-triggers.patch", 1), (True, "schedule-selected.patch", 12)]:
        count = 0
        for relative, digest in policy["sources"].items():
            source = (args.release_root / relative).read_text()
            assert hashlib.sha256(source.encode()).hexdigest() == digest, relative
            before = yaml.safe_load(source)
            after = yaml.safe_load(render(source, policy, expanded))
            before_tests, after_tests = before.pop("tests"), after.pop("tests")
            assert before == after, f"{relative}: non-test configuration changed"
            assert len(before_tests) == len(after_tests)
            for old, new in zip(before_tests, after_tests):
                if old == new:
                    continue
                count += 1
                assert not any(old.get(key) for key in ("postsubmit", "cron", "interval"))
                assert {k: v for k, v in old.items() if k not in fields} == {
                    k: v for k, v in new.items() if k not in fields
                }, f"{relative}/{old['as']}: execution configuration changed"
                assert new["always_run"] is False
                assert new["run_if_changed"] == policy["run_if_changed"]
                assert "skip_if_only_changed" not in new
        assert count == expected_count, (patch, count)
        subprocess.run(["git", "-C", str(args.release_root.resolve()), "apply", "--check", str(directory / patch)], check=True)
        print(f"{patch}: applies cleanly; {count} selector-only changes; execution and non-PR configuration preserved.")


if __name__ == "__main__":
    main()
