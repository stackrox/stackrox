"""Urgency calculation utilities for backport audit."""

from datetime import datetime, timedelta
from typing import Optional, Tuple


PRIORITY_URGENCY = {
    'Critical': 'critical',
    'Blocker': 'critical',
    'Major': 'high',
    'Normal': 'normal',
    'Minor': 'low',
    'Trivial': 'low'
}

CVE_TIMEFRAMES = {
    'Critical': 7,
    'Important': 28,
    'Moderate': 57,
    'Low': None
}


def parse_date(date_str: Optional[str]) -> Optional[datetime]:
    """
    Parse ISO 8601 date string to datetime.

    Args:
        date_str: ISO date string (YYYY-MM-DD) or None

    Returns:
        datetime object or None
    """
    if not date_str:
        return None
    try:
        return datetime.strptime(date_str, '%Y-%m-%d')
    except ValueError:
        return None


def calculate_urgency(
    priority: Optional[str],
    severity: Optional[str],
    due_date: Optional[str],
    sla_date: Optional[str],
    current_date: Optional[datetime] = None
) -> Tuple[str, str]:
    """
    Calculate urgency level and indicator for a Jira issue.

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
        current_date = datetime.utcnow()

    deadline = parse_date(sla_date) or parse_date(due_date)
    if deadline:
        if deadline < current_date:
            return ('overdue', '🔴')

        days_remaining = (deadline - current_date).days
        if days_remaining <= 3:
            return ('critical', '🔴')
        elif days_remaining <= 7:
            return ('high', '🟡')

    if priority:
        urgency = PRIORITY_URGENCY.get(priority, 'normal')
        if urgency == 'critical':
            return ('critical', '🔴')
        elif urgency == 'high':
            return ('high', '🟡')

    if severity in ['Critical', 'Important']:
        return ('high', '🟡')

    return ('normal', '🟢')


def format_deadline_info(
    due_date: Optional[str],
    sla_date: Optional[str],
    current_date: Optional[datetime] = None
) -> str:
    """
    Format deadline information for display.

    Args:
        due_date: Due date string (YYYY-MM-DD)
        sla_date: SLA date string (YYYY-MM-DD)
        current_date: Current date for testing

    Returns:
        Formatted string like "Due: 2026-04-20 (7 days)" or "No deadline"
    """
    if current_date is None:
        current_date = datetime.utcnow()

    deadline = parse_date(sla_date) or parse_date(due_date)
    if not deadline:
        return "No deadline"

    days_remaining = (deadline - current_date).days

    if days_remaining < 0:
        return f"Due: {deadline.strftime('%Y-%m-%d')} (OVERDUE by {abs(days_remaining)} days)"
    elif days_remaining == 0:
        return f"Due: {deadline.strftime('%Y-%m-%d')} (TODAY)"
    else:
        return f"Due: {deadline.strftime('%Y-%m-%d')} ({days_remaining} days)"
