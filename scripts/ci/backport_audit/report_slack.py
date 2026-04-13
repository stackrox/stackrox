"""Slack Block Kit payload generation for backport audit reports."""

from dataclasses import dataclass
from typing import Any

from .models import PR, JiraIssue, ReleaseBranch
from .report_markdown import _collect_issue_problems
from .slack import get_slack_mention
from .urgency import URGENCY_ORDER, calculate_urgency, format_deadline_info

# Minimum length for a valid markdown header (e.g., "*X*" has len > 2)
MIN_HEADER_LENGTH = 2


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


def _format_slack_issue_line(
    issue_info: tuple,
    jira_to_prs: dict[str, list[int]],
) -> str:
    """Format a single issue line for Slack report."""
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

    jira_link = f"<https://redhat.atlassian.net/browse/{jira_key}|{jira_key}>"
    pr_refs = jira_to_prs.get(jira_key, [])
    pr_links = ", ".join(
        [f"<https://github.com/stackrox/stackrox/pull/{pr}|#{pr}>" for pr in pr_refs]
    )
    pr_suffix = f" (PRs: {pr_links})" if pr_refs else ""

    # Format priority/severity as Slack emojis
    priority_parts = []
    if priority and priority != "No priority":
        priority_parts.append(f":jira-{priority.lower()}:")
    elif priority == "No priority":
        priority_parts.append("No priority")

    if severity:
        priority_parts.append(f":cve-{severity.lower()}:")

    priority_info = " ".join(priority_parts) if priority_parts else "No priority"

    return (
        f"• {urgency_icon} {jira_link}: {fix_icon} fixVer, "
        f"{affected_icon} affectedVer | "
        f"{priority_info} | {deadline_info}{pr_suffix}"
    )


def _generate_branch_section(
    branch: ReleaseBranch,
    prs: list[PR],
    orphaned: list[str],
    jira_issues: dict[str, JiraIssue],
) -> list[str]:
    """Generate section lines for a single branch."""
    section_lines = []
    section_lines.append(f"*{branch.name} (Expected: {branch.expected_version})*\n")

    prs_no_jira = [pr for pr in prs if not pr.jira_keys]
    if prs_no_jira:
        section_lines.append(f"\n*PRs Missing Jira Reference ({len(prs_no_jira)})*")
        prs_no_jira.sort(key=lambda p: p.author)

        for pr in prs_no_jira:
            mention = get_slack_mention(pr.author)
            pr_link = (
                f"<https://github.com/stackrox/stackrox/pull/{pr.number}|#{pr.number}>"
            )
            section_lines.append(f"- {mention} {pr_link}: {pr.title}")

    issues_with_problems, jira_to_prs = _collect_issue_problems(
        prs,
        jira_issues,
        branch.expected_version,
    )

    if issues_with_problems:
        issues_with_problems.sort(key=lambda x: URGENCY_ORDER.get(x[9], 99))
        count = len(issues_with_problems)
        section_lines.append(f"\n*Jira Issues with Missing Metadata ({count})*")

        for issue_info in issues_with_problems:
            section_lines.append(_format_slack_issue_line(issue_info, jira_to_prs))

    if orphaned:
        section_lines.extend(
            (
                f"\n*Orphaned Jira Issues ({len(orphaned)})*",
                f"Issues with fixVersion={branch.expected_version} but no "
                "corresponding PR:",
            )
        )

        for jira_key in sorted(orphaned):
            jira_link = f"<https://redhat.atlassian.net/browse/{jira_key}|{jira_key}>"
            section_lines.append(f"- {jira_link}")

    return section_lines


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
                    f"*Urgency breakdown:* 🔴 {urgency_counts['overdue'] + urgency_counts['critical']} "
                    f"critical/overdue, "
                    f"🟡 {urgency_counts['high']} high, "
                    f"🟢 {urgency_counts['normal']} normal"
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

        section_lines = _generate_branch_section(branch, prs, orphaned, jira_issues)
        section_text = "\n".join(section_lines)
        sections = _split_slack_sections(section_text, branch.name)

        blocks.extend(
            {
                "type": "section",
                "text": {
                    "type": "mrkdwn",
                    "text": section,
                },
            }
            for section in sections
        )

    return {
        "channel": slack_channel,
        "blocks": blocks,
    }


def _split_slack_sections(text: str, branch_name: str, max_chars: int = 2800) -> list[str]:
    """Split text into Slack-compatible sections.

    Args:
        text: Text to split
        branch_name: Branch name for continuation headers
        max_chars: Maximum characters per section

    Returns:
        List of section strings

    """
    lines = text.split("\n")
    sections = []
    current_section = []
    current_length = 0
    current_header = None

    for line in lines:
        line_length = len(line) + 1

        if line.startswith("*") and line.endswith("*") and len(line) > MIN_HEADER_LENGTH:
            current_header = line

        if current_length + line_length > max_chars and current_section:
            sections.append("\n".join(current_section))

            if current_header:
                current_section = [f"*{branch_name} (continued)*\n", current_header]
            else:
                current_section = [f"*{branch_name} (continued)*"]

            current_length = sum(len(section_line) + 1 for section_line in current_section)

        current_section.append(line)
        current_length += line_length

    if current_section:
        sections.append("\n".join(current_section))

    return sections
