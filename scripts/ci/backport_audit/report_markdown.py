"""Markdown report generation."""


from .models import PR, JiraIssue, ReleaseBranch
from .slack import get_slack_mention
from .urgency import URGENCY_ORDER, calculate_urgency, format_deadline_info


def _collect_issue_problems(
    prs: list[PR],
    jira_issues: dict[str, JiraIssue],
    expected_version: str,
) -> tuple[list[tuple], dict[str, list[int]]]:
    """Collect issues with missing metadata and track associated PRs.

    Returns:
        Tuple of (issues_with_problems list, jira_to_prs mapping)
    """
    issues_with_problems = []
    jira_to_prs: dict[str, list[int]] = {}

    for pr in prs:
        for jira_key in pr.jira_keys:
            if jira_key not in jira_issues:
                continue

            issue = jira_issues[jira_key]

            if jira_key not in jira_to_prs:
                jira_to_prs[jira_key] = []
            jira_to_prs[jira_key].append(pr.number)

            has_fix = (
                expected_version in issue.fix_versions
                if issue.fix_versions
                else False
            )
            has_affected = len(issue.affected_versions) > 0

            if not has_fix or not has_affected:
                fix_icon = ":white_check_mark:" if has_fix else ":x:"
                affected_icon = ":white_check_mark:" if has_affected else ":x:"

                urgency_level, urgency_icon = calculate_urgency(
                    issue.priority,
                    issue.severity,
                    issue.due_date,
                    issue.sla_date,
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
                    urgency_icon,
                )
                if issue_info not in issues_with_problems:
                    issues_with_problems.append(issue_info)

    return issues_with_problems, jira_to_prs


def _format_issue_line(
    issue_info: tuple,
    jira_to_prs: dict[str, list[int]],
) -> str:
    """Format a single issue line for markdown report."""
    (
        jira_key,
        fix_icon,
        affected_icon,
        assignee,
        team,
        component,
        priority,
        severity,
        deadline_info,
        _urgency_level,
        urgency_icon,
    ) = issue_info

    pr_refs = jira_to_prs.get(jira_key, [])
    pr_links = ", ".join([f"#{pr}" for pr in pr_refs])
    pr_suffix = f" (PRs: {pr_links})" if pr_refs else ""

    priority_info = f"Priority: {priority}"
    if severity:
        priority_info += f", Severity: {severity}"

    return (
        f"- {urgency_icon} {jira_key}: {fix_icon} fixVersion, "
        f"{affected_icon} affectedVersion | "
        f"{priority_info} | {deadline_info} | "
        f"Assignee: {assignee}, Team: {team}, Component: {component}{pr_suffix}"
    )


def generate_markdown(
    branches: list[ReleaseBranch],
    prs_by_branch: dict[str, list[PR]],
    jira_issues: dict[str, JiraIssue],
    orphaned_issues: dict[str, list[str]],
    timestamp: str,
) -> str:
    """Generate markdown report.

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
    lines.extend(("# Backport PR Audit Report", "", f"Generated: {timestamp}", ""))

    # Sort branches by version
    sorted_branches = sorted(
        branches,
        key=lambda b: [int(x) for x in b.expected_version.split(".")],
    )

    for branch in sorted_branches:
        prs = prs_by_branch.get(branch.name, [])
        orphaned = orphaned_issues.get(branch.name, [])

        # Skip empty branches
        if not prs and not orphaned:
            continue

        lines.extend((f"## {branch.name} (Expected: {branch.expected_version})", ""))

        # PRs without Jira reference
        prs_no_jira = [pr for pr in prs if not pr.jira_keys]
        if prs_no_jira:
            lines.extend((f"### PRs Missing Jira Reference ({len(prs_no_jira)})", ""))

            # Sort by author
            prs_no_jira.sort(key=lambda p: p.author)

            for pr in prs_no_jira:
                mention = get_slack_mention(pr.author)
                lines.append(f"- {mention} #{pr.number}: {pr.title}")

            lines.append("")

        issues_with_problems, jira_to_prs = _collect_issue_problems(
            prs,
            jira_issues,
            branch.expected_version,
        )

        if issues_with_problems:
            issues_with_problems.sort(key=lambda x: URGENCY_ORDER.get(x[9], 99))
            count = len(issues_with_problems)
            lines.extend((f"### Jira Issues with Missing Metadata ({count})", ""))

            for issue_info in issues_with_problems:
                lines.append(_format_issue_line(issue_info, jira_to_prs))

            lines.append("")

        # Orphaned Jira issues
        if orphaned:
            lines.extend((f"### Orphaned Jira Issues ({len(orphaned)})", "", f"Issues with fixVersion={branch.expected_version} but no corresponding PR:", ""))

            lines.extend(f"- {jira_key}" for jira_key in sorted(orphaned))

            lines.append("")

    return "\n".join(lines)
