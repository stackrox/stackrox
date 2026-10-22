#!/usr/bin/env python3
"""Record decisions from the pinned PR implementation in a temporary source archive."""

import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile


def main():
    """Run the baseline's own decision functions without changing a checkout."""
    directory = Path(__file__).resolve().parent
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    baseline = json.loads((directory / "baseline.json").read_text())
    with tempfile.TemporaryDirectory(prefix="native-ci-oracle-") as temp:
        root = Path(temp)
        archive = root / "source.tar"
        with archive.open("wb") as output:
            subprocess.run(["git", "-C", str(directory.parents[1]), "archive", baseline["sha"],
                            "ci/select", "ci/test-domains.yaml", "ci/decision-defaults", "go.mod", "go.sum"],
                           stdout=output, check=True)
        subprocess.run(["tar", "-xf", str(archive), "-C", temp], check=True)
        fixture = root / "ci/select/native_oracle_test.go"
        shutil.copyfile(directory / "testdata/oracle_test.go.txt", fixture)
        subprocess.run(["gofmt", "-w", str(fixture)], check=True)
        output = root / "expected.json"
        env = os.environ | {
            "NATIVE_SCENARIOS": str(directory / "testdata/scenarios.json"),
            "NATIVE_EXPECTED": str(output),
        }
        subprocess.run(["go", "test", "./ci/select", "-run", "^TestNativeOracle$", "-count=1"],
                       cwd=root, env=env, check=True)
        decisions = json.loads(output.read_text())
        args.output.write_text("{\n" + ",\n".join(
            "  " + json.dumps(name) + ": " + json.dumps(value, separators=(",", ":"))
            for name, value in sorted(decisions.items())
        ) + "\n}\n")


if __name__ == "__main__":
    main()
