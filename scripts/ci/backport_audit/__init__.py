"""Backport PR audit tool for StackRox release management."""

__version__ = "1.0.0"

from .config import Config, parse_args
from .github_client import GitHubClient
from .jira_client import JiraClient
from .models import (
    PR,
    BackportAuditError,
    GitHubError,
    JiraError,
    JiraIssue,
    ReleaseBranch,
)
from .report_markdown import generate_markdown
from .report_slack import generate_slack_payload
from .slack import SLACK_USER_MAP, get_slack_mention
from .urgency import (
    CVE_TIMEFRAMES,
    PRIORITY_URGENCY,
    URGENCY_ORDER,
    calculate_urgency,
    format_deadline_info,
    parse_date,
)
from .utils import (
    detect_release_branches,
    detect_release_version,
    extract_jira_keys,
    find_backport_label_adder,
    resolve_author,
)

__all__ = [
    "CVE_TIMEFRAMES",
    "PR",
    "PRIORITY_URGENCY",
    "SLACK_USER_MAP",
    "URGENCY_ORDER",
    "BackportAuditError",
    "Config",
    "GitHubClient",
    "GitHubError",
    "JiraClient",
    "JiraError",
    "JiraIssue",
    "ReleaseBranch",
    "__version__",
    "calculate_urgency",
    "detect_release_branches",
    "detect_release_version",
    "extract_jira_keys",
    "find_backport_label_adder",
    "format_deadline_info",
    "generate_markdown",
    "generate_slack_payload",
    "get_slack_mention",
    "parse_args",
    "parse_date",
    "resolve_author",
]
