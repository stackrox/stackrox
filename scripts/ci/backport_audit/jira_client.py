"""Jira client using urllib."""

import base64
import json
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import Request, urlopen

from .models import JiraIssue

# HTTP status codes
HTTP_NOT_FOUND = 404


class JiraClient:
    """Jira REST API client using urllib."""

    def __init__(self, user: str, token: str, base_url: str = "redhat.atlassian.net") -> None:
        self.user = user
        self.token = token
        self.base_url = base_url
        self._auth_header = self._make_auth_header()

    def _make_auth_header(self) -> str:
        """Create Basic Auth header."""
        credentials = f"{self.user}:{self.token}"
        encoded = base64.b64encode(credentials.encode()).decode()
        return f"Basic {encoded}"

    def get_issue(self, issue_key: str) -> JiraIssue | None:
        """Fetch Jira issue via REST API.

        Args:
            issue_key: Jira issue key (e.g., ROX-12345)

        Returns:
            JiraIssue or None if not found

        """
        # Fields per Patch Release Process (https://redhat.atlassian.net/wiki/spaces/StackRox/pages/309338452):
        # - priority: Bug priority (Critical→immediate Z-release, Major→next Z-stream, Normal→unlikely)
        # - duedate: "defines internal deadline for releasing a version with the fix"
        # - customfield_10001: Team field
        # - customfield_10840: Severity field - "contains the CVE severity rating which affects urgency"
        # TODO: Add SLA Date field once discovered - "informs about the legally binding deadline for Red Hat"
        fields = "fixVersions,versions,summary,status,assignee,components,customfield_10001,priority,duedate,customfield_10840"
        url = f"https://{self.base_url}/rest/api/3/issue/{issue_key}?fields={fields}"

        req = Request(url)
        req.add_header("Authorization", self._auth_header)
        req.add_header("Content-Type", "application/json")

        try:
            with urlopen(req, timeout=30) as response:
                data = json.loads(response.read())
                return self._parse_issue(data)
        except HTTPError as e:
            if e.code == HTTP_NOT_FOUND:
                return None
            return None
        except URLError:
            return None
        except json.JSONDecodeError:
            return None

    def _parse_issue(self, data: dict[str, Any]) -> JiraIssue:
        """Parse Jira API response into JiraIssue."""
        fields = data.get("fields", {})

        fix_versions = [v["name"] for v in fields.get("fixVersions", [])]
        affected_versions = [v["name"] for v in fields.get("versions", [])]

        assignee = None
        if fields.get("assignee"):
            assignee = fields["assignee"].get("displayName")

        team = None
        if fields.get("customfield_10001"):
            team = fields["customfield_10001"].get("name")

        components = [c["name"] for c in fields.get("components", [])]
        component = ", ".join(components) if components else None

        status = None
        if fields.get("status"):
            status = fields["status"].get("name")

        priority = None
        if fields.get("priority"):
            priority = fields["priority"].get("name")

        due_date = fields.get("duedate")

        severity = None
        severity_field = fields.get("customfield_10840")
        if severity_field:
            severity = severity_field.get("value")

        return JiraIssue(
            key=data["key"],
            summary=fields.get("summary", ""),
            fix_versions=fix_versions,
            affected_versions=affected_versions,
            assignee=assignee,
            team=team,
            component=component,
            status=status,
            priority=priority,
            severity=severity,
            due_date=due_date,
            sla_date=None,
        )

    def search_issues(self, jql: str, max_results: int = 1000) -> list[JiraIssue]:
        """Search Jira issues via JQL.

        Args:
            jql: JQL query string
            max_results: Maximum results to return

        Returns:
            List of JiraIssue objects

        """
        params = urlencode({
            "jql": jql,
            "fields": "key,summary",
            "maxResults": max_results,
        })
        url = f"https://{self.base_url}/rest/api/3/search?{params}"

        req = Request(url)
        req.add_header("Authorization", self._auth_header)
        req.add_header("Content-Type", "application/json")

        try:
            with urlopen(req, timeout=30) as response:
                data = json.loads(response.read())
                return [JiraIssue(
                        key=issue_data["key"],
                        summary=issue_data["fields"].get("summary", ""),
                        fix_versions=[],
                        affected_versions=[],
                        assignee=None,
                        team=None,
                        component=None,
                    ) for issue_data in data.get("issues", [])]
        except (HTTPError, URLError, json.JSONDecodeError):
            return []
