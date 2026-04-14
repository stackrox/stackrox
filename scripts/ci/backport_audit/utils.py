"""Utility functions for PR processing."""

import re
import subprocess
from typing import Any

from .github_client import GitHubClient
from .models import BackportAuditError, GitHubError, ReleaseBranch


def extract_jira_keys(text: str) -> list[str]:
    """Extract ROX-XXXXX Jira keys from text.

    Args:
        text: Text to search (PR title, body, etc.)

    Returns:
        Sorted list of unique Jira keys

    """
    pattern = r"ROX-\d+"
    matches = re.findall(pattern, text)
    return sorted(set(matches))


def find_backport_label_adder(pr_number: int, gh_client: GitHubClient) -> str:
    """Find who added the backport label to a PR.

    Args:
        pr_number: PR number
        gh_client: GitHub client

    Returns:
        GitHub username of label adder, or 'app/dependabot' if not found

    """
    try:
        events = gh_client.get_issue_events(pr_number)
        for event in events:
            if (event.get("event") == "labeled" and
                event.get("label", {}).get("name", "").startswith("backport") and
                event.get("actor", {}).get("login") != "github-actions[bot]"):
                return event["actor"]["login"]
    except GitHubError:
        pass

    return "app/dependabot"


def resolve_author(pr_data: dict[str, Any], gh_client: GitHubClient) -> str:
    """Resolve the real author of a PR.

    Handles rhacs-bot and dependabot by finding the original author
    or the person who added the backport label.

    Args:
        pr_data: PR data from gh CLI
        gh_client: GitHub client

    Returns:
        Resolved author username

    """
    author = pr_data["author"]["login"]
    body = pr_data.get("body", "")

    # Handle rhacs-bot
    if author == "rhacs-bot":
        # Extract original PR number from body
        match = re.search(r"from #(\d+)", body)
        if match:
            original_pr_number = int(match.group(1))
            try:
                original_pr = gh_client.get_pr_details(original_pr_number)
                author = original_pr["author"]["login"]

                # If original author is also dependabot, find label adder
                if author == "app/dependabot":
                    author = find_backport_label_adder(original_pr_number, gh_client)
            except GitHubError:
                pass

    # Handle direct dependabot PRs
    elif author == "app/dependabot":
        pr_number = pr_data["number"]
        author = find_backport_label_adder(pr_number, gh_client)

    return author


def detect_release_branches(branches_arg: str) -> list[str]:
    """Detect release branches from git.

    Args:
        branches_arg: "all" or comma-separated branch names

    Returns:
        List of release branch names

    """
    if branches_arg == "all":
        # Auto-detect from git remote branches
        try:
            result = subprocess.run(
                ["git", "branch", "-r"],
                capture_output=True,
                text=True,
                check=True,
                timeout=10,
            )
            branches = []
            for line in result.stdout.splitlines():
                match = re.search(r"origin/(release-\d+\.\d+)", line)
                if match:
                    branches.append(match.group(1))
            return sorted(set(branches))
        except subprocess.CalledProcessError as e:
            msg = f"Failed to detect release branches: {e.stderr}"
            raise BackportAuditError(msg) from e
        except subprocess.TimeoutExpired as e:
            msg = "Git command timed out"
            raise BackportAuditError(msg) from e
    else:
        # Use provided branches
        return [b.strip() for b in branches_arg.split(",") if b.strip()]


def detect_release_version(branch_name: str) -> ReleaseBranch:
    """Detect expected release version for a branch.

    Args:
        branch_name: Branch name (e.g., release-4.10)

    Returns:
        ReleaseBranch with version info

    """
    # Extract base version
    match = re.match(r"release-(\d+\.\d+)", branch_name)
    if not match:
        msg = f"Invalid branch format: {branch_name}"
        raise BackportAuditError(msg)

    base_version = match.group(1)

    # Find latest tag for this version
    try:
        result = subprocess.run(
            ["git", "tag"],
            capture_output=True,
            text=True,
            check=True,
            timeout=10,
        )

        # Filter tags for this version
        pattern = f"^{re.escape(base_version)}\\." + r"\d+$"
        matching_tags = [
            tag for tag in result.stdout.splitlines()
            if re.match(pattern, tag)
        ]

        latest_tag = None
        if matching_tags:
            # Sort by version (semantic sort)
            matching_tags.sort(key=lambda t: [int(x) for x in t.split(".")])
            latest_tag = matching_tags[-1]

        # Calculate next version
        if latest_tag:
            patch = int(latest_tag.split(".")[-1])
            expected_version = f"{base_version}.{patch + 1}"
        else:
            expected_version = f"{base_version}.0"

        return ReleaseBranch(
            name=branch_name,
            expected_version=expected_version,
            latest_tag=latest_tag,
        )

    except subprocess.CalledProcessError as e:
        msg = f"Failed to detect version for {branch_name}: {e.stderr}"
        raise BackportAuditError(msg) from e
    except subprocess.TimeoutExpired as e:
        msg = "Git command timed out"
        raise BackportAuditError(msg) from e
