#!/usr/bin/env python3
"""Carry out a decision for one OpenShift CI job.

An explicit /test comment for this job runs it even when the decision
skipped it. The comment does not remove a job the decision listed.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from datetime import datetime
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from decisionlib import Decision, DecisionError, parse_decision, read_list  # noqa: E402
from resolver import open_decision  # noqa: E402

_TEST_LINE = re.compile(r"^/test\b(.*)$")


def prow_action(
    job: str,
    decision: Decision | None,
    defaults: dict[str, str],
    enforce: bool,
    forced: bool,
) -> str:
    """prow_action returns run or skip for one OpenShift CI job.

    forced is a /test comment for this job. It runs the job. It does not
    skip a job the decision listed.
    """
    if not enforce or job not in defaults:
        return "run"
    if forced:
        return "run"
    if decision is None:
        return defaults[job]
    return "run" if job in decision.jobs else "skip"


def comment_requests(job: str, body: str) -> bool:
    """comment_requests is true when a /test line names this job or all."""
    for raw in body.splitlines():
        match = _TEST_LINE.match(raw.strip())
        if not match:
            continue
        names = match.group(1).split()
        if job in names or "all" in names:
            return True
    return False


def requests_since(comments: list[dict[str, str]], job: str, commit_time: str) -> bool:
    """requests_since is true when a /test comment was written at or after the commit.

    An older /test comment does not force a later commit.
    """
    if not commit_time:
        return False
    try:
        commit = _parse_time(commit_time)
    except ValueError:
        return False
    for comment in comments:
        body = str(comment.get("body") or "")
        if not comment_requests(job, body):
            continue
        try:
            created = _parse_time(str(comment.get("created_at") or ""))
        except ValueError:
            continue
        if created >= commit:
            return True
    return False


def main(argv: list[str] | None = None) -> int:
    """main prints run or skip. The log goes to stderr."""
    parser = argparse.ArgumentParser(description="Carry out a CI decision for one Prow job.")
    parser.add_argument("command", choices=("gate",))
    parser.add_argument("--job", required=True)
    parser.add_argument("--rules", required=True)
    parser.add_argument("--defaults", required=True)
    parser.add_argument("--labels-file", required=True)
    parser.add_argument("--files-file", default="")
    parser.add_argument("--files-unknown", action="store_true")
    parser.add_argument("--comments-file", default="")
    parser.add_argument("--commit-time", default="")
    parser.add_argument("--repo", required=True)
    parser.add_argument("--pr", required=True)
    parser.add_argument("--commit", required=True)
    parser.add_argument("--enforce", action="store_true")
    args = parser.parse_args(argv)

    if args.files_unknown == bool(args.files_file):
        print("pass either --files-file or --files-unknown", file=sys.stderr)
        return 1

    try:
        files = None if args.files_unknown else read_list(Path(args.files_file).read_text(encoding="utf-8"))
        labels = read_list(Path(args.labels_file).read_text(encoding="utf-8"))
        comments = _read_comments(args.comments_file)
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
    except (OSError, DecisionError, json.JSONDecodeError) as exc:
        print(f"dispatcher failed: {exc}", file=sys.stderr)
        return 1

    forced = requests_since(comments, args.job, args.commit_time)
    print(log_text, file=sys.stderr)
    if forced:
        print(f"a /test comment runs {args.job}", file=sys.stderr)
    action = prow_action(args.job, decision, defaults, args.enforce, forced)
    if not args.enforce:
        action = "run"
    print(action)
    return 0


def _read_comments(path: str) -> list[dict[str, str]]:
    if not path:
        return []
    raw = json.loads(Path(path).read_text(encoding="utf-8"))
    if not isinstance(raw, list):
        raise DecisionError("comments file must be a list")
    comments: list[dict[str, str]] = []
    for item in raw:
        if not isinstance(item, dict):
            raise DecisionError("each comment must be an object")
        comments.append({"created_at": str(item.get("created_at") or ""), "body": str(item.get("body") or "")})
    return comments


def _parse_time(value: str) -> datetime:
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


if __name__ == "__main__":
    sys.exit(main())
