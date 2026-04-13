"""Markdown report generation."""

from typing import Dict, List

from .models import PR, JiraIssue, ReleaseBranch
from .slack import get_slack_mention
from .urgency import calculate_urgency, format_deadline_info, URGENCY_ORDER


def generate_markdown(
    branches: List[ReleaseBranch],
    prs_by_branch: Dict[str, List[PR]],
    jira_issues: Dict[str, JiraIssue],
    orphaned_issues: Dict[str, List[str]],
    timestamp: str
) -> str:
    """
    Generate markdown report.

    Args:
        branches: List of release branches
        prs_by_branch: PRs grouped by branch
        jira_issues: Jira issues by key
        orphaned_issues: Orphaned Jira keys by branch
        timestamp: Report generation timestamp

    Returns:
        Markdown report string
    """
    lines = []
    lines.append("# Backport PR Audit Report")
    lines.append("")
    lines.append(f"Generated: {timestamp}")
    lines.append("")

    # Sort branches by version
    sorted_branches = sorted(
        branches,
        key=lambda b: [int(x) for x in b.expected_version.split('.')]
    )

    for branch in sorted_branches:
        prs = prs_by_branch.get(branch.name, [])
        orphaned = orphaned_issues.get(branch.name, [])

        # Skip empty branches
        if not prs and not orphaned:
            continue

        lines.append(f"## {branch.name} (Expected: {branch.expected_version})")
        lines.append("")

        # PRs without Jira reference
        prs_no_jira = [pr for pr in prs if not pr.jira_keys]
        if prs_no_jira:
            lines.append(f"### PRs Missing Jira Reference ({len(prs_no_jira)})")
            lines.append("")

            # Sort by author
            prs_no_jira.sort(key=lambda p: p.author)

            for pr in prs_no_jira:
                mention = get_slack_mention(pr.author)
                lines.append(f"- {mention} #{pr.number}: {pr.title}")

            lines.append("")

        # Jira issues with missing metadata
        issues_with_problems = []
        jira_to_prs: Dict[str, List[int]] = {}

        for pr in prs:
            for jira_key in pr.jira_keys:
                if jira_key not in jira_issues:
                    continue

                issue = jira_issues[jira_key]

                # Track PRs for this Jira
                if jira_key not in jira_to_prs:
                    jira_to_prs[jira_key] = []
                jira_to_prs[jira_key].append(pr.number)

                # Check for missing metadata
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
                        issue.severity,
                        deadline_info,
                        urgency_level,
                        urgency_icon
                    )
                    if issue_info not in issues_with_problems:
                        issues_with_problems.append(issue_info)

        if issues_with_problems:
            issues_with_problems.sort(key=lambda x: URGENCY_ORDER.get(x[9], 99))

            lines.append(f"### Jira Issues with Missing Metadata ({len(issues_with_problems)})")
            lines.append("")

            for (jira_key, fix_icon, affected_icon, assignee, team, component,
                 priority, severity, deadline_info, urgency_level, urgency_icon) in issues_with_problems:
                pr_refs = jira_to_prs.get(jira_key, [])
                pr_links = ', '.join([f"#{pr}" for pr in pr_refs])
                pr_suffix = f" (PRs: {pr_links})" if pr_refs else ""

                priority_info = f"Priority: {priority}"
                if severity:
                    priority_info += f", Severity: {severity}"

                lines.append(
                    f"- {urgency_icon} {jira_key}: {fix_icon} fixVersion, {affected_icon} affectedVersion | "
                    f"{priority_info} | {deadline_info} | "
                    f"Assignee: {assignee}, Team: {team}, Component: {component}{pr_suffix}"
                )

            lines.append("")

        # Orphaned Jira issues
        if orphaned:
            lines.append(f"### Orphaned Jira Issues ({len(orphaned)})")
            lines.append("")
            lines.append(f"Issues with fixVersion={branch.expected_version} but no corresponding PR:")
            lines.append("")

            for jira_key in sorted(orphaned):
                lines.append(f"- {jira_key}")

            lines.append("")

    return '\n'.join(lines)
