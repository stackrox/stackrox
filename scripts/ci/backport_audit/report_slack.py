"""Slack Block Kit payload generation for backport audit reports."""

from dataclasses import dataclass
from typing import Any

from .models import PR, JiraIssue, ReleaseBranch
from .slack import get_slack_mention
from .urgency import URGENCY_ORDER, calculate_urgency


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


def _create_table_cell_text(text: str, align: str | None = None, is_wrapped: bool = True) -> dict[str, Any]:
    """Create a simple text cell for table."""
    cell = {"type": "raw_text", "text": text, "is_wrapped": is_wrapped}
    if align:
        cell["align"] = align
    return cell


def _create_table_cell_emoji(emoji_name: str, align: str | None = None, is_wrapped: bool = False) -> dict[str, Any]:
    """Create a rich text cell with emoji element."""
    cell = {
        "type": "rich_text",
        "is_wrapped": is_wrapped,
        "elements": [
            {
                "type": "rich_text_section",
                "elements": [{"type": "emoji", "name": emoji_name}],
            }
        ],
    }
    if align:
        cell["align"] = align
    return cell


def _create_table_cell_mention(mention: str, align: str | None = None, is_wrapped: bool = True) -> dict[str, Any]:
    """Create a rich text cell for Slack mentions/emojis.

    Handles:
    - <@U123> user mentions
    - :konflux: emoji
    - @username plain text
    """
    elements = []

    # Check if it's a Slack user mention
    if mention.startswith("<@") and mention.endswith(">"):
        user_id = mention[2:-1]  # Extract U123 from <@U123>
        elements.append({"type": "user", "user_id": user_id})
    # Check if it's an emoji
    elif mention.startswith(":") and mention.endswith(":"):
        emoji_name = mention[1:-1]  # Extract konflux from :konflux:
        elements.append({"type": "emoji", "name": emoji_name})
    # Plain text mention
    else:
        elements.append({"type": "text", "text": mention})

    cell = {
        "type": "rich_text",
        "is_wrapped": is_wrapped,
        "elements": [{"type": "rich_text_section", "elements": elements}],
    }
    if align:
        cell["align"] = align
    return cell


def _create_table_cell_link(url: str, text: str, align: str | None = None, is_wrapped: bool = True) -> dict[str, Any]:
    """Create a rich text link cell for table."""
    cell = {
        "type": "rich_text",
        "is_wrapped": is_wrapped,
        "elements": [
            {
                "type": "rich_text_section",
                "elements": [{"type": "link", "url": url, "text": text}],
            }
        ],
    }
    if align:
        cell["align"] = align
    return cell


def _create_all_pr_rows(
    prs: list[PR],
    jira_issues: dict[str, JiraIssue],
    expected_version: str,
) -> list[list[dict[str, Any]]]:
    """Create table rows for all PRs, including those with and without issues."""
    all_rows = []
    jira_to_prs: dict[str, list[PR]] = {}
    pr_by_number: dict[int, PR] = {pr.number: pr for pr in prs}

    # Track which PRs we've already added
    processed_prs = set()

    # First, add all PRs with Jira issues
    for pr in prs:
        for jira_key in pr.jira_keys:
            if jira_key not in jira_to_prs:
                jira_to_prs[jira_key] = []
            jira_to_prs[jira_key].append(pr)

    # Collect all issues (both complete and with problems)
    all_issues = []
    for pr in prs:
        for jira_key in pr.jira_keys:
            if jira_key not in jira_issues:
                continue

            issue = jira_issues[jira_key]
            has_fix = expected_version in issue.fix_versions if issue.fix_versions else False
            has_affected = len(issue.affected_versions) > 0

            urgency_level, urgency_icon = calculate_urgency(
                issue.priority,
                issue.severity,
                issue.due_date,
                issue.sla_date,
            )

            # Get PR title from first associated PR
            pr_title = jira_to_prs.get(jira_key, [None])[0].title if jira_to_prs.get(jira_key) else "—"

            issue_info = (
                jira_key,
                ":white_check_mark:" if has_fix else ":x:",
                ":white_check_mark:" if has_affected else ":x:",
                issue.priority or "No priority",
                issue.severity,
                pr_title,
                urgency_level,
                urgency_icon,
                has_fix and has_affected,  # is_complete
            )
            if issue_info not in all_issues:
                all_issues.append(issue_info)

    # Sort: problems first (incomplete), then by urgency
    all_issues.sort(key=lambda x: (x[8], URGENCY_ORDER.get(x[6], 99)))

    # Create rows for all issues
    for issue_info in all_issues:
        (
            jira_key,
            fix_icon,
            affected_icon,
            priority,
            severity,
            pr_title,
            urgency_level,
            urgency_icon,
            is_complete,
        ) = issue_info

        pr_refs = jira_to_prs.get(jira_key, [])
        if pr_refs:
            pr_elements = []

            for i, pr_obj in enumerate(pr_refs):
                if i > 0:
                    pr_elements.append({"type": "text", "text": ", "})
                pr_elements.append({
                    "type": "link",
                    "url": f"https://github.com/stackrox/stackrox/pull/{pr_obj.number}",
                    "text": f"#{pr_obj.number}",
                })
                processed_prs.add(pr_obj.number)

            pr_cell = {
                "type": "rich_text",
                "align": "right",
                "is_wrapped": True,
                "elements": [{"type": "rich_text_section", "elements": pr_elements}],
            }

            # Use assignee if complete and assigned, otherwise notify PR authors
            issue = jira_issues.get(jira_key)
            has_assignee = issue and issue.assignee

            if is_complete and has_assignee:
                # Everything is correct and assigned - just show assignee
                author_cell = _create_table_cell_text(issue.assignee)
            else:
                # Notify PR authors about problems (missing metadata or unassigned)
                author_elements = []
                unique_authors = []

                for pr_obj in pr_refs:
                    author_mention = get_slack_mention(pr_obj.author)
                    if author_mention not in unique_authors:
                        unique_authors.append(author_mention)

                # Build author elements with mentions
                for i, author_mention in enumerate(unique_authors):
                    if i > 0:
                        author_elements.append({"type": "text", "text": ", "})

                    # Parse author mention to create appropriate element
                    if author_mention.startswith("<@") and author_mention.endswith(">"):
                        user_id = author_mention[2:-1]
                        author_elements.append({"type": "user", "user_id": user_id})
                    elif author_mention.startswith(":") and author_mention.endswith(":"):
                        emoji_name = author_mention[1:-1]
                        author_elements.append({"type": "emoji", "name": emoji_name})
                    else:
                        author_elements.append({"type": "text", "text": author_mention})

                # Add note if issue is unassigned
                if is_complete and not has_assignee:
                    author_elements.append({"type": "text", "text": " (issue unassigned)"})

                author_cell = {
                    "type": "rich_text",
                    "is_wrapped": True,
                    "elements": [{"type": "rich_text_section", "elements": author_elements}],
                }
        else:
            pr_cell = _create_table_cell_text("—", align="right", is_wrapped=False)
            author_cell = _create_table_cell_text("—", is_wrapped=False)

        urgency_emoji = urgency_icon.strip(":")
        fix_emoji = fix_icon.strip(":")
        affected_emoji = affected_icon.strip(":")
        priority_display = f":jira-{priority.lower()}:" if priority != "No priority" else ":jira-undefined:"
        priority_emoji = priority_display.strip(":")
        severity_display = severity if severity else "—"

        all_rows.append([
            _create_table_cell_emoji(urgency_emoji, align="center"),
            _create_table_cell_link(f"https://redhat.atlassian.net/browse/{jira_key}", jira_key),
            pr_cell,
            _create_table_cell_text(pr_title, align="left"),
            author_cell,  # Author from associated PRs
            _create_table_cell_emoji(fix_emoji, align="center"),
            _create_table_cell_emoji(affected_emoji, align="center"),
            _create_table_cell_emoji(priority_emoji, align="center"),
            _create_table_cell_text(severity_display),
        ])

    # Add PRs without Jira reference at the TOP (prepend)
    prs_no_jira = [pr for pr in prs if not pr.jira_keys and pr.number not in processed_prs]
    no_jira_rows = []
    for pr in prs_no_jira:
        pr_cell = {
            "type": "rich_text",
            "align": "right",
            "is_wrapped": True,
            "elements": [{
                "type": "rich_text_section",
                "elements": [{
                    "type": "link",
                    "url": f"https://github.com/stackrox/stackrox/pull/{pr.number}",
                    "text": f"#{pr.number}",
                }],
            }],
        }

        author_mention = get_slack_mention(pr.author)

        no_jira_rows.append([
            _create_table_cell_text("—", align="center", is_wrapped=False),  # Urgency
            _create_table_cell_emoji("x", align="center"),  # Issue (missing)
            pr_cell,  # PRs
            _create_table_cell_text(pr.title, align="left"),  # PR Title
            _create_table_cell_mention(author_mention),  # Author
            _create_table_cell_emoji("x", align="center"),  # fixVersion (missing)
            _create_table_cell_emoji("x", align="center"),  # affectedVersion (missing)
            _create_table_cell_emoji("jira-undefined", align="center"),  # Priority
            _create_table_cell_text("—", is_wrapped=False),  # Severity
        ])

    # Prepend No Jira PRs to put them at the top
    return no_jira_rows + all_rows


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

    # Comprehensive table with all PRs and issues
    if prs:
        all_rows = _create_all_pr_rows(prs, jira_issues, branch.expected_version)

        # Section header
        blocks.append(
            {
                "type": "section",
                "text": {
                    "type": "mrkdwn",
                    "text": f"*Release Contents ({len(all_rows)} items)*",
                },
            }
        )

        # Table with all issues and PRs
        table_rows = [
            # Header row
            [
                _create_table_cell_text("Urgency", align="center"),
                _create_table_cell_text("Issue"),
                _create_table_cell_text("PRs", align="right"),
                _create_table_cell_text("PR Title", align="left"),
                _create_table_cell_text("Author"),
                _create_table_cell_text("fixVersion", align="center"),
                _create_table_cell_text("affectedVersion", align="center"),
                _create_table_cell_text("Priority", align="center"),
                _create_table_cell_text("Severity"),
            ]
        ]

        # Add all data rows
        table_rows.extend(all_rows)

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
        {
            "type": "section",
            "text": {
                "type": "mrkdwn",
                "text": (
                    "<!subteam^S07D1FQCU9M>\n"
                    "*Action Required:* If you're mentioned in the table below:\n"
                    "• *:x: in Issue column*: Add a Jira reference to your PR description\n"
                    "• *:x: in fixVersion column*: Add the target release version to the Jira issue's fixVersion field\n"
                    "• *:x: in affectedVersion column*: Add affected versions to the Jira issue\n"
                    "• *(issue unassigned)*: Assign the Jira issue to the appropriate owner\n\n"
                    "See: <https://redhat.atlassian.net/wiki/spaces/StackRox/pages/309338452/Patch+Release+Process"
                    "|Patch Release Process Documentation>"
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

    # Add legend at the bottom as a context block
    blocks.append(
        {
            "type": "context",
            "elements": [
                {
                    "type": "mrkdwn",
                    "text": (
                        "*Legend:* Urgency: :red_circle: overdue/critical | "
                        ":large_yellow_circle: high | :large_green_circle: normal • "
                        "Versions: :white_check_mark: present | :x: missing"
                    ),
                }
            ],
        }
    )

    return {
        "channel": slack_channel,
        "blocks": blocks,
    }
