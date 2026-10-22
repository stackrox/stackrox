#!/usr/bin/env python3
"""Print explicit Prow commands for labels; never post a comment or select paths."""

import argparse
import json
from pathlib import Path


def requested_commands(labels, commands, statuses, schedule_selected=False):
    """Request additional work without restarting pending or successful checks."""
    # Statuses must contain only the latest status per context for the PR head.
    wanted = set()
    if "ci-run-all-tests" in labels:
        wanted.update(commands if schedule_selected else ["gke-qa-e2e-tests", "gke-ui-e2e-tests"])
    if "area/ui" in labels:
        wanted.add("gke-ui-e2e-tests")
    return [commands[name] for name in sorted(wanted)
            if statuses.get("ci/prow/" + name) not in ("pending", "success")]


def main():
    """Read a saved event and print a plan that a trusted bridge could submit."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("event", type=Path)
    parser.add_argument("--statuses", required=True, type=Path,
                        help="JSON map of current-head status contexts to latest states")
    parser.add_argument("--schedule-selected", action="store_true")
    args = parser.parse_args()
    event = json.loads(args.event.read_text())
    pr = event["pull_request"]
    if pr["state"] != "open" or pr["draft"] or pr["head"]["repo"]["fork"]:
        return
    commands = json.loads(Path(__file__).with_name("selectors.json").read_text())["commands"]
    labels = [label["name"] for label in pr["labels"]]
    statuses = json.loads(args.statuses.read_text())
    print("\n".join(requested_commands(labels, commands, statuses, args.schedule_selected)))


if __name__ == "__main__":
    main()
