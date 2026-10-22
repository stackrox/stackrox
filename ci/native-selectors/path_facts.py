#!/usr/bin/env python3
"""Report path predicates without interpreting jobs, labels, or dependencies."""

import argparse
import json
import os
from pathlib import Path
import re
import subprocess


def classify(files, patterns):
    """Return matching predicate names; an empty diff never matches every."""
    result = {"any": [], "every": []}
    for name, sources in patterns.items():
        if not sources or not all(isinstance(source, str) for source in sources):
            raise ValueError(f"{name}: expected a nonempty list of patterns")
        # Go's end anchor does not match before a trailing newline in a filename.
        compiled = [re.compile(source[:-1] + r"\Z" if source.endswith("$") else source)
                    for source in sources]
        matches = [any(pattern.search(path) for pattern in compiled) for path in files]
        if any(matches):
            result["any"].append(name)
        if matches and all(matches):
            result["every"].append(name)
    return result


def changed_files(base, head):
    """Read the three-dot diff, including deleted paths and rename destinations."""
    for commit in (base, head):
        if not re.fullmatch(r"[0-9a-fA-F]{40,64}", commit):
            raise ValueError("Expected full base and head commit IDs")
    output = subprocess.check_output(
        ["git", "diff", "--name-only", "-z", f"{base}...{head}", "--"]
    ).decode("utf-8")
    return output.removesuffix("\0").split("\0") if output else []


def main():
    """Write facts atomically after collection and pattern validation succeed."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--files", type=Path, help="JSON path list for local validation")
    args = parser.parse_args()
    files = (
        json.loads(args.files.read_text()) if args.files
        else changed_files(os.environ.get("CI_BASE", ""), os.environ.get("CI_HEAD", ""))
    )
    if not isinstance(files, list) or not all(isinstance(path, str) for path in files):
        raise ValueError("Expected a JSON list of paths")
    patterns = json.loads(Path(__file__).with_name("paths.json").read_text())
    facts = classify(files, patterns)
    output = "".join(f"{key}={json.dumps(value)}\n" for key, value in facts.items())
    output += "known=true\n"
    if os.environ.get("GITHUB_OUTPUT"):
        with open(os.environ["GITHUB_OUTPUT"], "a", encoding="utf-8") as stream:
            stream.write(output)
    print(output, end="")


if __name__ == "__main__":
    main()
