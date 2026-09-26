#!/usr/bin/env python3

"""
Syncs pkg/version/productstreams/major_version_bumps.yaml from master
to supported release branches.

For each supported version, the script:
1. Checks if a release branch exists and contains the bump file
2. Compares the file content with master
3. Creates or updates a sync PR if they differ

Supported versions are resolved from the RH Product Lifecycles API or
from the SUPPORTED_VERSIONS_OVERRIDE environment variable.
"""

import json
import logging
import os
import subprocess
import sys
import time
from urllib.error import URLError
from urllib.request import Request, urlopen

BUMP_FILE = "pkg/version/productstreams/major_version_bumps.yaml"
SYNC_BRANCH_PREFIX = "automated/sync-version-bumps-"
RH_API_URL = (
    "https://access.redhat.com/product-life-cycles/api/v1/products"
    "?name=Red%20Hat%20Advanced%20Cluster%20Security%20for%20Kubernetes"
)
MAX_RETRIES = 3
RETRY_DELAY = 5

logging.basicConfig(
    stream=sys.stderr, level=logging.DEBUG, format="%(levelname)s: %(message)s"
)


def main():
    dry_run = os.environ.get("DRY_RUN", "true") == "true"
    event_name = os.environ.get("GITHUB_EVENT_NAME", "")
    override = os.environ.get("SUPPORTED_VERSIONS_OVERRIDE", "")
    summary_path = os.environ.get("GITHUB_STEP_SUMMARY", "")

    if event_name == "pull_request":
        dry_run = True

    versions = get_supported_versions(override)
    master_content = git_show_safe("origin/master", BUMP_FILE)
    if master_content is None:
        logging.error(f"{BUMP_FILE} not found on master")
        sys.exit(1)

    configure_git()

    results = []
    for version in versions:
        branch = f"release-{version}"
        status, ok = process_branch(branch, master_content, dry_run)
        results.append((version, branch, status, ok))
        logging.info(f"{branch}: {status}")

    write_summary(results, summary_path, dry_run)

    if any(not ok for _, _, _, ok in results):
        sys.exit(1)


def get_supported_versions(override):
    if override:
        versions = [v.strip() for v in override.split(",") if v.strip()]
        logging.info(f"Using override supported versions: {versions}")
        return versions

    for attempt in range(1, MAX_RETRIES + 1):
        try:
            req = Request(url=RH_API_URL, headers={"User-Agent": "Mozilla/5.0"})
            with urlopen(req, timeout=30) as response:
                data = json.loads(response.read().decode("utf-8"))

            versions = [
                v["name"]
                for v in data["data"][0]["versions"]
                if v.get("type") != "End of life"
            ]
            logging.info(f"RH API: supported versions are {versions}")
            return versions
        except (URLError, KeyError, IndexError, json.JSONDecodeError) as e:
            logging.warning(
                f"RH API attempt {attempt}/{MAX_RETRIES} failed: {e}"
            )
            if attempt < MAX_RETRIES:
                time.sleep(RETRY_DELAY)

    logging.error(
        f"RH Product Lifecycles API is unreachable after {MAX_RETRIES} attempts. "
        "Trigger a manual run with the 'supported-versions' input to override."
    )
    sys.exit(1)


def process_branch(branch, master_content, dry_run):
    ref = f"origin/{branch}"

    if not branch_exists(ref):
        return ":grey_question: Branch does not exist", True

    branch_content = git_show_safe(ref, BUMP_FILE)
    if branch_content is None:
        return ":fast_forward: Branch exists but bump file not present", True

    if master_content == branch_content:
        return ":white_check_mark: File matches master", True

    if dry_run:
        return ":fast_forward: File differs (dry-run, no PR created)", True

    return create_or_update_pr(branch, master_content)


def create_or_update_pr(branch, master_content):
    sync_branch = f"{SYNC_BRANCH_PREFIX}{branch}"

    try:
        existing_pr = find_open_pr(sync_branch, branch)
    except subprocess.CalledProcessError as e:
        return ":x: Failed to query open PRs", False

    if existing_pr:
        pr_number = existing_pr["number"]
        sync_content = git_show_safe(f"origin/{sync_branch}", BUMP_FILE)
        if sync_content == master_content:
            return f":white_check_mark: PR #{pr_number} already has latest content", True

        try:
            update_sync_branch(sync_branch, master_content)
            return f":arrows_counterclockwise: PR #{pr_number} updated with new commit", True
        except subprocess.CalledProcessError as e:
            return f":x: Failed to update PR #{pr_number}", False
    else:
        try:
            cleanup_stale_branch(sync_branch)
            create_sync_branch(sync_branch, branch, master_content)
            pr_number = open_pr(sync_branch, branch)
            return f":arrow_right: PR #{pr_number} created", True
        except subprocess.CalledProcessError as e:
            return ":x: Failed to create PR", False


def find_open_pr(head, base):
    result = run(
        ["gh", "api", "repos/{owner}/{repo}/pulls", "--method", "GET",
         "--raw-field", "state=open", "--raw-field", f"base={base}",
         "--raw-field", f"head=stackrox:{head}", "--raw-field", "per_page=1"]
    )
    prs = json.loads(result.stdout)
    return {"number": prs[0]["number"]} if prs else None


def cleanup_stale_branch(sync_branch):
    """Delete a remote sync branch if it exists without an open PR."""
    if branch_exists(f"origin/{sync_branch}"):
        logging.info(f"Deleting stale remote branch {sync_branch}")
        run(["git", "push", "origin", "--delete", sync_branch], check=False)


def create_sync_branch(sync_branch, base_branch, master_content):
    try:
        run(["git", "checkout", "-B", sync_branch, f"origin/{base_branch}"])
        with open(BUMP_FILE, "w") as f:
            f.write(master_content)
        run(["git", "add", BUMP_FILE])
        run(["git", "commit", "-m", "chore: sync major_version_bumps.yaml from master"])
        # --force-with-lease in case cleanup_stale_branch failed to delete the remote branch.
        run(["git", "push", "--force-with-lease", "-u", "origin", sync_branch])
    finally:
        reset_worktree()


def update_sync_branch(sync_branch, master_content):
    try:
        run(["git", "fetch", "origin", sync_branch])
        run(["git", "checkout", "-B", sync_branch, f"origin/{sync_branch}"])
        with open(BUMP_FILE, "w") as f:
            f.write(master_content)
        run(["git", "add", BUMP_FILE])
        run(["git", "commit", "-m", "chore: sync major_version_bumps.yaml from master"])
        run(["git", "push", "origin", sync_branch])
    finally:
        reset_worktree()


def open_pr(sync_branch, base_branch):
    result = run([
        "gh", "pr", "create",
        "--head", sync_branch,
        "--base", base_branch,
        "--draft",
        "--title", f"chore({base_branch}): sync major_version_bumps.yaml from master",
        "--body", (
            f"Automated sync of `{BUMP_FILE}` from `master` to `{base_branch}`.\n\n"
            "This keeps the major version bump history up to date so that builds "
            "from this release branch compute correct version compatibility ranges."
        ),
    ])
    pr_url = result.stdout.strip()
    pr_number = pr_url.rstrip("/").split("/")[-1]
    return pr_number


def configure_git():
    name = os.environ.get("GIT_AUTHOR_NAME", "rhacs-bot")
    email = os.environ.get("GIT_AUTHOR_EMAIL", "rhacs-bot@redhat.com")
    run(["git", "config", "user.name", name])
    run(["git", "config", "user.email", email])


def reset_worktree():
    run(["git", "reset", "HEAD", "--", "."], check=False)
    run(["git", "checkout", "--", "."], check=False)
    run(["git", "clean", "-fd"], check=False)


def branch_exists(ref):
    return run(["git", "rev-parse", "--verify", ref], check=False).returncode == 0


def git_show_safe(ref, path):
    result = run(["git", "show", f"{ref}:{path}"], check=False)
    return result.stdout if result.returncode == 0 else None


def run(cmd, check=True):
    logging.debug(f"Running: {' '.join(cmd)}")
    result = subprocess.run(
        cmd,
        capture_output=True,
        text=True,
        check=False,
    )
    if check and result.returncode != 0:
        stderr = result.stderr.strip() if result.stderr else ""
        logging.error(f"Command failed: {' '.join(cmd)}\n{stderr}")
        raise subprocess.CalledProcessError(result.returncode, cmd)
    return result


def write_summary(results, summary_path, dry_run):
    lines = ["## Version bump sync summary\n"]

    if dry_run:
        lines.append("> **Dry-run mode** — no PRs were created or updated.\n")

    lines.append("| Version | Branch | Status |")
    lines.append("|---------|--------|--------|")
    for version, branch, status, _ in results:
        lines.append(f"| {version} | {branch} | {status} |")
    lines.append("")

    summary = "\n".join(lines)
    logging.info(f"\n{summary}")

    if summary_path:
        with open(summary_path, "a") as f:
            f.write(summary)


if __name__ == "__main__":
    main()
