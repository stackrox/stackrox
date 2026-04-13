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
    merged: bool = False  # True if PR is merged (from git commits)
    commit_sha: str | None = None  # Git commit SHA (for merged PRs)


@dataclass
class JiraIssue:
    """Jira issue data.

    Critical fields for ProdSec vulnerability tracking:

    - fix_versions: MUST be set when issue is resolved as Done (marks vulnerability as Fixed)
    - affected_versions: MUST include all affected RHACS versions currently in Full Support
      WARNING: DO NOT CREATE SEPARATE JIRA ISSUES - amend this field to add versions
    - priority: Bug priority (Critical→immediate Z-release, Major→next Z-stream)
    - severity: CVE severity rating (Critical: 7 days, Important: 28 days, Moderate: 57 days)
      Policy (2025): Handle all Important/Critical + Moderate with CVSS >= 7.0
    - due_date: "defines internal deadline for releasing a version with the fix"
    - sla_date: "informs about the legally binding deadline for Red Hat; usually is Due date + some buffer"
      Note: Issues must be closed before SLA deadline, regardless of severity

    References:
    - Patch Release Process: https://redhat.atlassian.net/wiki/spaces/StackRox/pages/309338452/Patch+Release+Process
    - ProdSec Triage: https://redhat.atlassian.net/wiki/spaces/StackRox/pages/309334614/How+to+triage+and+resolve+ProdSec+Jiras
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
