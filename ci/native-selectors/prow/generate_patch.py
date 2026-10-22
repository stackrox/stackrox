#!/usr/bin/env python3
"""Render a source-config patch without changing an openshift/release checkout."""

import argparse
import difflib
import hashlib
import json
from pathlib import Path

import yaml

DIRECTORY = Path(__file__).parent


def render(source, policy, schedule_selected):
    """Replace selector fields only on enrolled presubmit entries."""
    document = yaml.compose(source)
    tests = next(value for key, value in document.value if key.value == "tests")
    changes = []
    for node in tests.value:
        fields = {key.value: value for key, value in node.value}
        # Node serialization retains the sequence entry's nesting for safe parsing.
        entry = yaml.safe_load(yaml.serialize(node))
        target = entry.get("steps", {}).get("env", {}).get("TEST_SUITE", entry["as"])
        if entry.get("postsubmit") or entry.get("cron") or entry.get("interval"):
            continue
        if target not in policy["filtered_suites"]:
            continue
        automatic = entry.get("always_run") is not False or bool(entry.get("run_if_changed") or entry.get("skip_if_only_changed"))
        if not automatic and not schedule_selected:
            continue
        # Preserve all execution fields, job names, timeouts, and optional flags.
        for key, value in node.value:
            if key.value in ("always_run", "run_if_changed", "skip_if_only_changed"):
                start = key.start_mark.index
                end = value.end_mark.index
                changes.append((start, end, ""))
        changes.append((node.start_mark.index, node.start_mark.index,
                        "always_run: false\n  run_if_changed: " + policy["run_if_changed"] + "\n  "))
    for start, end, replacement in sorted(changes, reverse=True):
        source = source[:start] + replacement + source[end:]
    # Empty mapping lines can include the original sequence marker. Fold them
    # into the following key without changing any values.
    source = source.replace("\n  \n", "\n")
    return source


def main():
    """Check pinned inputs and write a reviewable patch for one trigger policy."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--release-root", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--schedule-selected", action="store_true",
                        help="Also enroll currently manual nongroovy and VM presubmits")
    args = parser.parse_args()
    policy = json.loads((DIRECTORY / "selectors.json").read_text())
    patches = []
    for relative, digest in policy["sources"].items():
        source = (args.release_root / relative).read_text()
        if hashlib.sha256(source.encode()).hexdigest() != digest:
            raise ValueError(f"{relative}: source differs from the pinned release snapshot")
        updated = render(source, policy, args.schedule_selected)
        yaml.safe_load(updated)
        patches.extend(difflib.unified_diff(source.splitlines(True), updated.splitlines(True),
                                           fromfile="a/" + relative, tofile="b/" + relative))
    args.output.write_text("".join(patches))


if __name__ == "__main__":
    main()
