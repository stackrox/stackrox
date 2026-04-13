"""GitHub client using gh CLI."""

import json
import subprocess
from typing import Any, Dict, List

from .models import GitHubError


class GitHubClient:
    """GitHub operations via gh CLI."""

    def fetch_prs(self, label: str = "backport", state: str = "open") -> List[Dict[str, Any]]:
        """
        Fetch PRs using gh CLI.

        Args:
            label: Label to filter by
            state: PR state (open, closed, all)

        Returns:
            List of PR dictionaries
        """
        cmd = [
            'gh', 'pr', 'list',
            '--repo', 'stackrox/stackrox',
            '--search', f'label:{label} draft:false',
            '--state', state,
            '--limit', '1000',
            '--json', 'number,title,author,baseRefName,body,state'
        ]

        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                check=True,
                timeout=60
            )
            return json.loads(result.stdout)
        except subprocess.CalledProcessError as e:
            raise GitHubError(f"Failed to fetch PRs: {e.stderr}")
        except subprocess.TimeoutExpired:
            raise GitHubError("gh CLI command timed out")
        except json.JSONDecodeError as e:
            raise GitHubError(f"Invalid JSON from gh CLI: {e}")

    def get_pr_details(self, pr_number: int) -> Dict[str, Any]:
        """
        Get PR details via gh CLI.

        Args:
            pr_number: PR number

        Returns:
            PR details dictionary
        """
        cmd = ['gh', 'pr', 'view', str(pr_number), '--json', 'author,body']

        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                check=True,
                timeout=30
            )
            return json.loads(result.stdout)
        except subprocess.CalledProcessError as e:
            raise GitHubError(f"Failed to fetch PR #{pr_number}: {e.stderr}")
        except subprocess.TimeoutExpired:
            raise GitHubError(f"gh CLI command timed out for PR #{pr_number}")
        except json.JSONDecodeError as e:
            raise GitHubError(f"Invalid JSON from gh CLI for PR #{pr_number}: {e}")

    def get_issue_events(self, pr_number: int) -> List[Dict[str, Any]]:
        """
        Get issue events via gh API.

        Args:
            pr_number: PR number

        Returns:
            List of event dictionaries
        """
        cmd = [
            'gh', 'api',
            f'repos/stackrox/stackrox/issues/{pr_number}/events'
        ]

        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                check=True,
                timeout=30
            )
            return json.loads(result.stdout)
        except subprocess.CalledProcessError as e:
            raise GitHubError(f"Failed to fetch events for PR #{pr_number}: {e.stderr}")
        except subprocess.TimeoutExpired:
            raise GitHubError(f"gh API command timed out for PR #{pr_number}")
        except json.JSONDecodeError as e:
            raise GitHubError(f"Invalid JSON from gh API for PR #{pr_number}: {e}")
