"""GitHub client using gh CLI."""

import json
import subprocess
from typing import Any

from .models import GitHubError


class GitHubClient:
    """GitHub operations via gh CLI."""

    def fetch_prs(self, label: str = "backport", state: str = "open") -> list[dict[str, Any]]:
        """Fetch PRs using gh CLI.

        Args:
            label: Label to filter by
            state: PR state (open, closed, all)

        Returns:
            List of PR dictionaries

        """
        cmd = [
            "gh", "pr", "list",
            "--repo", "stackrox/stackrox",
            "--search", f"label:{label} draft:false",
            "--state", state,
            "--limit", "1000",
            "--json", "number,title,author,baseRefName,body,state",
        ]

        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                check=True,
                timeout=60,
            )
            return json.loads(result.stdout)
        except subprocess.CalledProcessError as e:
            msg = f"Failed to fetch PRs: {e.stderr}"
            raise GitHubError(msg) from e
        except subprocess.TimeoutExpired as e:
            msg = "gh CLI command timed out"
            raise GitHubError(msg) from e
        except json.JSONDecodeError as e:
            msg = f"Invalid JSON from gh CLI: {e}"
            raise GitHubError(msg) from e

    def get_pr_details(self, pr_number: int) -> dict[str, Any]:
        """Get PR details via gh CLI.

        Args:
            pr_number: PR number

        Returns:
            PR details dictionary

        """
        cmd = ["gh", "pr", "view", str(pr_number), "--json", "author,body,title"]

        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                check=True,
                timeout=30,
            )
            return json.loads(result.stdout)
        except subprocess.CalledProcessError as e:
            msg = f"Failed to fetch PR #{pr_number}: {e.stderr}"
            raise GitHubError(msg) from e
        except subprocess.TimeoutExpired as e:
            msg = f"gh CLI command timed out for PR #{pr_number}"
            raise GitHubError(msg) from e
        except json.JSONDecodeError as e:
            msg = f"Invalid JSON from gh CLI for PR #{pr_number}: {e}"
            raise GitHubError(msg) from e

    def get_issue_events(self, pr_number: int) -> list[dict[str, Any]]:
        """Get issue events via gh API.

        Args:
            pr_number: PR number

        Returns:
            List of event dictionaries

        """
        cmd = [
            "gh", "api",
            f"repos/stackrox/stackrox/issues/{pr_number}/events",
        ]

        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                check=True,
                timeout=30,
            )
            return json.loads(result.stdout)
        except subprocess.CalledProcessError as e:
            msg = f"Failed to fetch events for PR #{pr_number}: {e.stderr}"
            raise GitHubError(msg) from e
        except subprocess.TimeoutExpired as e:
            msg = f"gh API command timed out for PR #{pr_number}"
            raise GitHubError(msg) from e
        except json.JSONDecodeError as e:
            msg = f"Invalid JSON from gh API for PR #{pr_number}: {e}"
            raise GitHubError(msg) from e
