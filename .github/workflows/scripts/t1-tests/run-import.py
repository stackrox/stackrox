#!/usr/bin/env python3
"""Capture the original import block's real output and propagate its exit code."""

import os
from pathlib import Path
import signal
import subprocess
import sys

import lib


def main():
    evidence = Path(os.environ["T1_EVIDENCE"])
    script = Path("/tmp/t1-tests/Validate-bundle-import.sh")
    log = evidence / "real-import.log"
    result = {"script_sha256": lib.digest(script), "import_exit": None, "timed_out": False}
    process = None
    try:
        with log.open("wb") as output:
            process = subprocess.Popen(["bash", str(script)], stdout=output, stderr=subprocess.STDOUT, start_new_session=True)
            result["import_exit"] = process.wait(timeout=3600)
        return result["import_exit"]
    except subprocess.TimeoutExpired:
        result["timed_out"] = True
        raise
    finally:
        if process is not None and process.poll() is None:
            os.killpg(process.pid, signal.SIGTERM)
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                os.killpg(process.pid, signal.SIGKILL)
                process.wait(timeout=10)
        lib.write_json(evidence / "import-result.json", result)
        if log.exists():
            print(log.read_text(errors="replace"), end="")


if __name__ == "__main__":
    sys.exit(main())
