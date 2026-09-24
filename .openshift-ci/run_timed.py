#!/usr/bin/env python3

"""Run a CI command with best-effort e2e timing sentinels."""

import argparse
import os
import subprocess
import sys

from e2e_timing import record_skipped, timed_span


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--phase", required=True)
    parser.add_argument("--name", required=True)
    parser.add_argument("command", nargs=argparse.REMAINDER)
    args = parser.parse_args(argv)

    command = args.command
    if command and command[0] == "--":
        command = command[1:]
    if not command:
        parser.error("a command is required after --")

    infra_only = os.getenv("E2E_INFRA_ONLY", "false").lower() == "true"
    attributes = {"infra_only": str(infra_only).lower()}
    if infra_only:
        for phase in ("test-execution", "post-test-collection"):
            record_skipped(
                phase,
                args.name,
                reason="e2e-infra-only",
                attributes=attributes,
            )

    try:
        with timed_span(args.phase, args.name, attributes=attributes):
            subprocess.run(command, check=True)
    except subprocess.CalledProcessError as err:
        return err.returncode if err.returncode >= 0 else 128 - err.returncode
    except KeyboardInterrupt:
        return 130

    return 0


if __name__ == "__main__":
    sys.exit(main())
