"""Slack Block Kit payload generation for backport audit reports."""

from dataclasses import dataclass
from typing import Any

from .models import PR, JiraIssue, ReleaseBranch
from .report_markdown import _collect_issue_problems
from .slack import get_slack_mention
from .urgency import URGENCY_ORDER, calculate_urgency, format_deadline_info


@dataclass
class ReportData:
    """Container for report data to reduce parameter count."""

    branches: list[ReleaseBranch]
    prs_by_branch: dict[str, list[PR]]
    jira_issues: dict[str, JiraIssue]
    orphaned_issues: dict[str, list[str]]
    timestamp: str
    github_run_url: str | None
    slack_channel: str


def _calculate_urgency_stats(
    data: ReportData,
) -> tuple[int, int, dict[str, int]]:
    """Calculate total PRs without Jira and urgency statistics.

    Returns:
        Tuple of (total_prs_no_jira, total_jira_issues, urgency_counts)
    """
    total_prs_no_jira = sum(
        len([pr for pr in data.prs_by_branch.get(b.name, []) if not pr.jira_keys])
        for b in data.branches
    )

    total_jira_issues = 0
    urgency_counts = {"overdue": 0, "critical": 0, "high": 0, "normal": 0, "low": 0}

    for branch in data.branches:
        prs = data.prs_by_branch.get(branch.name, [])
        for pr in prs:
            for jira_key in pr.jira_keys:
                if jira_key in data.jira_issues:
                    issue = data.jira_issues[jira_key]
                    has_fix = (
                        branch.expected_version in issue.fix_versions
                        if issue.fix_versions
                        else False
                    )
                    has_affected = len(issue.affected_versions) > 0
                    if not has_fix or not has_affected:
                        total_jira_issues += 1
                        urgency_level, _ = calculate_urgency(
                            issue.priority,
                            issue.severity,
                            issue.due_date,
                            issue.sla_date,
                        )
                        urgency_counts[urgency_level] = (
                            urgency_counts.get(urgency_level, 0) + 1
                        )
                        break

    return total_prs_no_jira, total_jira_issues, urgency_counts


def _create_table_cell_text(text: str) -> dict[str, Any]:
    """Create a simple text cell for table."""
    return {"type": "raw_text", "text": text}


def _create_table_cell_link(url: str, text: str) -> dict[str, Any]:
    """Create a rich text link cell for table."""
    return {
        "type": "rich_text",
        "elements": [
            {
                "type": "rich_text_section",
                "elements": [{"type": "link", "url": url, "text": text}],
            }
        ],
    }


def _create_issue_table_row(
    issue_info: tuple,
    jira_to_prs: dict[str, list[int]],
) -> list[dict[str, Any]]:
    """Create table row for a single issue."""
    (
        jira_key,
        fix_icon,
        affected_icon,
        _assignee,
        _team,
        _component,
        priority,
        severity,
        deadline_info,
        _urgency_level,
        urgency_icon,
    ) = issue_info

    pr_refs = jira_to_prs.get(jira_key, [])
    if pr_refs:
        pr_links = ", ".join([f"#{pr}" for pr in pr_refs])
    else:
        pr_links = "—"

    # Format priority as Slack emoji
    if priority and priority != "No priority":
        priority_display = f":jira-{priority.lower()}:"
    else:
        priority_display = ":jira-undefined:"

    severity_display = severity if severity else "—"

    return [
        _create_table_cell_text(urgency_icon),
        _create_table_cell_link(
            f"https://redhat.atlassian.net/browse/{jira_key}", jira_key
        ),
        _create_table_cell_text(fix_icon),
        _create_table_cell_text(affected_icon),
        _create_table_cell_text(priority_display),
        _create_table_cell_text(severity_display),
        _create_table_cell_text(deadline_info),
        _create_table_cell_text(pr_links),
    ]


def _generate_branch_blocks(
    branch: ReleaseBranch,
    prs: list[PR],
    orphaned: list[str],
    jira_issues: dict[str, JiraIssue],
) -> list[dict[str, Any]]:
    """Generate Slack blocks for a single branch."""
    blocks = []

    # Branch header
    blocks.append(
        {
            "type": "section",
            "text": {
                "type": "mrkdwn",
                "text": f"*{branch.name} (Expected: {branch.expected_version})*",
            },
        }
    )

    # PRs without Jira reference
    prs_no_jira = [pr for pr in prs if not pr.jira_keys]
    if prs_no_jira:
        prs_no_jira.sort(key=lambda p: p.author)
        pr_lines = [f"*PRs Missing Jira Reference ({len(prs_no_jira)})*\n"]

        for pr in prs_no_jira:
            mention = get_slack_mention(pr.author)
            pr_link = (
                f"<https://github.com/stackrox/stackrox/pull/{pr.number}|#{pr.number}>"
            )
            pr_lines.append(f"• {mention} {pr_link}: {pr.title}")

        blocks.append(
            {
                "type": "section",
                "text": {"type": "mrkdwn", "text": "\n".join(pr_lines)},
            }
        )

    # Jira issues with missing metadata (as table)
    issues_with_problems, jira_to_prs = _collect_issue_problems(
        prs,
        jira_issues,
        branch.expected_version,
    )

    if issues_with_problems:
        issues_with_problems.sort(key=lambda x: URGENCY_ORDER.get(x[9], 99))
        count = len(issues_with_problems)

        # Section header with legend
        blocks.append(
            {
                "type": "section",
                "text": {
                    "type": "mrkdwn",
                    "text": (
                        f"*Jira Issues with Missing Metadata ({count})*\n"
                        "_Legend: :red_circle: overdue/critical | :large_yellow_circle: high | "
                        ":large_green_circle: normal | :white_check_mark: present | :x: missing_"
                    ),
                },
            }
        )

        # Table with issues
        table_rows = [
            # Header row
            [
                _create_table_cell_text("U"),
                _create_table_cell_text("Issue"),
                _create_table_cell_text("Fix"),
                _create_table_cell_text("Aff"),
                _create_table_cell_text("Priority"),
                _create_table_cell_text("Severity"),
                _create_table_cell_text("Deadline"),
                _create_table_cell_text("PRs"),
            ]
        ]

        # Data rows
        for issue_info in issues_with_problems:
            table_rows.append(_create_issue_table_row(issue_info, jira_to_prs))

        blocks.append({"type": "table", "rows": table_rows})

    # Orphaned Jira issues
    if orphaned:
        orphan_lines = [
            f"*Orphaned Jira Issues ({len(orphaned)})*",
            f"Issues with fixVersion={branch.expected_version} but no corresponding PR:\n",
        ]

        for jira_key in sorted(orphaned):
            jira_link = f"<https://redhat.atlassian.net/browse/{jira_key}|{jira_key}>"
            orphan_lines.append(f"• {jira_link}")

        blocks.append(
            {
                "type": "section",
                "text": {"type": "mrkdwn", "text": "\n".join(orphan_lines)},
            }
        )

    return blocks


def generate_slack_payload(
    branches: list[ReleaseBranch],
    prs_by_branch: dict[str, list[PR]],
    jira_issues: dict[str, JiraIssue],
    orphaned_issues: dict[str, list[str]],
    timestamp: str,
    github_run_url: str | None,
    slack_channel: str,
) -> dict[str, Any]:
    """Generate Slack payload with Block Kit format.

    Args:
        branches: List of release branches
        prs_by_branch: PRs grouped by branch
        jira_issues: Jira issues by key
        orphaned_issues: Orphaned Jira keys by branch
        timestamp: Report generation timestamp
        github_run_url: GitHub Actions run URL
        slack_channel: Slack channel ID

    Returns:
        Slack payload dictionary

    """
    data = ReportData(
        branches=branches,
        prs_by_branch=prs_by_branch,
        jira_issues=jira_issues,
        orphaned_issues=orphaned_issues,
        timestamp=timestamp,
        github_run_url=github_run_url,
        slack_channel=slack_channel,
    )

    total_prs_no_jira, total_jira_issues, urgency_counts = _calculate_urgency_stats(
        data
    )

    blocks = [
        {
            "type": "header",
            "text": {
                "type": "plain_text",
                "text": "📋 Backport PR Audit Report",
            },
        },
        {
            "type": "section",
            "text": {
                "type": "mrkdwn",
                "text": (
                    f"*Generated:* {timestamp}\n"
                    f"*Total PRs missing Jira:* {total_prs_no_jira}\n"
                    f"*Total Jira issues with missing metadata:* {total_jira_issues}\n"
                    f"*Urgency breakdown:* :red_circle: {urgency_counts['overdue'] + urgency_counts['critical']} "
                    f"critical/overdue, "
                    f":large_yellow_circle: {urgency_counts['high']} high, "
                    f":large_green_circle: {urgency_counts['normal']} normal"
                ),
            },
        },
    ]

    if github_run_url:
        blocks.append(
            {
                "type": "section",
                "text": {
                    "type": "mrkdwn",
                    "text": f"<{github_run_url}|View full report in GitHub Actions>",
                },
            }
        )

    blocks.append({"type": "divider"})

    sorted_branches = sorted(
        branches,
        key=lambda b: [int(x) for x in b.expected_version.split(".")],
    )

    for branch in sorted_branches:
        prs = prs_by_branch.get(branch.name, [])
        orphaned = orphaned_issues.get(branch.name, [])

        if not prs and not orphaned:
            continue

        branch_blocks = _generate_branch_blocks(branch, prs, orphaned, jira_issues)
        blocks.extend(branch_blocks)

    return {
        "channel": slack_channel,
        "blocks": blocks,
    }


