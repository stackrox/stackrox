#!/usr/bin/env python3
"""Carry out a decision for GitHub Actions.

This does not decide again. It reads the decision text. Without the label
ci-dispatcher-enforce the job runs the way it does today, and the log is
only printed. With the label, a listed job runs and any other enrolled job
skips. Skipping still starts the job and leaves it successful, so a required
check can merge.
"""

from __future__ import annotations

import argparse
import os
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from decisionlib import Decision, DecisionError, parse_decision, read_list  # noqa: E402
from resolver import open_decision  # noqa: E402

ENFORCE_LABEL = "ci-dispatcher-enforce"


def gha_action(
    job: str,
    decision: Decision | None,
    defaults: dict[str, str],
    enforce: bool,
) -> str:
    """gha_action returns run or skip for one GitHub Actions job."""
    if not enforce or job not in defaults:
        return "run"
    if decision is None:
        return defaults[job]
    return "run" if job in decision.jobs else "skip"


def main(argv: list[str] | None = None) -> int:
    """main prints the decision, or prints run/skip for one job."""
    parser = argparse.ArgumentParser(description="Carry out a CI decision on GitHub Actions.")
    parser.add_argument("command", choices=("print", "gate"))
    parser.add_argument("--job", default="")
    parser.add_argument("--rules", required=True)
    parser.add_argument("--defaults", required=True)
    parser.add_argument("--labels-file", required=True)
    parser.add_argument("--files-file", default="")
    parser.add_argument("--files-unknown", action="store_true")
    parser.add_argument("--repo", required=True)
    parser.add_argument("--pr", required=True)
    parser.add_argument("--commit", required=True)
    parser.add_argument("--enforce", action="store_true")
    parser.add_argument("--decision-out", default="")
    args = parser.parse_args(argv)

    if args.command == "gate" and not args.job:
        print("gate needs --job", file=sys.stderr)
        return 1
    if args.files_unknown == bool(args.files_file):
        print("pass either --files-file or --files-unknown", file=sys.stderr)
        return 1

    try:
        files = None
        if not args.files_unknown:
            files = _read_lines(args.files_file)
        labels = _read_lines(args.labels_file)
        decision_text, log_text, defaults = open_decision(
            args.rules,
            args.defaults,
            files,
            labels,
            repo=args.repo,
            pr=str(args.pr),
            commit=args.commit,
        )
        decision = parse_decision(decision_text)
    except (OSError, DecisionError) as exc:
        print(f"dispatcher failed: {exc}", file=sys.stderr)
        return 1

    if args.decision_out:
        Path(args.decision_out).write_text(decision_text, encoding="utf-8")

    if args.command == "print":
        _print_plan(decision_text, log_text, args.enforce)
        return 0

    # Ask what the decision itself says, then let a missing label keep today's run.
    planned = gha_action(args.job, decision, defaults, True)
    print(log_text, file=sys.stderr)
    if args.enforce:
        action = planned
        print(f"{args.job}: {action}", file=sys.stderr)
    else:
        action = "run"
        print(
            f"{args.job}: decision would {planned}; "
            f"label {ENFORCE_LABEL} is absent so the job runs",
            file=sys.stderr,
        )
    print(action)
    return 0


def _print_plan(decision_text: str, log_text: str, enforce: bool) -> None:
    """_print_plan writes the decision and the log, and appends them to the GitHub step summary when that file is set."""
    label = "set" if enforce else "absent"
    body = (
        f"ci-dispatcher-enforce is {label}.\n\n"
        f"Decision:\n{decision_text}\n"
        f"Log:\n{log_text}"
    )
    sys.stdout.write(body)
    summary = os.environ.get("GITHUB_STEP_SUMMARY")
    if summary:
        with open(summary, "a", encoding="utf-8") as handle:
            handle.write("## CI decision\n\n```\n" + body + "```\n")


def _read_lines(path: str) -> list[str]:
    """_read_lines reads one path or label per line from a text file."""
    return read_list(Path(path).read_text(encoding="utf-8"))


if __name__ == "__main__":
    sys.exit(main())
