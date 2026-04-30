#!/usr/bin/env python3
"""
Backport PR audit tool for StackRox release management.

Validates PRs and Jira issues for release branches, generating reports
for release managers to review before cutting releases.

Features:
- Auto-detects release branches and expected versions from git tags
- Fetches open backport PRs via gh CLI
- Resolves real authors (handles rhacs-bot, dependabot)
- Validates Jira issues via REST API
- Finds orphaned Jira issues (in fixVersion but no PR)
- Generates markdown report and Slack JSON payload

Requirements:
- Python 3.9+ (stdlib only, no pip dependencies)
- gh CLI (authenticated)
- git (for tag/branch operations)
- Environment: JIRA_USER, JIRA_TOKEN

Development:
    # Install ruff for linting (optional)
    pip install ruff

    # Check code
    ruff check scripts/ci/backport_audit/
    ruff format scripts/ci/backport_audit/

Usage:
    # Audit all release branches
    ./scripts/ci/backport-audit.py --branches all

    # Audit specific branches
    ./scripts/ci/backport-audit.py --branches release-4.10,release-4.9

    # Custom output directory
    ./scripts/ci/backport-audit.py --output-dir /tmp

    # With GitHub Actions run URL (for Slack link)
    ./scripts/ci/backport-audit.py \\
        --github-run-url https://github.com/stackrox/stackrox/actions/runs/123

Outputs:
    - backport-audit-report.md: Markdown report for GitHub step summary
    - slack-payload.json: Slack Block Kit payload for posting

Exit Codes:
    0: Success
    1: Expected error (config, API failure)
    2: Unexpected error (bug)
"""

import sys
from backport_audit.__main__ import main

if __name__ == "__main__":
    sys.exit(main())
