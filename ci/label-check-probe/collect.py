#!/usr/bin/env python3

import argparse
from collections import Counter
from datetime import UTC, datetime
import json
import os
from pathlib import Path
import subprocess
import time
from urllib.parse import quote, urlencode


def command_json(arguments):
    try:
        result = subprocess.run(
            arguments, check=True, capture_output=True, text=True, timeout=60
        )
        return json.loads(result.stdout)
    except (OSError, subprocess.SubprocessError, ValueError) as error:
        stderr = getattr(error, "stderr", None)
        if isinstance(stderr, bytes):
            stderr = stderr.decode(errors="replace")
        return {
            "error": str(error),
            "stderr": stderr,
        }


def api(endpoint, paginate=False):
    arguments = ["gh", "api", endpoint]
    if paginate:
        arguments.extend(["--paginate", "--slurp"])
    return command_json(arguments)


def pages(value, key):
    if not isinstance(value, list):
        return []
    return [item for page in value for item in page.get(key, [])]


def snapshot(repository, number, destination):
    destination.mkdir(parents=True, exist_ok=True)

    def save(name, value):
        (destination / f"{name}.json").write_text(json.dumps(value, indent=2) + "\n")
        return value

    prefix = f"repos/{repository}"
    pull = save("pull", api(f"{prefix}/pulls/{number}"))
    rollup = save("rollup", command_json([
        "gh", "pr", "view", str(number), "--repo", repository, "--json",
        "url,headRefOid,baseRefName,mergeable,mergeStateStatus,statusCheckRollup",
    ]))
    if "head" not in pull:
        raise RuntimeError(f"Cannot read PR metadata: {pull}")

    base = quote(pull["base"]["ref"], safe="")
    rules = save("rules", api(f"{prefix}/rules/branches/{base}"))
    save("branch-protection", api(f"{prefix}/branches/{base}/protection/required_status_checks"))
    active_rules = rules if isinstance(rules, list) else []
    required = [
        check
        for rule in active_rules
        if rule.get("type") == "required_status_checks"
        for check in rule.get("parameters", {}).get("required_status_checks", [])
    ]
    shas = sorted({sha for sha in [pull["head"]["sha"], pull.get("merge_commit_sha"), os.environ.get("GITHUB_SHA")] if sha})
    summary = {
        "captured_at": datetime.now(UTC).isoformat(),
        "head_sha": pull["head"]["sha"],
        "merge_sha": pull.get("merge_commit_sha"),
        "required_checks": required,
        "merge_state": rollup,
        "check_names": {},
        "duplicate_names": {},
    }
    for sha in shas:
        checks = save(f"checks-{sha}", api(
            f"{prefix}/commits/{sha}/check-runs?filter=all&per_page=100", paginate=True
        ))
        save(f"statuses-{sha}", api(f"{prefix}/commits/{sha}/statuses?per_page=100", paginate=True))
        items = pages(checks, "check_runs")
        summary["check_names"][sha] = [
            {key: check.get(key) for key in ["id", "name", "status", "conclusion", "check_suite", "app", "details_url", "started_at", "completed_at"]}
            for check in items
        ]
        counts = Counter(check["name"] for check in items)
        summary["duplicate_names"][sha] = {name: count for name, count in counts.items() if count > 1}

    query = urlencode({"branch": pull["head"]["ref"], "event": "pull_request", "per_page": 30})
    runs = save("runs", api(f"{prefix}/actions/runs?{query}"))
    for run in runs.get("workflow_runs", []):
        if run["head_sha"] not in shas:
            continue
        if run["name"] not in ["Unit Tests", "e2e-dispatch", "CI label probe evidence"]:
            continue
        run_id = run["id"]
        save(f"run-{run_id}", api(f"{prefix}/actions/runs/{run_id}"))
        save(f"jobs-{run_id}", api(f"{prefix}/actions/runs/{run_id}/jobs?filter=all&per_page=100", paginate=True))
    save("summary", summary)
    print(f"Saved snapshot to {destination}", flush=True)


def main():
    parser = argparse.ArgumentParser(description="Read-only evidence for label/check aggregation.")
    parser.add_argument("--repository", default=os.environ.get("GITHUB_REPOSITORY"))
    parser.add_argument("--pr", type=int)
    parser.add_argument("--output", type=Path, default=Path("label-probe-evidence"))
    parser.add_argument("--intervals", default="0,20,60", help="Snapshot offsets in seconds.")
    arguments = parser.parse_args()
    event_path = os.environ.get("GITHUB_EVENT_PATH")
    event = json.loads(Path(event_path).read_text()) if event_path else {}
    number = arguments.pr or event.get("number")
    if not arguments.repository or not number:
        parser.error("Specify --repository and --pr outside a pull_request workflow.")
    context = {
        "action": event.get("action"),
        "label": event.get("label", {}).get("name"),
        "run_id": os.environ.get("GITHUB_RUN_ID"),
        "run_attempt": os.environ.get("GITHUB_RUN_ATTEMPT"),
        "event_head_sha": event.get("pull_request", {}).get("head", {}).get("sha"),
        "tested_sha": os.environ.get("GITHUB_SHA"),
    }
    arguments.output.mkdir(parents=True, exist_ok=True)
    (arguments.output / "event.json").write_text(json.dumps(context, indent=2) + "\n")
    started = time.monotonic()
    for index, offset in enumerate(map(int, arguments.intervals.split(","))):
        time.sleep(max(0, started + offset - time.monotonic()))
        snapshot(arguments.repository, number, arguments.output / f"snapshot-{index:02}")


if __name__ == "__main__":
    main()
