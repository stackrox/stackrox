#!/usr/bin/env python3
"""Read completed Actions runs/artifacts and reject unrelated negative failures."""

import argparse
import json
from pathlib import Path
import re
import subprocess
import sys

import lib


REPOSITORY = "stackrox/stackrox"
TEST_BRANCH = "alkmim/ROX-36961-t1-negative-tests"


def api(path):
    return json.loads(lib.command(["gh", "api", "--method", "GET", path]))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, required=True)
    parser.add_argument("--run", type=int, required=True)
    parser.add_argument("--case", choices=("healthy", "gate-control", "W5", "W6", "W7", "W9"), required=True)
    parser.add_argument("--expected-sha", required=True)
    parser.add_argument("--evidence", type=Path, required=True)
    parser.add_argument("--healthy-assessment", type=Path)
    parser.add_argument("--gate-assessment", type=Path)
    args = parser.parse_args()
    if not re.fullmatch(r"[0-9a-f]{40}", args.expected_sha):
        raise RuntimeError("Provide the reviewed full test-branch SHA")
    evidence = args.evidence.resolve()
    evidence.mkdir(parents=True, exist_ok=False)
    positive = args.case in ("healthy", "gate-control")
    assessment = {"case": args.case, "implementation_commit": lib.BASE, "test_commit": args.expected_sha, "status": "Failed"}
    assessment["builds"] = {}
    healthy_control = gate_control = None
    try:
        if args.case != "healthy":
            if args.healthy_assessment is None:
                raise RuntimeError("A successful real healthy-upload control is required first")
            control = json.loads(args.healthy_assessment.read_text())
            if control["status"] != "Passed" or control["case"] != "healthy" or control["test_commit"] != args.expected_sha:
                raise RuntimeError("Healthy control is not a successful matching test revision")
            if control["implementation_commit"] != lib.BASE:
                raise RuntimeError("Healthy control has a different implementation base")
            healthy_control = control
            assessment["healthy_run"] = control["run_url"]
        if args.case == "W9":
            if args.gate_assessment is None:
                raise RuntimeError("W9 requires the successful SQL gate-release control")
            control = json.loads(args.gate_assessment.read_text())
            if control["status"] != "Passed" or control["case"] != "gate-control" or control["test_commit"] != args.expected_sha:
                raise RuntimeError("Gate-release control does not match the reviewed test revision")
            gate_control = control
            assessment["gate_control_run"] = control["run_url"]
        run = api(f"repos/{REPOSITORY}/actions/runs/{args.run}")
        lib.write_json(evidence / "run.json", run)
        assessment["run_url"] = run["html_url"]
        if run["status"] != "completed" or run["event"] != "workflow_dispatch" or run["head_sha"] != args.expected_sha or run["head_branch"] != TEST_BRANCH:
            raise RuntimeError("Run is unfinished or not the reviewed disposable test ref")
        if run["conclusion"] != ("success" if positive else "failure"):
            raise RuntimeError("Overall run conclusion does not match its intended control/fault")
        jobs = []
        page = 1
        while True:
            batch = api(f"repos/{REPOSITORY}/actions/runs/{args.run}/jobs?filter=latest&per_page=100&page={page}")["jobs"]
            jobs.extend(batch)
            if len(batch) < 100:
                break
            page += 1
        lib.write_json(evidence / "jobs.json", jobs)
        lib.command([
            "gh", "run", "download", str(args.run), "--repo", REPOSITORY,
            "--pattern", f"t1-{args.case}-*-{run['run_attempt']}",
            "--dir", str(evidence / "artifacts"),
        ], timeout=300)
        configured = set(lib.document(args.repo.resolve(), lib.CONFIG)["bundles"])
        seen = set()
        diagnostics = {
            "W5": r"(?:##\[error\]|::error::)unexpected ZIP entry: t1-negative-unexpected",
            "W6": r"manual: invalid vulnerability source",
            "W9": r"database|connection|SQLSTATE|context canceled|conn closed",
        }
        for manifest in (evidence / "artifacts").glob("*/run.json"):
            details = json.loads(manifest.read_text())
            stream = details["stream"]
            if stream in seen or stream not in configured:
                raise RuntimeError("Duplicate or unknown stream artifact")
            seen.add(stream)
            if details["implementation_commit"] != lib.BASE or details["test_commit"] != args.expected_sha or details["case"] != args.case or str(details["run_id"]) != str(args.run) or str(details["run_attempt"]) != str(run["run_attempt"]):
                raise RuntimeError("Artifact provenance does not match the requested run/case")
            if details["dry_run"] != "true" or details["force"] != "true":
                raise RuntimeError("Artifact inputs are not the required force/dry-run pair")
            checked = json.loads(manifest.with_name("object-check.json").read_text())
            if checked["status"] != "Passed" or checked["http_status"] != (200 if positive else 404):
                raise RuntimeError("Missing successful real object/read-back or authenticated absence check")
            matching = [job for job in jobs if job["name"] == f"bundle ({stream})"]
            if len(matching) != 1 or matching[0]["conclusion"] != ("success" if positive else "failure"):
                raise RuntimeError(f"Unexpected bundle-job result for {stream}")
            # The run's job list can retain non-final step snapshots after completion.
            job = api(f"repos/{REPOSITORY}/actions/jobs/{matching[0]['id']}")
            lib.write_json(evidence / f"{stream}-job.json", job)
            if job["run_id"] != args.run or job["status"] != "completed" or job["conclusion"] != ("success" if positive else "failure"):
                raise RuntimeError(f"Final job metadata is unavailable or inconsistent for {stream}")
            log = lib.command(["gh", "api", "--method", "GET", f"repos/{REPOSITORY}/actions/jobs/{job['id']}/logs"], timeout=180).decode(errors="replace")
            (evidence / f"{stream}-job.log").write_text(log)
            steps = {step["name"]: step["conclusion"] for step in job["steps"]}
            if not positive:
                gate = {"W5": "Validate ZIP entries", "W6": "Validate source content", "W7": "Validate bundle import", "W9": "Validate bundle import"}[args.case]
                if steps.get(gate) != "failure" or steps.get("Upload bundle") != "skipped":
                    raise RuntimeError("Failure occurred at the wrong workflow gate")
                if args.case != "W7" and not re.search(diagnostics[args.case], log, re.IGNORECASE):
                    raise RuntimeError("The intended gate diagnostic is absent from the real job log")
                if args.case == "W6" and "missing or empty source file" not in log:
                    raise RuntimeError("W6 did not report the intended empty-source cause")
                if args.case == "W7":
                    imported = json.loads(manifest.with_name("import-result.json").read_text())
                    text = manifest.with_name("real-import.log").read_text(errors="replace")
                    if imported["import_exit"] != 1 or imported["timed_out"] or not lib.manual_json_error(text):
                        raise RuntimeError("W7 did not report the intended JSON error for the manual bundle in actual import output")
            if args.case in ("healthy", "gate-control", "W7", "W9"):
                build = json.loads(manifest.with_name("updater-build.json").read_text())
                if build["stream"] != stream or not re.fullmatch(r"[0-9a-f]{40}", build["resolved_commit"]):
                    raise RuntimeError("Actual stream-build provenance is missing")
                for control in (healthy_control, gate_control):
                    if control is not None:
                        prior = control["builds"][stream]
                        if any(build[key] != prior[key] for key in ("resolved_commit", "go_mod_sha256", "claircore")):
                            raise RuntimeError("A stream ref changed since its positive control; reestablish the matching control before assessing this run")
                assessment["builds"][stream] = build
        if seen != configured:
            raise RuntimeError(f"Missing evidence for configured streams: {sorted(configured - seen)}")
        assessment["validation_status"] = "Passed"
        assessment["publication_gate"] = "Positive control" if positive else "Passed (W8 real positive/negative pair)"
        notify = [job for job in jobs if job["name"] == "notify"]
        if not positive and len(notify) == 1:
            final_notify = api(f"repos/{REPOSITORY}/actions/jobs/{notify[0]['id']}")
            lib.write_json(evidence / "notify-job.json", final_notify)
            if final_notify["run_id"] != args.run or final_notify["status"] != "completed":
                raise RuntimeError("Final notification-job metadata is unavailable or inconsistent")
            notify = [final_notify]
        assessment["notification_job"] = notify[0]["conclusion"] if len(notify) == 1 else "missing"
        assessment["slack_delivery"] = "Not expected" if positive else "Pending: retain a real test-channel message reference; job success alone is not delivery proof"
        if not positive and assessment["notification_job"] != "success":
            raise RuntimeError("Validation passed, but the failure-notification job failed or is missing")
        assessment["status"] = "Passed"
    except Exception as error:
        assessment["error"] = str(error)
    lib.write_json(evidence / "assessment.json", assessment)
    print(json.dumps(assessment, indent=2))
    return 0 if assessment["status"] == "Passed" else 1


if __name__ == "__main__":
    sys.exit(main())
