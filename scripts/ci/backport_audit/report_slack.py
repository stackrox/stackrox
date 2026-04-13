"""Slack Block Kit payload generation for backport audit reports."""

from typing import Any, Dict, List, Optional

from .models import PR, JiraIssue, ReleaseBranch
from .slack import get_slack_mention
from .urgency import calculate_urgency, format_deadline_info, URGENCY_ORDER


def generate_slack_payload(
    branches: List[ReleaseBranch],
    prs_by_branch: Dict[str, List[PR]],
    jira_issues: Dict[str, JiraIssue],
    orphaned_issues: Dict[str, List[str]],
    timestamp: str,
    github_run_url: Optional[str],
    slack_channel: str
) -> Dict[str, Any]:
    """
    Generate Slack payload with Block Kit format.

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
    total_prs_no_jira = sum(
        len([pr for pr in prs_by_branch.get(b.name, []) if not pr.jira_keys])
        for b in branches
    )

    total_jira_issues = 0
    urgency_counts = {'overdue': 0, 'critical': 0, 'high': 0, 'normal': 0, 'low': 0}

    for branch in branches:
        prs = prs_by_branch.get(branch.name, [])
        for pr in prs:
            for jira_key in pr.jira_keys:
                if jira_key in jira_issues:
                    issue = jira_issues[jira_key]
                    has_fix = branch.expected_version in issue.fix_versions if issue.fix_versions else False
                    has_affected = len(issue.affected_versions) > 0
                    if not has_fix or not has_affected:
                        total_jira_issues += 1
                        urgency_level, _ = calculate_urgency(
                            issue.priority,
                            issue.severity,
                            issue.due_date,
                            issue.sla_date
                        )
                        urgency_counts[urgency_level] = urgency_counts.get(urgency_level, 0) + 1
                        break

    blocks = []

    blocks.append({
        "type": "header",
        "text": {
            "type": "plain_text",
            "text": "📋 Backport PR Audit Report"
        }
    })

    summary_text = (
        f"*Generated:* {timestamp}\n"
        f"*Total PRs missing Jira:* {total_prs_no_jira}\n"
        f"*Total Jira issues with missing metadata:* {total_jira_issues}\n"
        f"*Urgency breakdown:* 🔴 {urgency_counts['overdue'] + urgency_counts['critical']} critical/overdue, "
        f"🟡 {urgency_counts['high']} high, "
        f"🟢 {urgency_counts['normal']} normal"
    )
    blocks.append({
        "type": "section",
        "text": {
            "type": "mrkdwn",
            "text": summary_text
        }
    })

    if github_run_url:
        blocks.append({
            "type": "section",
            "text": {
                "type": "mrkdwn",
                "text": f"<{github_run_url}|View full report in GitHub Actions>"
            }
        })

    blocks.append({"type": "divider"})

    sorted_branches = sorted(
        branches,
        key=lambda b: [int(x) for x in b.expected_version.split('.')]
    )

    for branch in sorted_branches:
        prs = prs_by_branch.get(branch.name, [])
        orphaned = orphaned_issues.get(branch.name, [])

        if not prs and not orphaned:
            continue

        section_lines = []
        section_lines.append(f"*{branch.name} (Expected: {branch.expected_version})*\n")

        prs_no_jira = [pr for pr in prs if not pr.jira_keys]
        if prs_no_jira:
            section_lines.append(f"\n*PRs Missing Jira Reference ({len(prs_no_jira)})*")
            prs_no_jira.sort(key=lambda p: p.author)

            for pr in prs_no_jira:
                mention = get_slack_mention(pr.author)
                pr_link = f"<https://github.com/stackrox/stackrox/pull/{pr.number}|#{pr.number}>"
                section_lines.append(f"- {mention} {pr_link}: {pr.title}")

        issues_with_problems = []
        jira_to_prs: Dict[str, List[int]] = {}

        for pr in prs:
            for jira_key in pr.jira_keys:
                if jira_key not in jira_issues:
                    continue

                issue = jira_issues[jira_key]
                if jira_key not in jira_to_prs:
                    jira_to_prs[jira_key] = []
                jira_to_prs[jira_key].append(pr.number)

                has_fix = branch.expected_version in issue.fix_versions if issue.fix_versions else False
                has_affected = len(issue.affected_versions) > 0

                if not has_fix or not has_affected:
                    fix_icon = ":white_check_mark:" if has_fix else ":x:"
                    affected_icon = ":white_check_mark:" if has_affected else ":x:"

                    urgency_level, urgency_icon = calculate_urgency(
                        issue.priority,
                        issue.severity,
                        issue.due_date,
                        issue.sla_date
                    )

                    deadline_info = format_deadline_info(issue.due_date, issue.sla_date)

                    issue_info = (
                        jira_key,
                        fix_icon,
                        affected_icon,
                        issue.assignee or "Unassigned",
                        issue.team or "No team",
                        issue.component or "No component",
                        issue.priority or "No priority",
                        deadline_info,
                        urgency_level,
                        urgency_icon
                    )
                    if issue_info not in issues_with_problems:
                        issues_with_problems.append(issue_info)

        if issues_with_problems:
            issues_with_problems.sort(key=lambda x: URGENCY_ORDER.get(x[8], 99))

            section_lines.append(f"\n*Jira Issues with Missing Metadata ({len(issues_with_problems)})*")

            for (jira_key, fix_icon, affected_icon, assignee, team, component,
                 priority, deadline_info, urgency_level, urgency_icon) in issues_with_problems:
                jira_link = f"<https://redhat.atlassian.net/browse/{jira_key}|{jira_key}>"
                pr_refs = jira_to_prs.get(jira_key, [])
                pr_links = ', '.join([
                    f"<https://github.com/stackrox/stackrox/pull/{pr}|#{pr}>"
                    for pr in pr_refs
                ])
                pr_suffix = f" (PRs: {pr_links})" if pr_refs else ""

                section_lines.append(
                    f"• {urgency_icon} {jira_link}: {fix_icon} fixVer, {affected_icon} affectedVer | "
                    f"P: {priority} | {deadline_info}{pr_suffix}"
                )

        if orphaned:
            section_lines.append(f"\n*Orphaned Jira Issues ({len(orphaned)})*")
            section_lines.append(f"Issues with fixVersion={branch.expected_version} but no corresponding PR:")

            for jira_key in sorted(orphaned):
                jira_link = f"<https://redhat.atlassian.net/browse/{jira_key}|{jira_key}>"
                section_lines.append(f"- {jira_link}")

        section_text = '\n'.join(section_lines)
        sections = _split_slack_sections(section_text, branch.name)

        for section in sections:
            blocks.append({
                "type": "section",
                "text": {
                    "type": "mrkdwn",
                    "text": section
                }
            })

    return {
        "channel": slack_channel,
        "blocks": blocks
    }


def _split_slack_sections(text: str, branch_name: str, max_chars: int = 2800) -> List[str]:
    """
    Split text into Slack-compatible sections.

    Args:
        text: Text to split
        branch_name: Branch name for continuation headers
        max_chars: Maximum characters per section

    Returns:
        List of section strings
    """
    lines = text.split('\n')
    sections = []
    current_section = []
    current_length = 0
    current_header = None

    for line in lines:
        line_length = len(line) + 1

        if line.startswith('*') and line.endswith('*') and len(line) > 2:
            current_header = line

        if current_length + line_length > max_chars and current_section:
            sections.append('\n'.join(current_section))

            if current_header:
                current_section = [f"*{branch_name} (continued)*\n", current_header]
            else:
                current_section = [f"*{branch_name} (continued)*"]

            current_length = sum(len(l) + 1 for l in current_section)

        current_section.append(line)
        current_length += line_length

    if current_section:
        sections.append('\n'.join(current_section))

    return sections
