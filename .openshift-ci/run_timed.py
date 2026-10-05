#!/usr/bin/env python3

"""Run a CI command with best-effort e2e timing sentinels."""

import argparse
import os
import signal
import subprocess
import sys

from e2e_timing import child_timing_environment, record_skipped, timed_span


_EVENT_PREFIXES = ("e2e_timing ", "e2e_parallel_audit ")
_TRUE_VALUES = {"1", "true", "yes", "on"}


def _audit_event_file(phase):
    audit_enabled = os.getenv("E2E_PARALLELIZATION_AUDIT_ENABLED", "").lower()
    artifact_dir = os.getenv("ARTIFACT_DIR")
    if phase != "test-lane-command" or audit_enabled not in _TRUE_VALUES or not artifact_dir:
        return None
    return os.path.join(artifact_dir, "e2e-events.jsonl")


def _run_with_event_capture(command, env, event_file):
    """Stream child output unchanged while retaining only E2E event records."""
    try:
        event_output = open(event_file, "a", encoding="utf-8")
    except OSError:
        return subprocess.run(command, check=True, env=env)

    process = None
    try:
        process = subprocess.Popen(
            command,
            stdout=subprocess.PIPE,
            env=env,
            text=True,
            encoding="utf-8",
            errors="replace",
            bufsize=1,
        )
        assert process.stdout is not None
        for line in process.stdout:
            sys.stdout.write(line)
            sys.stdout.flush()
            if any(prefix in line for prefix in _EVENT_PREFIXES):
                try:
                    event_output.write(line)
                    event_output.flush()
                except OSError:
                    event_output.close()
                    event_output = None
        return_code = process.wait()
    except KeyboardInterrupt:
        if process is not None and process.poll() is None:
            process.send_signal(signal.SIGINT)
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.terminate()
                process.wait()
        raise
    finally:
        if process is not None and process.stdout is not None:
            process.stdout.close()
        if event_output is not None:
            event_output.close()

    if return_code:
        raise subprocess.CalledProcessError(return_code, command)


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
    event_file = _audit_event_file(args.phase)
    if infra_only and args.phase == "test-lane-command":
        for phase in ("test-execution", "post-test-collection"):
            record_skipped(
                phase,
                args.name,
                reason="e2e-infra-only",
                attributes=attributes,
            )

    try:
        with timed_span(
            args.phase,
            args.name,
            attributes=attributes,
            event_file=event_file,
        ):
            if event_file:
                _run_with_event_capture(
                    command,
                    child_timing_environment(),
                    event_file,
                )
            else:
                subprocess.run(
                    command,
                    check=True,
                    env=child_timing_environment(),
                )
    except subprocess.CalledProcessError as err:
        return err.returncode if err.returncode >= 0 else 128 - err.returncode
    except KeyboardInterrupt:
        return 130

    return 0


if __name__ == "__main__":
    sys.exit(main())
