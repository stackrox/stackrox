"""Data models and exceptions for backport audit tool."""

from dataclasses import dataclass
from typing import List, Optional


# Exception hierarchy
class BackportAuditError(Exception):
    """Base exception for backport audit tool."""


class GitHubError(BackportAuditError):
    """GitHub API/CLI error."""


class JiraError(BackportAuditError):
    """Jira API error."""


@dataclass
class PR:
    """Pull request data."""
    number: int
    title: str
    author: str
    base_ref: str
    jira_keys: List[str]
    body: str


@dataclass
class JiraIssue:
    """Jira issue data."""
    key: str
    summary: str
    fix_versions: List[str]
    affected_versions: List[str]
    assignee: Optional[str]
    team: Optional[str]
    component: Optional[str]


@dataclass
class ReleaseBranch:
    """Release branch with version info."""
    name: str
    expected_version: str
    latest_tag: Optional[str]
