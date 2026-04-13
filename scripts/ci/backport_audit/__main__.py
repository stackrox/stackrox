#!/usr/bin/env python3
"""Main orchestration for backport audit tool."""

import json
import os
import sys
import traceback
from datetime import datetime
from typing import Dict, List

from . import __version__
from .config import Config, parse_args
from .github_client import GitHubClient
from .jira_client import JiraClient
from .models import BackportAuditError, PR
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
    print(f"Backport Audit Tool v{__version__}")

    try:
        args = parse_args()
        config = Config.from_env(args)

        print(f"Configuration: branches={config.branches}, output_dir={config.output_dir}")

        gh_client = GitHubClient()
        jira_client = JiraClient(config.jira_user, config.jira_token, config.jira_base_url)

        print("Detecting release branches...")
        branch_names = detect_release_branches(config.branches)
        if not branch_names:
            print("ERROR: No release branches detected", file=sys.stderr)
            return 1

        branches = []
        for branch_name in branch_names:
            branch = detect_release_version(branch_name)
            branches.append(branch)
            print(f"  {branch.name} → {branch.expected_version} (latest tag: {branch.latest_tag or 'none'})")

        print("Fetching backport PRs from GitHub...")
        all_prs_data = gh_client.fetch_prs("backport", "open")
        print(f"Found {len(all_prs_data)} total backport PRs")

        print("Processing PRs...")
        prs_by_branch: Dict[str, List[PR]] = {}
        all_jira_keys = set()

        for pr_data in all_prs_data:
            base_ref = pr_data['baseRefName']
            if base_ref not in branch_names:
                continue

            author = resolve_author(pr_data, gh_client)

            jira_keys = extract_jira_keys(pr_data['title'])
            all_jira_keys.update(jira_keys)

            pr = PR(
                number=pr_data['number'],
                title=pr_data['title'],
                author=author,
                base_ref=base_ref,
                jira_keys=jira_keys,
                body=pr_data.get('body', '')
            )

            if base_ref not in prs_by_branch:
                prs_by_branch[base_ref] = []
            prs_by_branch[base_ref].append(pr)

        print(f"Processed {sum(len(prs) for prs in prs_by_branch.values())} PRs targeting release branches")

        print(f"Validating {len(all_jira_keys)} unique Jira issues...")
        jira_issues = {}

        for jira_key in sorted(all_jira_keys):
            issue = jira_client.get_issue(jira_key)
            if issue:
                jira_issues[jira_key] = issue

        print(f"Validated {len(jira_issues)} Jira issues")

        print("Finding orphaned Jira issues...")
        orphaned_issues: Dict[str, List[str]] = {}

        for branch in branches:
            jql = f'project = {config.jira_project} AND fixVersion = "{branch.expected_version}"'
            jira_issues_for_branch = jira_client.search_issues(jql)

            pr_jira_keys = set()
            for pr in prs_by_branch.get(branch.name, []):
                pr_jira_keys.update(pr.jira_keys)

            orphaned = []
            for issue in jira_issues_for_branch:
                if issue.key not in pr_jira_keys:
                    orphaned.append(issue.key)

            if orphaned:
                orphaned_issues[branch.name] = orphaned
                print(f"  {branch.name}: {len(orphaned)} orphaned issues")

        print("Generating reports...")
        timestamp = datetime.utcnow().strftime("%Y-%m-%d %H:%M:%S UTC")

        os.makedirs(config.output_dir, exist_ok=True)

        markdown = generate_markdown(
            branches, prs_by_branch, jira_issues, orphaned_issues, timestamp
        )

        markdown_path = os.path.join(config.output_dir, config.report_file)
        with open(markdown_path, 'w') as f:
            f.write(markdown)
        print(f"Markdown report written to: {markdown_path}")

        slack_channel = os.getenv('SLACK_CHANNEL', 'C05AZF8T7GW')
        slack_payload = generate_slack_payload(
            branches, prs_by_branch, jira_issues, orphaned_issues,
            timestamp, config.github_run_url, slack_channel
        )

        slack_path = os.path.join(config.output_dir, config.slack_payload_file)
        with open(slack_path, 'w') as f:
            json.dump(slack_payload, f, indent=2)
        print(f"Slack payload written to: {slack_path}")

        print("✅ Audit complete")
        return 0

    except BackportAuditError as e:
        print(f"ERROR: {e}", file=sys.stderr)
        return 1
    except Exception as e:
        print(f"UNEXPECTED ERROR: {e}", file=sys.stderr)
        traceback.print_exc()
        return 2


if __name__ == "__main__":
    sys.exit(main())
