"""Backport PR audit tool for StackRox release management."""

__version__ = "1.0.0"

from .models import (
    BackportAuditError,
    GitHubError,
    JiraError,
    PR,
    JiraIssue,
    ReleaseBranch,
)
from .config import Config, parse_args
from .github_client import GitHubClient
from .jira_client import JiraClient
from .slack import SLACK_USER_MAP, get_slack_mention
from .utils import (
    extract_jira_keys,
    find_backport_label_adder,
    resolve_author,
    detect_release_branches,
    detect_release_version,
)
from .report_markdown import generate_markdown

__all__ = [
    "__version__",
    "BackportAuditError",
    "GitHubError",
    "JiraError",
    "PR",
    "JiraIssue",
    "ReleaseBranch",
    "Config",
    "parse_args",
    "GitHubClient",
    "JiraClient",
    "SLACK_USER_MAP",
    "get_slack_mention",
    "extract_jira_keys",
    "find_backport_label_adder",
    "resolve_author",
    "detect_release_branches",
    "detect_release_version",
    "generate_markdown",
]
