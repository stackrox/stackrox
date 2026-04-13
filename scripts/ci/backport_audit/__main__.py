#!/usr/bin/env python3
"""Main orchestration for backport audit tool."""

import json
import os
import pathlib
import re
import subprocess
import sys
import traceback
from datetime import datetime, timezone

from .config import Config, parse_args
from .github_client import GitHubClient
from .jira_client import JiraClient
from .models import PR, BackportAuditError, GitHubError
from .report_markdown import generate_markdown
from .report_slack import generate_slack_payload
from .models import JiraIssue, ReleaseBranch
from .utils import (
    detect_release_branches,
    detect_release_version,
    extract_jira_keys,
    resolve_author,
)

# Default Slack channel for backport audit notifications
DEFAULT_SLACK_CHANNEL = "C05AZF8T7GW"


def _fetch_and_group_prs(
    gh_client: GitHubClient,
    branch_names: list[str],
) -> tuple[dict[str, list[PR]], set[str]]:
    """Fetch PRs and group by branch, collecting Jira keys."""
    all_prs_data = gh_client.fetch_prs("backport", "open")
    prs_by_branch: dict[str, list[PR]] = {}
    all_jira_keys = set()

    for pr_data in all_prs_data:
        base_ref = pr_data["baseRefName"]
        if base_ref not in branch_names:
            continue

        author = resolve_author(pr_data, gh_client)
        jira_keys = extract_jira_keys(pr_data["title"])
        all_jira_keys.update(jira_keys)

        pr = PR(
            number=pr_data["number"],
            title=pr_data["title"],
            author=author,
            base_ref=base_ref,
            jira_keys=jira_keys,
            body=pr_data.get("body", ""),
        )

        if base_ref not in prs_by_branch:
            prs_by_branch[base_ref] = []
        prs_by_branch[base_ref].append(pr)

    return prs_by_branch, all_jira_keys


def _fetch_merged_prs_from_commits(
    gh_client: GitHubClient,
    branches: list[ReleaseBranch],
) -> tuple[dict[str, list[PR]], set[str]]:
    """Fetch merged PRs by parsing git commits after last tag.

    Args:
        gh_client: GitHub client for API calls
        branches: List of release branches with tag info

    Returns:
        Tuple of (PRs grouped by branch, all Jira keys)
    """
    prs_by_branch: dict[str, list[PR]] = {}
    all_jira_keys = set()

    for branch in branches:
        if not branch.latest_tag:
            # No tag yet, skip merged PR collection
            continue

        try:
            # Fetch commits after last tag: hash|subject|author_name|author_email
            result = subprocess.run(
                [
                    "git", "log",
                    f"{branch.latest_tag}..origin/{branch.name}",
                    "--format=%H|%s|%an|%ae"
                ],
                capture_output=True,
                text=True,
                check=True,
                timeout=30,
            )

            for line in result.stdout.strip().splitlines():
                if not line:
                    continue

                parts = line.split("|", 3)
                if len(parts) != 4:
                    continue

                commit_sha, subject, author_name, author_email = parts

                # Extract PR number from commit message like "(#19752)"
                pr_match = re.search(r"\(#(\d+)\)", subject)
                if not pr_match:
                    continue

                pr_number = int(pr_match.group(1))

                # Try to fetch PR details from GitHub API
                try:
                    pr_data = gh_client.get_pr_details(pr_number)
                    title = pr_data.get("title", subject)
                    author = pr_data.get("author", {}).get("login", author_name)
                    body = pr_data.get("body", "")
                except GitHubError:
                    # Fallback to git commit data
                    title = subject
                    author = author_name
                    body = ""

                jira_keys = extract_jira_keys(title)
                all_jira_keys.update(jira_keys)

                pr = PR(
                    number=pr_number,
                    title=title,
                    author=author,
                    base_ref=branch.name,
                    jira_keys=jira_keys,
                    body=body,
                    merged=True,
                    commit_sha=commit_sha,
                )

                if branch.name not in prs_by_branch:
                    prs_by_branch[branch.name] = []
                prs_by_branch[branch.name].append(pr)

        except subprocess.CalledProcessError:
            # Git command failed, skip this branch
            continue
        except subprocess.TimeoutExpired:
            # Git command timed out, skip this branch
            continue

    return prs_by_branch, all_jira_keys


def _fetch_jira_issues(
    jira_client: JiraClient,
    jira_keys: set[str],
) -> dict[str, JiraIssue]:
    """Fetch Jira issues for given keys."""
    jira_issues = {}
    for jira_key in sorted(jira_keys):
        issue = jira_client.get_issue(jira_key)
        if issue:
            jira_issues[jira_key] = issue
    return jira_issues


def _detect_orphaned_issues(
    jira_client: JiraClient,
    branches: list[ReleaseBranch],
    prs_by_branch: dict[str, list[PR]],
    jira_project: str,
) -> dict[str, list[str]]:
    """Detect Jira issues with fixVersion but no corresponding PR."""
    orphaned_issues: dict[str, list[str]] = {}

    for branch in branches:
        jql = f'project = {jira_project} AND fixVersion = "{branch.expected_version}"'
        jira_issues_for_branch = jira_client.search_issues(jql)

        pr_jira_keys = set()
        for pr in prs_by_branch.get(branch.name, []):
            pr_jira_keys.update(pr.jira_keys)

        orphaned = [
            issue.key
            for issue in jira_issues_for_branch
            if issue.key not in pr_jira_keys
        ]

        if orphaned:
            orphaned_issues[branch.name] = orphaned

    return orphaned_issues


def _write_outputs(
    config: Config,
    branches: list[ReleaseBranch],
    prs_by_branch: dict[str, list[PR]],
    jira_issues: dict[str, JiraIssue],
    orphaned_issues: dict[str, list[str]],
    timestamp: str,
) -> None:
    """Write markdown and Slack outputs."""
    pathlib.Path(config.output_dir).mkdir(exist_ok=True, parents=True)

    markdown = generate_markdown(
        branches,
        prs_by_branch,
        jira_issues,
        orphaned_issues,
        timestamp,
    )
    markdown_path = pathlib.Path(config.output_dir) / config.report_file
    markdown_path.write_text(markdown, encoding="utf-8")

    slack_channel = os.getenv("SLACK_CHANNEL", DEFAULT_SLACK_CHANNEL)
    slack_payload = generate_slack_payload(
        branches,
        prs_by_branch,
        jira_issues,
        orphaned_issues,
        timestamp,
        config.github_run_url,
        slack_channel,
    )
    slack_path = pathlib.Path(config.output_dir) / config.slack_payload_file
    with slack_path.open("w", encoding="utf-8") as f:
        json.dump(slack_payload, f, indent=2)


def main() -> int:
    """Orchestrate backport audit: fetch PRs/Jira, generate reports."""
    try:
        args = parse_args()
        config = Config.from_env(args)

        gh_client = GitHubClient()
        jira_client = JiraClient(
            config.jira_user,
            config.jira_token,
            config.jira_base_url,
        )

        branch_names = detect_release_branches(config.branches)
        if not branch_names:
            return 1

        branches = [detect_release_version(name) for name in branch_names]

        # Fetch open PRs
        prs_by_branch, all_jira_keys = _fetch_and_group_prs(gh_client, branch_names)

        # Fetch merged PRs from git commits after last tag (only for branches with open PRs)
        branches_with_open_prs = [b for b in branches if b.name in prs_by_branch]
        merged_prs_by_branch, merged_jira_keys = _fetch_merged_prs_from_commits(
            gh_client, branches_with_open_prs
        )

        # Merge both sources
        for branch_name, merged_prs in merged_prs_by_branch.items():
            if branch_name not in prs_by_branch:
                prs_by_branch[branch_name] = []
            prs_by_branch[branch_name].extend(merged_prs)

        all_jira_keys.update(merged_jira_keys)

        jira_issues = _fetch_jira_issues(jira_client, all_jira_keys)
        orphaned_issues = _detect_orphaned_issues(
            jira_client,
            branches,
            prs_by_branch,
            config.jira_project,
        )

        timestamp = datetime.now(tz=timezone.utc).strftime("%Y-%m-%d %H:%M:%S UTC")

        _write_outputs(
            config,
            branches,
            prs_by_branch,
            jira_issues,
            orphaned_issues,
            timestamp,
        )

        return 0

    except BackportAuditError:
        return 1
    except Exception:
        traceback.print_exc()
        return 2


if __name__ == "__main__":
    sys.exit(main())
