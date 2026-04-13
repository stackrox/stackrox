"""Data models and exceptions for backport audit tool."""

from dataclasses import dataclass


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
    jira_keys: list[str]
    body: str


@dataclass
class JiraIssue:
    """Jira issue data."""

    key: str
    summary: str
    fix_versions: list[str]
    affected_versions: list[str]
    assignee: str | None
    team: str | None
    component: str | None
    priority: str | None = None
    severity: str | None = None
    due_date: str | None = None
    sla_date: str | None = None


@dataclass
class ReleaseBranch:
    """Release branch with version info."""

    name: str
    expected_version: str
    latest_tag: str | None
