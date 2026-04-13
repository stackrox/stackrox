"""Urgency calculation utilities for backport audit.

See: https://redhat.atlassian.net/wiki/spaces/StackRox/pages/309338452/Patch+Release+Process
"""

from datetime import datetime, timezone

# Bug Ticket Priority Guidelines from Patch Release Process:
# - Critical: "Candidate for immediate Z-release"
# - Major: "Next Z-stream release"
# - Normal: "Unlikely to go into a Z-stream release"
PRIORITY_URGENCY = {
    "Critical": "critical",
    "Blocker": "critical",
    "Major": "high",
    "Normal": "normal",
    "Minor": "low",
    "Trivial": "low",
}

# Default delivery timeframes (2026 Q1) from ProdSec:
# - Critical: "7 business days"
# - Important: "28 calendar days"
# - Moderate: "57 calendar days"
# See: https://redhat.atlassian.net/wiki/spaces/StackRox/pages/309338452/Patch+Release+Process
CVE_TIMEFRAMES = {
    "Critical": 7,
    "Important": 28,
    "Moderate": 57,
    "Low": None,
}

URGENCY_ORDER = {
    "overdue": 0,
    "critical": 1,
    "high": 2,
    "normal": 3,
    "low": 4,
}

# Deadline urgency thresholds (in days)
# Based on typical escalation patterns for critical issues
# requiring immediate attention vs. upcoming deadlines
CRITICAL_DEADLINE_DAYS = 3  # ≤3 days: critical urgency (🔴)
HIGH_DEADLINE_DAYS = 7  # ≤7 days: high urgency (🟡)


def parse_date(date_str: str | None) -> datetime | None:
    """Parse ISO 8601 date string to timezone-aware datetime.

    Args:
        date_str: ISO date string (YYYY-MM-DD) or None

    Returns:
        Timezone-aware datetime object (UTC) or None

    """
    if not date_str:
        return None
    try:
        naive_dt = datetime.strptime(date_str, "%Y-%m-%d")
        return naive_dt.replace(tzinfo=timezone.utc)
    except ValueError:
        return None


def calculate_urgency(
    priority: str | None,
    severity: str | None,
    due_date: str | None,
    sla_date: str | None,
    current_date: datetime | None = None,
) -> tuple[str, str]:
    """Calculate urgency level and indicator for a Jira issue.

    Urgency is determined by deadlines and issue metadata per Patch Release Process:
    - "Jira trackers with Due date or SLA Date — whatever is sooner"
    - CVE severity affects urgency: Critical/Important CVEs require faster resolution
    - Bug priority determines delivery target (Critical→immediate, Major→next Z-stream)

    Args:
        priority: Jira priority (Critical, Major, Normal, etc.)
        severity: CVE severity (Critical, Important, Moderate, Low)
        due_date: Due date string (YYYY-MM-DD)
        sla_date: SLA date string (YYYY-MM-DD)
        current_date: Current date for testing (defaults to now)

    Returns:
        Tuple of (urgency_level, icon)
        - urgency_level: 'overdue', 'critical', 'high', 'normal', 'low'
        - icon: Visual indicator (🔴, 🟡, 🟢, ⚪)

    """
    if current_date is None:
        current_date = datetime.now(tz=timezone.utc)

    # Deadline priority: SLA Date (Red Hat legally binding) > Due date (internal)
    # Per Patch Release Process: "SLA Date informs about the legally binding deadline
    # for Red Hat; usually is Due date + some buffer"
    deadline = parse_date(sla_date) or parse_date(due_date)
    if deadline:
        if deadline < current_date:
            return ("overdue", ":red_circle:")

        days_remaining = (deadline - current_date).days
        if days_remaining <= CRITICAL_DEADLINE_DAYS:
            return ("critical", ":red_circle:")
        if days_remaining <= HIGH_DEADLINE_DAYS:
            return ("high", ":large_yellow_circle:")

    if priority:
        urgency = PRIORITY_URGENCY.get(priority, "normal")
        if urgency == "critical":
            return ("critical", ":red_circle:")
        if urgency == "high":
            return ("high", ":large_yellow_circle:")

    if severity in {"Critical", "Important"}:
        return ("high", ":large_yellow_circle:")

    return ("normal", ":large_green_circle:")


def format_deadline_info(
    due_date: str | None,
    sla_date: str | None,
    current_date: datetime | None = None,
) -> str:
    """Format deadline information for display.

    Args:
        due_date: Due date string (YYYY-MM-DD)
        sla_date: SLA date string (YYYY-MM-DD)
        current_date: Current date for testing

    Returns:
        Formatted string like "Due: 2026-04-20 (7 days)" or "No deadline"

    """
    if current_date is None:
        current_date = datetime.now(tz=timezone.utc)

    deadline = parse_date(sla_date) or parse_date(due_date)
    if not deadline:
        return "No deadline"

    days_remaining = (deadline - current_date).days

    if days_remaining < 0:
        return f"Due: {deadline.strftime('%Y-%m-%d')} (OVERDUE by {abs(days_remaining)} days)"
    if days_remaining == 0:
        return f"Due: {deadline.strftime('%Y-%m-%d')} (TODAY)"
    return f"Due: {deadline.strftime('%Y-%m-%d')} ({days_remaining} days)"
