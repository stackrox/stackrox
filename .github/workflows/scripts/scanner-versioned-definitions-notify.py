#!/usr/bin/env python3
"""
Notify Slack about scanner vulnerability update failures.

Checks status.json in GCS for each version stream and sends
a Slack notification if any vulnerability updaters failed.

Usage:
    python scanner-versioned-definitions-notify.py --webhook-url $SLACK_WEBHOOK --workflow-url $URL dev v2
    python scanner-versioned-definitions-notify.py --webhook-url $SLACK_WEBHOOK --workflow-url $URL --job-failed dev v2
"""

import argparse
import json
import subprocess
import sys
import tempfile
import urllib.request
from pathlib import Path


GCS_BUCKET = "gs://definitions.stackrox.io/v4/vulnerability-bundles"


def gsutil_copy(src, dest):
    """Copy a file from GCS."""
    return subprocess.run(["gsutil", "cp", src, dest], capture_output=True, text=True)


def gsutil_exists(path):
    """Check if a GCS path exists."""
    result = subprocess.run(["gsutil", "-q", "stat", path], capture_output=True)
    return result.returncode == 0


def get_failed_updaters(status_path):
    """Extract failed updater names from status.json."""
    with open(status_path) as f:
        status = json.load(f)
    return [u["name"] for u in status.get("updaters", []) if u.get("status") == "failed"]


def check_partial_failures(versions):
    """Check each version stream for partial failures. Returns {version: [failed_updaters]}."""
    failures = {}

    with tempfile.TemporaryDirectory() as tmpdir:
        for version in versions:
            status_url = f"{GCS_BUCKET}/{version}/bundles/status.json"
            local_path = Path(tmpdir) / f"status-{version}.json"

            if not gsutil_exists(status_url):
                print(f"status.json not found for {version}, skipping (older git refs may predate partial failure support)")
                continue

            result = gsutil_copy(status_url, local_path)
            if result.returncode != 0:
                print(f"Warning: could not download status.json for {version}: {result.stderr}")
                continue

            failed = get_failed_updaters(local_path)
            if failed:
                failures[version] = failed

    return failures


def send_slack_message(webhook_url, message):
    """Send a message to Slack."""
    data = json.dumps({"text": message}).encode("utf-8")
    req = urllib.request.Request(
        webhook_url,
        data=data,
        headers={"Content-Type": "application/json"},
    )
    try:
        urllib.request.urlopen(req)
        return True
    except urllib.error.URLError as e:
        print(f"Error sending Slack message: {e}")
        return False


def main():
    parser = argparse.ArgumentParser(description="Notify Slack about scanner vulnerability update failures")
    parser.add_argument("--webhook-url", required=True, help="Slack webhook URL")
    parser.add_argument("--workflow-url", required=True, help="URL to workflow run")
    parser.add_argument("--job-failed", action="store_true", help="Indicates the build job failed completely")
    parser.add_argument("versions", nargs="+", help="Version streams to check (e.g., dev v2)")
    args = parser.parse_args()

    partial_failures = check_partial_failures(args.versions)

    if not partial_failures and not args.job_failed:
        print("No failures to report")
        return 0

    if partial_failures:
        details = "\n".join(f"  {v}: {', '.join(f)}" for v, f in sorted(partial_failures.items()))
        message = f"Vulnerability update completed with partial failures:\n{details}\n\nSee <{args.workflow_url}|workflow run> for details."
    else:
        message = f"<{args.workflow_url}|Vulnerability update workflow> failed completely."

    print(f"Sending notification:\n{message}")
    if send_slack_message(args.webhook_url, message):
        print("Notification sent")
        return 0
    else:
        return 1


if __name__ == "__main__":
    sys.exit(main())
