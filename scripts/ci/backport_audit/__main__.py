#!/usr/bin/env python3
"""Main orchestration for backport audit tool."""

import json
import os
import pathlib
import sys
import traceback
from datetime import datetime

from .config import Config, parse_args
from .github_client import GitHubClient
from .jira_client import JiraClient
from .models import PR, BackportAuditError
from .report_markdown import generate_markdown
from .report_slack import generate_slack_payload
from .utils import (
    detect_release_branches,
    detect_release_version,
    extract_jira_keys,
    resolve_author,
)


def main() -> int:
    """Main orchestration function."""
    try:
        args = parse_args()
        config = Config.from_env(args)

        gh_client = GitHubClient()
        jira_client = JiraClient(config.jira_user, config.jira_token, config.jira_base_url)

        branch_names = detect_release_branches(config.branches)
        if not branch_names:
            return 1

        branches = []
        for branch_name in branch_names:
            branch = detect_release_version(branch_name)
            branches.append(branch)

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

        jira_issues = {}

        for jira_key in sorted(all_jira_keys):
            issue = jira_client.get_issue(jira_key)
            if issue:
                jira_issues[jira_key] = issue

        orphaned_issues: dict[str, list[str]] = {}

        for branch in branches:
            jql = f'project = {config.jira_project} AND fixVersion = "{branch.expected_version}"'
            jira_issues_for_branch = jira_client.search_issues(jql)

            pr_jira_keys = set()
            for pr in prs_by_branch.get(branch.name, []):
                pr_jira_keys.update(pr.jira_keys)

            orphaned = [issue.key for issue in jira_issues_for_branch if issue.key not in pr_jira_keys]

            if orphaned:
                orphaned_issues[branch.name] = orphaned

        timestamp = datetime.utcnow().strftime("%Y-%m-%d %H:%M:%S UTC")

        pathlib.Path(config.output_dir).mkdir(exist_ok=True, parents=True)

        markdown = generate_markdown(
            branches, prs_by_branch, jira_issues, orphaned_issues, timestamp,
        )

        markdown_path = os.path.join(config.output_dir, config.report_file)
        pathlib.Path(markdown_path).write_text(markdown, encoding="utf-8")

        slack_channel = os.getenv("SLACK_CHANNEL", "C05AZF8T7GW")
        slack_payload = generate_slack_payload(
            branches, prs_by_branch, jira_issues, orphaned_issues,
            timestamp, config.github_run_url, slack_channel,
        )

        slack_path = os.path.join(config.output_dir, config.slack_payload_file)
        with pathlib.Path(slack_path).open("w", encoding="utf-8") as f:
            json.dump(slack_payload, f, indent=2)

        return 0

    except BackportAuditError:
        return 1
    except Exception:
        traceback.print_exc()
        return 2


if __name__ == "__main__":
    sys.exit(main())
