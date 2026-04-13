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
    """Jira issue data.

    Urgency-related fields per Patch Release Process:
    https://redhat.atlassian.net/wiki/spaces/StackRox/pages/309338452/Patch+Release+Process

    - priority: Bug priority (Critical→immediate Z-release, Major→next Z-stream)
    - severity: CVE severity rating (Critical: 7 days, Important: 28 days, Moderate: 57 days)
    - due_date: "defines internal deadline for releasing a version with the fix"
    - sla_date: "informs about the legally binding deadline for Red Hat; usually is Due date + some buffer"
    """

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
