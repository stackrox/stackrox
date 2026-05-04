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
    github_notice,
    github_warning,
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
        title_keys = extract_jira_keys(pr_data["title"])
        body_keys = extract_jira_keys(pr_data.get("body", ""))
        jira_keys = sorted(set(title_keys + body_keys))
        all_jira_keys.update(jira_keys)

        pr = PR(
            number=pr_data["number"],
            title=pr_data["title"],
            author=author,
            base_ref=base_ref,
            jira_keys=jira_keys,
            body=pr_data.get("body", ""),
            state=pr_data.get("state", "open"),
        )

        prs_by_branch.setdefault(base_ref, []).append(pr)

    return prs_by_branch, all_jira_keys


def _fetch_merged_prs_from_commits(
    gh_client: GitHubClient,
    branches: list[ReleaseBranch],
) -> tuple[dict[str, list[PR]], set[str]]:
    """Fetch merged PRs by parsing git commits after last tag.

    Uses latest_tag instead of current_version because current_version might be
    an RC tag, which would hide most merged PRs for the release.
    """
    prs_by_branch: dict[str, list[PR]] = {}
    all_jira_keys = set()

    for branch in branches:
        base_ref = branch.latest_tag or branch.current_version
        if not base_ref:
            continue

        try:
            result = subprocess.run(
                [
                    "git", "log",
                    f"{base_ref}..origin/{branch.name}",
                    "--format=%H|%an|%ae|%s"
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
                    raise BackportAuditError(f"Unexpected git log format: {line}")

                commit_sha, author_name, author_email, subject = parts

                pr_match = re.search(r"\(#(\d+)\)", subject)
                if not pr_match:
                    print(f"WARNING: commit without PR number {commit_sha} {subject}", file=sys.stderr)
                    continue

                pr_number = int(pr_match.group(1))

                try:
                    pr_data = gh_client.get_pr_details(pr_number)
                    title = pr_data.get("title", subject)
                    author = resolve_author(pr_data, gh_client)
                    body = pr_data.get("body", "")
                except GitHubError:
                    title = subject
                    author = author_name
                    body = ""

                title_keys = extract_jira_keys(title)
                body_keys = extract_jira_keys(body)
                jira_keys = sorted(set(title_keys + body_keys))
                all_jira_keys.update(jira_keys)

                pr = PR(
                    number=pr_number,
                    title=title,
                    author=author,
                    base_ref=branch.name,
                    jira_keys=jira_keys,
                    body=body,
                    merged=True,
                    state="merged",
                    commit_sha=commit_sha,
                )

                prs_by_branch.setdefault(branch.name, []).append(pr)

        except subprocess.CalledProcessError as e:
            github_warning(f"git log failed for branch {branch.name}: {e}")
            continue
        except subprocess.TimeoutExpired as e:
            github_warning(f"git log timed out for branch {branch.name}: {e}")
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
        try:
            jira_issues_for_branch = jira_client.search_issues(jql)
        except Exception as e:
            github_warning(f"Jira search failed for {branch.expected_version}: {e}")
            continue

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

        prs_by_branch, all_jira_keys = _fetch_and_group_prs(gh_client, branch_names)

        merged_prs_by_branch, merged_jira_keys = _fetch_merged_prs_from_commits(
            gh_client, branches
        )

        for branch_name, merged_prs in merged_prs_by_branch.items():
            prs_by_branch.setdefault(branch_name, []).extend(merged_prs)

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

        github_notice(f"Backport audit completed for branches: {', '.join(branch_names)}")

        return 0

    except BackportAuditError as e:
        print(f"ERROR: {e}", file=sys.stderr)
        traceback.print_exc()
        return 1
    except Exception:
        traceback.print_exc()
        return 2


if __name__ == "__main__":
    sys.exit(main())
