"""Configuration and argument parsing."""

import argparse
import os
from dataclasses import dataclass

from .models import BackportAuditError


@dataclass
class Config:
    """Configuration from environment and arguments."""

    jira_user: str
    jira_token: str
    jira_base_url: str = "redhat.atlassian.net"
    jira_project: str = "ROX"
    github_token: str | None = None
    output_dir: str = "."
    report_file: str = "backport-audit-report.md"
    slack_payload_file: str = "slack-payload.json"
    branches: str = "all"
    github_run_url: str | None = None

    @classmethod
    def from_env(cls, args: argparse.Namespace) -> "Config":
        """Create config from environment and CLI args."""
        jira_user = os.environ.get("JIRA_USER")
        jira_token = os.environ.get("JIRA_TOKEN")

        if not jira_user:
            msg = "JIRA_USER environment variable is required"
            raise BackportAuditError(msg)
        if not jira_token:
            msg = "JIRA_TOKEN environment variable is required"
            raise BackportAuditError(msg)

        return cls(
            jira_user=jira_user,
            jira_token=jira_token,
            jira_base_url=os.getenv("JIRA_BASE_URL", "redhat.atlassian.net"),
            jira_project=os.getenv("JIRA_PROJECT", "ROX"),
            github_token=os.getenv("GITHUB_TOKEN"),
            output_dir=args.output_dir,
            branches=args.branches,
            github_run_url=args.github_run_url,
        )


def parse_args() -> argparse.Namespace:
    """Parse command-line arguments."""
    parser = argparse.ArgumentParser(
        description="Audit backport PRs and validate Jira issues for release management.",
    )
    parser.add_argument(
        "--branches",
        default="all",
        help='Release branches (comma-separated or "all")',
    )
    parser.add_argument(
        "--output-dir",
        default=".",
        help="Output directory for reports",
    )
    parser.add_argument(
        "--github-run-url",
        help="GitHub Actions run URL for Slack link",
    )
    return parser.parse_args()
