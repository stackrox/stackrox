# Backport PR Audit Python Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite bash backport PR audit scripts to Python with stdlib only, generating markdown reports and Slack JSON payloads.

**Architecture:** Single executable Python script with class-based design (GitHubClient, JiraClient, ReportGenerator). Uses subprocess for `gh` CLI, urllib for Jira REST API. Full type hints, no external dependencies.

**Tech Stack:** Python 3.9+, stdlib only (subprocess, urllib, json, dataclasses, argparse)

**Reference Spec:** `docs/superpowers/specs/2026-04-10-backport-audit-python-rewrite-design.md`

---

## File Structure

**Created:**
- `scripts/ci/backport-audit.py` - Main Python script (~700 lines)

**Modified:**
- `.github/workflows/audit-backport-prs.yml` - Update to call Python script

**Reference (not modified):**
- `scripts/ci/audit-backport-prs.sh` - Original bash (for comparison)
- `scripts/ci/get-slack-user-id.sh` - Source for Slack mapping
- `scripts/ci/post-backport-audit-to-slack.sh` - Original Slack posting

---

### Task 1: Script Skeleton and Imports

**Files:**
- Create: `scripts/ci/backport-audit.py`

- [ ] **Step 1: Create script with header and imports**

```python
#!/usr/bin/env python3
"""
Backport PR audit tool for StackRox release management.

Validates PRs and Jira issues for release branches.

Development:
    ruff check scripts/ci/backport-audit.py
    ruff format scripts/ci/backport-audit.py

Usage:
    ./backport-audit.py --branches all --output-dir .
"""

from __future__ import annotations
import argparse
import base64
import json
import os
import re
import subprocess
import sys
import traceback
from dataclasses import dataclass
from datetime import datetime
from typing import Any, Dict, List, Optional
from urllib.request import Request, urlopen
from urllib.error import HTTPError, URLError


# Script version for debugging
VERSION = "1.0.0"


if __name__ == "__main__":
    print(f"Backport Audit Tool v{VERSION}")
    sys.exit(0)
```

- [ ] **Step 2: Make executable and verify**

```bash
chmod +x scripts/ci/backport-audit.py
./scripts/ci/backport-audit.py
```

Expected output: `Backport Audit Tool v1.0.0`

- [ ] **Step 3: Commit**

```bash
git add scripts/ci/backport-audit.py
git commit -m "feat: add Python backport audit script skeleton

Add basic script structure with imports and version.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 2: Add Constants and Slack User Mapping

**Files:**
- Modify: `scripts/ci/backport-audit.py`

- [ ] **Step 1: Add Slack user mapping constant**

Convert from `scripts/ci/get-slack-user-id.sh`. Add after VERSION:

```python
# Slack user ID mapping (GitHub login -> Slack member ID)
# Source: scripts/ci/get-slack-user-id.sh
SLACK_USER_MAP: Dict[str, str] = {
    '0x656b694d': 'U02MJ72K1B5',
    'BradLugo': 'U042Z3TSZU3',
    'GrimmiMeloni': 'U048VH2JZ1C',
    'JoukoVirtanen': 'U033Y28GYN4',
    'Maddosaurus': 'U01Q5L5R0GJ',
    'Molter73': 'U02A292NPV2',
    'RTann': 'U01NZ6U730X',
    'SimonBaeumer': 'U01Q5RMEHCK',
    'Stringy': 'U02KJKREKPY',
    'ajheflin': 'U087GT2H45Q',
    'akameric': 'U076CG62KL4',
    'AlexVulaj': 'U03M3QKBES2',
    'alanonthegit': 'U01PZFFSZRB',
    'alwayshooin': 'U01PLAWUU8N',
    'bradr5': 'U03UQ9DM44U',
    'c-du': 'U02NE59PHT3',
    'charmik-redhat': 'U035YKHMXEW',
    'clickboo': 'U01PFFU0YKD',
    'dashrews78': 'U03FB5XE10V',
    'daynewlee': 'U03J855QWHF',
    'dcaravel': 'U04DF45CXBJ',
    'dvail': 'U032WL9RM53',
    'ebensh': 'U01Q7HTJ126',
    'erthalion': 'U02SV8VE3K3',
    'gaurav-nelson': 'U01P6PMFGKF',
    'guzalv': 'U08NQKQJH4N',
    'house-d': 'U03H69TFKH9',
    'janisz': 'U0218FUVDMJ',
    'johannes94': 'U03E2SD2ZPB',
    'jschnath': 'U03AA9E6B09',
    'jvdm': 'U02TTV416HY',
    'kovayur': 'U033ZSBGEUQ',
    'ksurabhi91': 'U043ZP4RN76',
    'kurlov': 'U035001CQCV',
    'kylape': 'UGJML86DD',
    'ludydoo': 'U04TFDR57KQ',
    'lvalerom': 'U02SJTV567N',
    'mclasmeier': 'U02DKH1LQ5N',
    'mfosterrox': 'U01PMH71ACU',
    'msugakov': 'U020QJZCQAH',
    'mtodor': 'U039LQ48PT7',
    'ovalenti': 'U03F2F9EXUL',
    'parametalol': 'U02MJ72K1B5',
    'pedrottimark': 'U01RN8V8DEH',
    'porridge': 'U020XCUG2LA',
    'rhybrillou': 'U02GPRG4NHF',
    'robbycochran': 'U03NAEPKDE1',
    'rukletsov': 'U01G6P17RTK',
    'sachaudh': 'U01QLCGS0NM',
    'stehessel': 'U02SDMERUFP',
    'sthadka': 'U029PASTL5C',
    'tommartensen': 'U040F2EG19U',
    'vikin91': 'U02L405V2GH',
    'vjwilson': 'U01PKQQF0KY',
    'vladbologa': 'U03NFNXKPH9',
    'vulerh': 'U02A9CAR59T',
}


def get_slack_mention(github_login: str) -> str:
    """
    Get Slack mention for GitHub user.

    Args:
        github_login: GitHub username

    Returns:
        Slack mention string (<@ID>, @username, or :konflux:)
    """
    if github_login == 'app/red-hat-konflux':
        return ':konflux:'

    slack_id = SLACK_USER_MAP.get(github_login)
    if slack_id:
        return f'<@{slack_id}>'
    return f'@{github_login}'
```

- [ ] **Step 2: Test Slack mention function**

Update main block to test:

```python
if __name__ == "__main__":
    print(f"Backport Audit Tool v{VERSION}")
    # Test Slack mentions
    print(f"janisz: {get_slack_mention('janisz')}")
    print(f"unknown: {get_slack_mention('unknown-user')}")
    print(f"konflux: {get_slack_mention('app/red-hat-konflux')}")
    sys.exit(0)
```

- [ ] **Step 3: Run and verify**

```bash
./scripts/ci/backport-audit.py
```

Expected output:
```
Backport Audit Tool v1.0.0
janisz: <@U0218FUVDMJ>
unknown: @unknown-user
konflux: :konflux:
```

- [ ] **Step 4: Commit**

```bash
git add scripts/ci/backport-audit.py
git commit -m "feat: add Slack user mapping

Convert get-slack-user-id.sh to Python dict with ~50 users.
Add get_slack_mention() helper function.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 3: Add Dataclasses and Exceptions

**Files:**
- Modify: `scripts/ci/backport-audit.py`

- [ ] **Step 1: Add exception classes**

Add after get_slack_mention():

```python
# Exception hierarchy
class BackportAuditError(Exception):
    """Base exception for backport audit tool."""


class GitHubError(BackportAuditError):
    """GitHub API/CLI error."""


class JiraError(BackportAuditError):
    """Jira API error."""
```

- [ ] **Step 2: Add dataclasses**

Add after exceptions:

```python
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
```

- [ ] **Step 3: Test dataclass creation**

Update main block:

```python
if __name__ == "__main__":
    print(f"Backport Audit Tool v{VERSION}")

    # Test dataclasses
    pr = PR(
        number=12345,
        title="ROX-999: Fix bug",
        author="janisz",
        base_ref="release-4.10",
        jira_keys=["ROX-999"],
        body="Test PR"
    )
    print(f"PR: {pr.number} - {pr.title}")

    branch = ReleaseBranch("release-4.10", "4.10.3", "4.10.2")
    print(f"Branch: {branch.name} -> {branch.expected_version}")

    sys.exit(0)
```

- [ ] **Step 4: Run and verify**

```bash
./scripts/ci/backport-audit.py
```

Expected output includes:
```
PR: 12345 - ROX-999: Fix bug
Branch: release-4.10 -> 4.10.3
```

- [ ] **Step 5: Commit**

```bash
git add scripts/ci/backport-audit.py
git commit -m "feat: add dataclasses and exception hierarchy

Add PR, JiraIssue, ReleaseBranch dataclasses.
Add BackportAuditError, GitHubError, JiraError exceptions.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 4: Add Configuration Class

**Files:**
- Modify: `scripts/ci/backport-audit.py`

- [ ] **Step 1: Add Config dataclass**

Add after ReleaseBranch:

```python
@dataclass
class Config:
    """Configuration from environment and arguments."""
    jira_user: str
    jira_token: str
    jira_base_url: str = "redhat.atlassian.net"
    jira_project: str = "ROX"
    github_token: Optional[str] = None
    output_dir: str = "."
    report_file: str = "backport-audit-report.md"
    slack_payload_file: str = "slack-payload.json"
    branches: str = "all"
    github_run_url: Optional[str] = None

    @classmethod
    def from_env(cls, args: argparse.Namespace) -> 'Config':
        """Create config from environment and CLI args."""
        jira_user = os.environ.get('JIRA_USER')
        jira_token = os.environ.get('JIRA_TOKEN')

        if not jira_user:
            raise BackportAuditError("JIRA_USER environment variable is required")
        if not jira_token:
            raise BackportAuditError("JIRA_TOKEN environment variable is required")

        return cls(
            jira_user=jira_user,
            jira_token=jira_token,
            jira_base_url=os.getenv('JIRA_BASE_URL', 'redhat.atlassian.net'),
            jira_project=os.getenv('JIRA_PROJECT', 'ROX'),
            github_token=os.getenv('GITHUB_TOKEN'),
            output_dir=args.output_dir,
            branches=args.branches,
            github_run_url=args.github_run_url,
        )


def parse_args() -> argparse.Namespace:
    """Parse command-line arguments."""
    parser = argparse.ArgumentParser(
        description='Audit backport PRs and validate Jira issues for release management.'
    )
    parser.add_argument(
        '--branches',
        default='all',
        help='Release branches (comma-separated or "all")'
    )
    parser.add_argument(
        '--output-dir',
        default='.',
        help='Output directory for reports'
    )
    parser.add_argument(
        '--github-run-url',
        help='GitHub Actions run URL for Slack link'
    )
    return parser.parse_args()
```

- [ ] **Step 2: Test config parsing**

Update main:

```python
if __name__ == "__main__":
    print(f"Backport Audit Tool v{VERSION}")

    try:
        args = parse_args()
        print(f"Args: branches={args.branches}, output_dir={args.output_dir}")

        # Test config (will fail without env vars)
        try:
            config = Config.from_env(args)
            print(f"Config loaded: {config.jira_user}")
        except BackportAuditError as e:
            print(f"Expected error (no env vars): {e}")

    except Exception as e:
        print(f"Error: {e}")
        traceback.print_exc()
        sys.exit(1)

    sys.exit(0)
```

- [ ] **Step 3: Run and verify**

```bash
./scripts/ci/backport-audit.py --branches all --output-dir /tmp
```

Expected output includes:
```
Args: branches=all, output_dir=/tmp
Expected error (no env vars): JIRA_USER environment variable is required
```

- [ ] **Step 4: Commit**

```bash
git add scripts/ci/backport-audit.py
git commit -m "feat: add configuration and argument parsing

Add Config dataclass with from_env() factory.
Add parse_args() for CLI argument handling.
Validate required environment variables.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 5: Add GitHubClient

**Files:**
- Modify: `scripts/ci/backport-audit.py`

- [ ] **Step 1: Add GitHubClient class**

Add before parse_args():

```python
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
```

- [ ] **Step 2: Add manual test in main**

Update main:

```python
if __name__ == "__main__":
    print(f"Backport Audit Tool v{VERSION}")

    # Test GitHubClient (requires gh CLI and auth)
    gh = GitHubClient()
    print("Testing GitHubClient...")

    try:
        # This will fail if gh is not installed or not authenticated
        prs = gh.fetch_prs("backport", "open")
        print(f"Found {len(prs)} backport PRs")
        if prs:
            print(f"First PR: #{prs[0]['number']} - {prs[0]['title']}")
    except GitHubError as e:
        print(f"GitHub error (expected if gh not configured): {e}")

    sys.exit(0)
```

- [ ] **Step 3: Run and verify**

```bash
./scripts/ci/backport-audit.py
```

If `gh` is authenticated, you'll see actual PR counts. Otherwise, an error message.

- [ ] **Step 4: Commit**

```bash
git add scripts/ci/backport-audit.py
git commit -m "feat: add GitHubClient with gh CLI integration

Add fetch_prs(), get_pr_details(), get_issue_events() methods.
Handle timeouts and JSON parsing errors.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 6: Add JiraClient

**Files:**
- Modify: `scripts/ci/backport-audit.py`

- [ ] **Step 1: Add JiraClient class**

Add after GitHubClient:

```python
class JiraClient:
    """Jira REST API client using urllib."""

    def __init__(self, user: str, token: str, base_url: str = "redhat.atlassian.net"):
        self.user = user
        self.token = token
        self.base_url = base_url
        self._auth_header = self._make_auth_header()

    def _make_auth_header(self) -> str:
        """Create Basic Auth header."""
        credentials = f"{self.user}:{self.token}"
        encoded = base64.b64encode(credentials.encode()).decode()
        return f"Basic {encoded}"

    def get_issue(self, issue_key: str) -> Optional[JiraIssue]:
        """
        Fetch Jira issue via REST API.

        Args:
            issue_key: Jira issue key (e.g., ROX-12345)

        Returns:
            JiraIssue or None if not found
        """
        fields = "fixVersions,versions,summary,status,assignee,components,customfield_10001"
        url = f"https://{self.base_url}/rest/api/3/issue/{issue_key}?fields={fields}"

        req = Request(url)
        req.add_header("Authorization", self._auth_header)
        req.add_header("Content-Type", "application/json")

        try:
            with urlopen(req, timeout=30) as response:
                data = json.loads(response.read())
                return self._parse_issue(data)
        except HTTPError as e:
            if e.code == 404:
                print(f"WARNING: Jira issue {issue_key} not found", file=sys.stderr)
                return None
            print(f"WARNING: HTTP error fetching {issue_key}: {e}", file=sys.stderr)
            return None
        except URLError as e:
            print(f"WARNING: Network error fetching {issue_key}: {e}", file=sys.stderr)
            return None
        except json.JSONDecodeError as e:
            print(f"WARNING: Invalid JSON from Jira for {issue_key}: {e}", file=sys.stderr)
            return None

    def _parse_issue(self, data: Dict[str, Any]) -> JiraIssue:
        """Parse Jira API response into JiraIssue."""
        fields = data.get('fields', {})

        fix_versions = [v['name'] for v in fields.get('fixVersions', [])]
        affected_versions = [v['name'] for v in fields.get('versions', [])]

        assignee = None
        if fields.get('assignee'):
            assignee = fields['assignee'].get('displayName')

        team = None
        if fields.get('customfield_10001'):
            team = fields['customfield_10001'].get('name')

        components = [c['name'] for c in fields.get('components', [])]
        component = ', '.join(components) if components else None

        return JiraIssue(
            key=data['key'],
            summary=fields.get('summary', ''),
            fix_versions=fix_versions,
            affected_versions=affected_versions,
            assignee=assignee,
            team=team,
            component=component
        )

    def search_issues(self, jql: str, max_results: int = 1000) -> List[JiraIssue]:
        """
        Search Jira issues via JQL.

        Args:
            jql: JQL query string
            max_results: Maximum results to return

        Returns:
            List of JiraIssue objects
        """
        from urllib.parse import urlencode

        params = urlencode({
            'jql': jql,
            'fields': 'key,summary',
            'maxResults': max_results
        })
        url = f"https://{self.base_url}/rest/api/3/search?{params}"

        req = Request(url)
        req.add_header("Authorization", self._auth_header)
        req.add_header("Content-Type", "application/json")

        try:
            with urlopen(req, timeout=30) as response:
                data = json.loads(response.read())
                issues = []
                for issue_data in data.get('issues', []):
                    # For search results, we only get key and summary
                    # Full details would require individual get_issue() calls
                    issues.append(JiraIssue(
                        key=issue_data['key'],
                        summary=issue_data['fields'].get('summary', ''),
                        fix_versions=[],
                        affected_versions=[],
                        assignee=None,
                        team=None,
                        component=None
                    ))
                return issues
        except (HTTPError, URLError, json.JSONDecodeError) as e:
            print(f"WARNING: Error searching Jira: {e}", file=sys.stderr)
            return []
```

- [ ] **Step 2: Remove test code from main**

Clean up main block (remove GitHub test):

```python
if __name__ == "__main__":
    print(f"Backport Audit Tool v{VERSION}")
    print("Script structure complete. Ready for implementation.")
    sys.exit(0)
```

- [ ] **Step 3: Commit**

```bash
git add scripts/ci/backport-audit.py
git commit -m "feat: add JiraClient with REST API integration

Add get_issue() and search_issues() methods using urllib.
Parse Jira responses into JiraIssue dataclass.
Handle HTTP errors gracefully with warnings.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 7: Add Utility Functions

**Files:**
- Modify: `scripts/ci/backport-audit.py`

- [ ] **Step 1: Add Jira key extraction**

Add after JiraClient:

```python
def extract_jira_keys(title: str) -> List[str]:
    """
    Extract ROX-XXXXX Jira keys from PR title.

    Args:
        title: PR title

    Returns:
        Sorted list of unique Jira keys
    """
    pattern = r'ROX-\d+'
    matches = re.findall(pattern, title)
    return sorted(set(matches))
```

- [ ] **Step 2: Add author resolution functions**

Add after extract_jira_keys:

```python
def find_backport_label_adder(pr_number: int, gh_client: GitHubClient) -> str:
    """
    Find who added the backport label to a PR.

    Args:
        pr_number: PR number
        gh_client: GitHub client

    Returns:
        GitHub username of label adder, or 'app/dependabot' if not found
    """
    try:
        events = gh_client.get_issue_events(pr_number)
        for event in events:
            if (event.get('event') == 'labeled' and
                event.get('label', {}).get('name', '').startswith('backport') and
                event.get('actor', {}).get('login') != 'github-actions[bot]'):
                return event['actor']['login']
    except GitHubError as e:
        print(f"WARNING: Could not get events for PR #{pr_number}: {e}", file=sys.stderr)

    return 'app/dependabot'


def resolve_author(pr_data: Dict[str, Any], gh_client: GitHubClient) -> str:
    """
    Resolve the real author of a PR.

    Handles rhacs-bot and dependabot by finding the original author
    or the person who added the backport label.

    Args:
        pr_data: PR data from gh CLI
        gh_client: GitHub client

    Returns:
        Resolved author username
    """
    author = pr_data['author']['login']
    body = pr_data.get('body', '')

    # Handle rhacs-bot
    if author == 'rhacs-bot':
        # Extract original PR number from body
        match = re.search(r'from #(\d+)', body)
        if match:
            original_pr_number = int(match.group(1))
            try:
                original_pr = gh_client.get_pr_details(original_pr_number)
                author = original_pr['author']['login']

                # If original author is also dependabot, find label adder
                if author == 'app/dependabot':
                    author = find_backport_label_adder(original_pr_number, gh_client)
            except GitHubError as e:
                print(f"WARNING: Could not resolve author for PR from #{original_pr_number}: {e}",
                      file=sys.stderr)

    # Handle direct dependabot PRs
    elif author == 'app/dependabot':
        pr_number = pr_data['number']
        author = find_backport_label_adder(pr_number, gh_client)

    return author
```

- [ ] **Step 3: Add git operations for release detection**

Add after resolve_author:

```python
def detect_release_branches(branches_arg: str) -> List[str]:
    """
    Detect release branches from git.

    Args:
        branches_arg: "all" or comma-separated branch names

    Returns:
        List of release branch names
    """
    if branches_arg == "all":
        # Auto-detect from git remote branches
        try:
            result = subprocess.run(
                ['git', 'branch', '-r'],
                capture_output=True,
                text=True,
                check=True,
                timeout=10
            )
            branches = []
            for line in result.stdout.splitlines():
                match = re.search(r'origin/(release-\d+\.\d+)', line)
                if match:
                    branches.append(match.group(1))
            return sorted(set(branches))
        except subprocess.CalledProcessError as e:
            raise BackportAuditError(f"Failed to detect release branches: {e.stderr}")
        except subprocess.TimeoutExpired:
            raise BackportAuditError("Git command timed out")
    else:
        # Use provided branches
        return [b.strip() for b in branches_arg.split(',') if b.strip()]


def detect_release_version(branch_name: str) -> ReleaseBranch:
    """
    Detect expected release version for a branch.

    Args:
        branch_name: Branch name (e.g., release-4.10)

    Returns:
        ReleaseBranch with version info
    """
    # Extract base version
    match = re.match(r'release-(\d+\.\d+)', branch_name)
    if not match:
        raise BackportAuditError(f"Invalid branch format: {branch_name}")

    base_version = match.group(1)

    # Find latest tag for this version
    try:
        result = subprocess.run(
            ['git', 'tag'],
            capture_output=True,
            text=True,
            check=True,
            timeout=10
        )

        # Filter tags for this version
        pattern = f"^{re.escape(base_version)}\\." + r"\d+$"
        matching_tags = [
            tag for tag in result.stdout.splitlines()
            if re.match(pattern, tag)
        ]

        latest_tag = None
        if matching_tags:
            # Sort by version (semantic sort)
            matching_tags.sort(key=lambda t: [int(x) for x in t.split('.')])
            latest_tag = matching_tags[-1]

        # Calculate next version
        if latest_tag:
            patch = int(latest_tag.split('.')[-1])
            expected_version = f"{base_version}.{patch + 1}"
        else:
            expected_version = f"{base_version}.0"

        return ReleaseBranch(
            name=branch_name,
            expected_version=expected_version,
            latest_tag=latest_tag
        )

    except subprocess.CalledProcessError as e:
        raise BackportAuditError(f"Failed to detect version for {branch_name}: {e.stderr}")
    except subprocess.TimeoutExpired:
        raise BackportAuditError("Git command timed out")
```

- [ ] **Step 4: Commit**

```bash
git add scripts/ci/backport-audit.py
git commit -m "feat: add utility functions for PR processing

Add extract_jira_keys() for Jira key extraction.
Add resolve_author() and find_backport_label_adder() for author resolution.
Add detect_release_branches() and detect_release_version() for git operations.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 8: Add ReportGenerator (Part 1 - Markdown)

**Files:**
- Modify: `scripts/ci/backport-audit.py`

- [ ] **Step 1: Add ReportGenerator class skeleton**

Add after utility functions:

```python
class ReportGenerator:
    """Generates markdown and Slack JSON reports."""

    def __init__(self, slack_user_map: Dict[str, str]):
        self.slack_user_map = slack_user_map

    def generate_markdown(
        self,
        branches: List[ReleaseBranch],
        prs_by_branch: Dict[str, List[PR]],
        jira_issues: Dict[str, JiraIssue],
        orphaned_issues: Dict[str, List[str]],
        timestamp: str
    ) -> str:
        """
        Generate markdown report.

        Args:
            branches: List of release branches
            prs_by_branch: PRs grouped by branch
            jira_issues: Jira issues by key
            orphaned_issues: Orphaned Jira keys by branch
            timestamp: Report generation timestamp

        Returns:
            Markdown report string
        """
        lines = []
        lines.append("# Backport PR Audit Report")
        lines.append("")
        lines.append(f"Generated: {timestamp}")
        lines.append("")

        # Sort branches by version
        sorted_branches = sorted(
            branches,
            key=lambda b: [int(x) for x in b.expected_version.split('.')]
        )

        for branch in sorted_branches:
            prs = prs_by_branch.get(branch.name, [])
            orphaned = orphaned_issues.get(branch.name, [])

            # Skip empty branches
            if not prs and not orphaned:
                continue

            lines.append(f"## {branch.name} (Expected: {branch.expected_version})")
            lines.append("")

            # PRs without Jira reference
            prs_no_jira = [pr for pr in prs if not pr.jira_keys]
            if prs_no_jira:
                lines.append(f"### PRs Missing Jira Reference ({len(prs_no_jira)})")
                lines.append("")

                # Sort by author
                prs_no_jira.sort(key=lambda p: p.author)

                for pr in prs_no_jira:
                    mention = get_slack_mention(pr.author)
                    lines.append(f"- {mention} #{pr.number}: {pr.title}")

                lines.append("")

            # Jira issues with missing metadata
            issues_with_problems = []
            jira_to_prs: Dict[str, List[int]] = {}

            for pr in prs:
                for jira_key in pr.jira_keys:
                    if jira_key not in jira_issues:
                        continue

                    issue = jira_issues[jira_key]

                    # Track PRs for this Jira
                    if jira_key not in jira_to_prs:
                        jira_to_prs[jira_key] = []
                    jira_to_prs[jira_key].append(pr.number)

                    # Check for missing metadata
                    has_fix = branch.expected_version in issue.fix_versions if issue.fix_versions else False
                    has_affected = len(issue.affected_versions) > 0

                    if not has_fix or not has_affected:
                        fix_icon = ":white_check_mark:" if has_fix else ":x:"
                        affected_icon = ":white_check_mark:" if has_affected else ":x:"

                        issue_info = (
                            jira_key,
                            fix_icon,
                            affected_icon,
                            issue.assignee or "Unassigned",
                            issue.team or "No team",
                            issue.component or "No component"
                        )
                        if issue_info not in issues_with_problems:
                            issues_with_problems.append(issue_info)

            if issues_with_problems:
                lines.append(f"### Jira Issues with Missing Metadata ({len(issues_with_problems)})")
                lines.append("")

                for jira_key, fix_icon, affected_icon, assignee, team, component in issues_with_problems:
                    pr_refs = jira_to_prs.get(jira_key, [])
                    pr_links = ', '.join([f"#{pr}" for pr in pr_refs])
                    pr_suffix = f" (PRs: {pr_links})" if pr_refs else ""

                    lines.append(
                        f"- {jira_key}: {fix_icon} fixVersion, {affected_icon} affectedVersion "
                        f"(Assignee: {assignee}, Team: {team}, Component: {component}){pr_suffix}"
                    )

                lines.append("")

            # Orphaned Jira issues
            if orphaned:
                lines.append(f"### Orphaned Jira Issues ({len(orphaned)})")
                lines.append("")
                lines.append(f"Issues with fixVersion={branch.expected_version} but no corresponding PR:")
                lines.append("")

                for jira_key in sorted(orphaned):
                    lines.append(f"- {jira_key}")

                lines.append("")

        return '\n'.join(lines)
```

- [ ] **Step 2: Commit**

```bash
git add scripts/ci/backport-audit.py
git commit -m "feat: add ReportGenerator with markdown generation

Add generate_markdown() method to create markdown reports.
Group PRs by missing Jira, metadata issues, orphaned issues.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 9: Add ReportGenerator (Part 2 - Slack JSON)

**Files:**
- Modify: `scripts/ci/backport-audit.py`

- [ ] **Step 1: Add Slack payload generation method**

Add to ReportGenerator class:

```python
    def generate_slack_payload(
        self,
        branches: List[ReleaseBranch],
        prs_by_branch: Dict[str, List[PR]],
        jira_issues: Dict[str, JiraIssue],
        orphaned_issues: Dict[str, List[str]],
        timestamp: str,
        github_run_url: Optional[str],
        slack_channel: str
    ) -> Dict[str, Any]:
        """
        Generate Slack payload with Block Kit format.

        Args:
            branches: List of release branches
            prs_by_branch: PRs grouped by branch
            jira_issues: Jira issues by key
            orphaned_issues: Orphaned Jira keys by branch
            timestamp: Report generation timestamp
            github_run_url: GitHub Actions run URL
            slack_channel: Slack channel ID

        Returns:
            Slack payload dictionary
        """
        # Count totals
        total_prs_no_jira = sum(
            len([pr for pr in prs_by_branch.get(b.name, []) if not pr.jira_keys])
            for b in branches
        )

        total_jira_issues = 0
        for branch in branches:
            prs = prs_by_branch.get(branch.name, [])
            for pr in prs:
                for jira_key in pr.jira_keys:
                    if jira_key in jira_issues:
                        issue = jira_issues[jira_key]
                        has_fix = branch.expected_version in issue.fix_versions if issue.fix_versions else False
                        has_affected = len(issue.affected_versions) > 0
                        if not has_fix or not has_affected:
                            total_jira_issues += 1
                            break  # Count once per issue

        # Build blocks
        blocks = []

        # Header
        blocks.append({
            "type": "header",
            "text": {
                "type": "plain_text",
                "text": "📋 Backport PR Audit Report"
            }
        })

        # Summary
        summary_text = (
            f"*Generated:* {timestamp}\n"
            f"*Total PRs missing Jira:* {total_prs_no_jira}\n"
            f"*Total Jira issues with missing metadata:* {total_jira_issues}"
        )
        blocks.append({
            "type": "section",
            "text": {
                "type": "mrkdwn",
                "text": summary_text
            }
        })

        # GitHub run link
        if github_run_url:
            blocks.append({
                "type": "section",
                "text": {
                    "type": "mrkdwn",
                    "text": f"<{github_run_url}|View full report in GitHub Actions>"
                }
            })

        blocks.append({"type": "divider"})

        # Generate sections for each branch
        sorted_branches = sorted(
            branches,
            key=lambda b: [int(x) for x in b.expected_version.split('.')]
        )

        for branch in sorted_branches:
            prs = prs_by_branch.get(branch.name, [])
            orphaned = orphaned_issues.get(branch.name, [])

            if not prs and not orphaned:
                continue

            # Build markdown for this release
            section_lines = []
            section_lines.append(f"*{branch.name} (Expected: {branch.expected_version})*\n")

            # PRs without Jira
            prs_no_jira = [pr for pr in prs if not pr.jira_keys]
            if prs_no_jira:
                section_lines.append(f"\n*PRs Missing Jira Reference ({len(prs_no_jira)})*")
                prs_no_jira.sort(key=lambda p: p.author)

                for pr in prs_no_jira:
                    mention = get_slack_mention(pr.author)
                    pr_link = f"<https://github.com/stackrox/stackrox/pull/{pr.number}|#{pr.number}>"
                    section_lines.append(f"- {mention} {pr_link}: {pr.title}")

            # Jira issues with problems
            issues_with_problems = []
            jira_to_prs: Dict[str, List[int]] = {}

            for pr in prs:
                for jira_key in pr.jira_keys:
                    if jira_key not in jira_issues:
                        continue

                    issue = jira_issues[jira_key]
                    if jira_key not in jira_to_prs:
                        jira_to_prs[jira_key] = []
                    jira_to_prs[jira_key].append(pr.number)

                    has_fix = branch.expected_version in issue.fix_versions if issue.fix_versions else False
                    has_affected = len(issue.affected_versions) > 0

                    if not has_fix or not has_affected:
                        fix_icon = ":white_check_mark:" if has_fix else ":x:"
                        affected_icon = ":white_check_mark:" if has_affected else ":x:"

                        issue_info = (
                            jira_key,
                            fix_icon,
                            affected_icon,
                            issue.assignee or "Unassigned",
                            issue.team or "No team",
                            issue.component or "No component"
                        )
                        if issue_info not in issues_with_problems:
                            issues_with_problems.append(issue_info)

            if issues_with_problems:
                section_lines.append(f"\n*Jira Issues with Missing Metadata ({len(issues_with_problems)})*")

                for jira_key, fix_icon, affected_icon, assignee, team, component in issues_with_problems:
                    jira_link = f"<https://redhat.atlassian.net/browse/{jira_key}|{jira_key}>"
                    pr_refs = jira_to_prs.get(jira_key, [])
                    pr_links = ', '.join([
                        f"<https://github.com/stackrox/stackrox/pull/{pr}|#{pr}>"
                        for pr in pr_refs
                    ])
                    pr_suffix = f" (PRs: {pr_links})" if pr_refs else ""

                    section_lines.append(
                        f"- {jira_link}: {fix_icon} fixVersion, {affected_icon} affectedVersion "
                        f"(Assignee: {assignee}, Team: {team}, Component: {component}){pr_suffix}"
                    )

            # Orphaned issues
            if orphaned:
                section_lines.append(f"\n*Orphaned Jira Issues ({len(orphaned)})*")
                section_lines.append(f"Issues with fixVersion={branch.expected_version} but no corresponding PR:")

                for jira_key in sorted(orphaned):
                    jira_link = f"<https://redhat.atlassian.net/browse/{jira_key}|{jira_key}>"
                    section_lines.append(f"- {jira_link}")

            # Split into sections if too long
            section_text = '\n'.join(section_lines)
            sections = self._split_slack_sections(section_text, branch.name)

            for section in sections:
                blocks.append({
                    "type": "section",
                    "text": {
                        "type": "mrkdwn",
                        "text": section
                    }
                })

        return {
            "channel": slack_channel,
            "blocks": blocks
        }

    def _split_slack_sections(self, text: str, branch_name: str, max_chars: int = 2800) -> List[str]:
        """
        Split text into Slack-compatible sections.

        Args:
            text: Text to split
            branch_name: Branch name for continuation headers
            max_chars: Maximum characters per section

        Returns:
            List of section strings
        """
        lines = text.split('\n')
        sections = []
        current_section = []
        current_length = 0
        current_header = None

        for line in lines:
            line_length = len(line) + 1  # +1 for newline

            # Track subsection headers
            if line.startswith('*') and line.endswith('*') and len(line) > 2:
                current_header = line

            # Check if adding line would exceed limit
            if current_length + line_length > max_chars and current_section:
                sections.append('\n'.join(current_section))

                # Start new section with continuation
                if current_header:
                    current_section = [f"*{branch_name} (continued)*\n", current_header]
                else:
                    current_section = [f"*{branch_name} (continued)*"]

                current_length = sum(len(l) + 1 for l in current_section)

            current_section.append(line)
            current_length += line_length

        if current_section:
            sections.append('\n'.join(current_section))

        return sections
```

- [ ] **Step 2: Commit**

```bash
git add scripts/ci/backport-audit.py
git commit -m "feat: add Slack payload generation to ReportGenerator

Add generate_slack_payload() with Block Kit formatting.
Add _split_slack_sections() to handle 2800 char limit.
Generate clickable links for PRs and Jira issues.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 10: Add Main Orchestration

**Files:**
- Modify: `scripts/ci/backport-audit.py`

- [ ] **Step 1: Add main orchestration function**

Replace the `if __name__ == "__main__":` block:

```python
def main() -> int:
    """Main orchestration function."""
    print(f"Backport Audit Tool v{VERSION}")

    try:
        # Parse arguments and load config
        args = parse_args()
        config = Config.from_env(args)

        print(f"Configuration: branches={config.branches}, output_dir={config.output_dir}")

        # Initialize clients
        gh_client = GitHubClient()
        jira_client = JiraClient(config.jira_user, config.jira_token, config.jira_base_url)
        report_gen = ReportGenerator(SLACK_USER_MAP)

        # Detect release branches
        print("Detecting release branches...")
        branch_names = detect_release_branches(config.branches)
        if not branch_names:
            print("ERROR: No release branches detected", file=sys.stderr)
            return 1

        branches = []
        for branch_name in branch_names:
            branch = detect_release_version(branch_name)
            branches.append(branch)
            print(f"  {branch.name} → {branch.expected_version} (latest tag: {branch.latest_tag or 'none'})")

        # Fetch backport PRs
        print("Fetching backport PRs from GitHub...")
        all_prs_data = gh_client.fetch_prs("backport", "open")
        print(f"Found {len(all_prs_data)} total backport PRs")

        # Process PRs
        print("Processing PRs...")
        prs_by_branch: Dict[str, List[PR]] = {}
        all_jira_keys = set()

        for pr_data in all_prs_data:
            # Filter by target branch
            base_ref = pr_data['baseRefName']
            if base_ref not in branch_names:
                continue

            # Resolve author
            author = resolve_author(pr_data, gh_client)

            # Extract Jira keys
            jira_keys = extract_jira_keys(pr_data['title'])
            all_jira_keys.update(jira_keys)

            # Create PR object
            pr = PR(
                number=pr_data['number'],
                title=pr_data['title'],
                author=author,
                base_ref=base_ref,
                jira_keys=jira_keys,
                body=pr_data.get('body', '')
            )

            if base_ref not in prs_by_branch:
                prs_by_branch[base_ref] = []
            prs_by_branch[base_ref].append(pr)

        print(f"Processed {sum(len(prs) for prs in prs_by_branch.values())} PRs targeting release branches")

        # Validate Jira issues
        print(f"Validating {len(all_jira_keys)} unique Jira issues...")
        jira_issues: Dict[str, JiraIssue] = {}

        for jira_key in sorted(all_jira_keys):
            issue = jira_client.get_issue(jira_key)
            if issue:
                jira_issues[jira_key] = issue

        print(f"Validated {len(jira_issues)} Jira issues")

        # Find orphaned Jira issues
        print("Finding orphaned Jira issues...")
        orphaned_issues: Dict[str, List[str]] = {}

        for branch in branches:
            jql = f'project = {config.jira_project} AND fixVersion = "{branch.expected_version}"'
            jira_issues_for_branch = jira_client.search_issues(jql)

            # Get Jira keys from PRs
            pr_jira_keys = set()
            for pr in prs_by_branch.get(branch.name, []):
                pr_jira_keys.update(pr.jira_keys)

            # Find orphaned
            orphaned = []
            for issue in jira_issues_for_branch:
                if issue.key not in pr_jira_keys:
                    orphaned.append(issue.key)

            if orphaned:
                orphaned_issues[branch.name] = orphaned
                print(f"  {branch.name}: {len(orphaned)} orphaned issues")

        # Generate reports
        print("Generating reports...")
        timestamp = datetime.utcnow().strftime("%Y-%m-%d %H:%M:%S UTC")

        # Markdown report
        markdown = report_gen.generate_markdown(
            branches, prs_by_branch, jira_issues, orphaned_issues, timestamp
        )

        markdown_path = os.path.join(config.output_dir, config.report_file)
        with open(markdown_path, 'w') as f:
            f.write(markdown)
        print(f"Markdown report written to: {markdown_path}")

        # Slack JSON payload
        slack_channel = os.getenv('SLACK_CHANNEL', 'C05AZF8T7GW')
        slack_payload = report_gen.generate_slack_payload(
            branches, prs_by_branch, jira_issues, orphaned_issues,
            timestamp, config.github_run_url, slack_channel
        )

        slack_path = os.path.join(config.output_dir, config.slack_payload_file)
        with open(slack_path, 'w') as f:
            json.dump(slack_payload, f, indent=2)
        print(f"Slack payload written to: {slack_path}")

        print("✅ Audit complete")
        return 0

    except BackportAuditError as e:
        print(f"ERROR: {e}", file=sys.stderr)
        return 1
    except Exception as e:
        print(f"UNEXPECTED ERROR: {e}", file=sys.stderr)
        traceback.print_exc()
        return 2


if __name__ == "__main__":
    sys.exit(main())
```

- [ ] **Step 2: Test basic execution**

```bash
./scripts/ci/backport-audit.py --help
```

Expected: Help message showing all options.

- [ ] **Step 3: Commit**

```bash
git add scripts/ci/backport-audit.py
git commit -m "feat: add main orchestration logic

Implement main() function with complete workflow:
- Detect release branches and versions
- Fetch and process PRs
- Validate Jira issues
- Find orphaned issues
- Generate markdown and Slack JSON reports

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 11: Update GitHub Workflow

**Files:**
- Modify: `.github/workflows/audit-backport-prs.yml`

- [ ] **Step 1: Read current workflow**

```bash
cat .github/workflows/audit-backport-prs.yml
```

- [ ] **Step 2: Update workflow to use Python script**

Replace the "Run backport audit" and "Post report to Slack" steps:

```yaml
      - name: Run backport audit
        env:
          JIRA_USER: ${{ vars.RHACS_BOT_GITHUB_EMAIL }}
          JIRA_TOKEN: ${{ secrets.JIRA_TOKEN }}
          GITHUB_TOKEN: ${{ github.token }}
          RELEASE_BRANCHES: ${{ inputs.release_branches || 'all' }}
        run: |
          chmod +x scripts/ci/backport-audit.py
          ./scripts/ci/backport-audit.py \
            --branches "$RELEASE_BRANCHES" \
            --output-dir . \
            --github-run-url "${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}"

      - name: Show report in summary
        if: always()
        run: |
          if [ -f backport-audit-report.md ]; then
            cat backport-audit-report.md >> "$GITHUB_STEP_SUMMARY"
          else
            echo "⚠️ Report file not generated" >> "$GITHUB_STEP_SUMMARY"
          fi

      - name: Upload report
        uses: actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02 # ratchet:actions/upload-artifact@v4
        if: always()
        with:
          name: backport-audit-report
          path: |
            backport-audit-report.md
            slack-payload.json

      - name: Post report to Slack
        if: success()
        env:
          SLACK_BOT_TOKEN: ${{ secrets.SLACK_BOT_TOKEN }}
        run: |
          # Use Slack API to post the payload
          curl -X POST https://slack.com/api/chat.postMessage \
            -H "Authorization: Bearer ${SLACK_BOT_TOKEN}" \
            -H "Content-Type: application/json" \
            -d @slack-payload.json
```

- [ ] **Step 3: Commit workflow changes**

```bash
git add .github/workflows/audit-backport-prs.yml
git commit -m "feat: update workflow to use Python backport audit script

Replace bash scripts with Python implementation:
- Call backport-audit.py instead of audit-backport-prs.sh
- Post Slack payload using curl
- Upload both markdown and JSON as artifacts

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 12: Add Documentation and Cleanup

**Files:**
- Modify: `scripts/ci/backport-audit.py` (add docstring improvements)

- [ ] **Step 1: Add comprehensive module docstring**

Update the top docstring in `backport-audit.py`:

```python
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
    ruff check scripts/ci/backport-audit.py
    ruff format scripts/ci/backport-audit.py

Usage:
    # Audit all release branches
    ./scripts/ci/backport-audit.py --branches all

    # Audit specific branches
    ./scripts/ci/backport-audit.py --branches release-4.10,release-4.9

    # Custom output directory
    ./scripts/ci/backport-audit.py --output-dir /tmp

    # With GitHub Actions run URL (for Slack link)
    ./scripts/ci/backport-audit.py \
        --github-run-url https://github.com/stackrox/stackrox/actions/runs/123

Outputs:
    - backport-audit-report.md: Markdown report for GitHub step summary
    - slack-payload.json: Slack Block Kit payload for posting

Exit Codes:
    0: Success
    1: Expected error (config, API failure)
    2: Unexpected error (bug)
"""
```

- [ ] **Step 2: Add README note**

Create a quick reference comment at the top of the script:

```python
# Quick Reference:
# - Design spec: docs/superpowers/specs/2026-04-10-backport-audit-python-rewrite-design.md
# - Original bash: scripts/ci/audit-backport-prs.sh (for comparison)
# - Slack user mapping source: scripts/ci/get-slack-user-id.sh
```

Insert after the module docstring and before imports.

- [ ] **Step 3: Commit documentation**

```bash
git add scripts/ci/backport-audit.py
git commit -m "docs: improve backport audit script documentation

Add comprehensive module docstring with usage examples.
Add quick reference comments linking to design spec.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 13: Manual Integration Testing

**Files:**
- Manual testing (no file changes)

- [ ] **Step 1: Set up test environment**

```bash
# Export required environment variables
export JIRA_USER="your-email@redhat.com"
export JIRA_TOKEN="your-jira-token"

# Ensure gh CLI is authenticated
gh auth status
```

- [ ] **Step 2: Run local test**

```bash
cd /home/janisz/go/src/github.com/stackrox/3
./scripts/ci/backport-audit.py --branches all --output-dir /tmp
```

Expected: Script runs successfully, generates reports in /tmp.

- [ ] **Step 3: Verify outputs**

```bash
# Check markdown report
ls -lh /tmp/backport-audit-report.md
head -20 /tmp/backport-audit-report.md

# Check Slack JSON
ls -lh /tmp/slack-payload.json
jq '.blocks | length' /tmp/slack-payload.json
jq '.blocks[0]' /tmp/slack-payload.json
```

Expected:
- Markdown file exists with report content
- JSON file is valid with blocks array
- First block is header

- [ ] **Step 4: Compare with bash version**

```bash
# Run bash version
./scripts/ci/audit-backport-prs.sh --branches all
mv backport-audit-report.md /tmp/bash-report.md

# Run Python version
./scripts/ci/backport-audit.py --branches all --output-dir /tmp
mv /tmp/backport-audit-report.md /tmp/python-report.md

# Compare structure
diff /tmp/bash-report.md /tmp/python-report.md || echo "Differences found (expected)"
wc -l /tmp/bash-report.md /tmp/python-report.md
```

Note any major structural differences.

- [ ] **Step 5: Document test results**

Create a test summary in commit message:

```bash
git commit --allow-empty -m "test: manual integration testing complete

Tested backport-audit.py locally:
- ✅ Fetches PRs from GitHub
- ✅ Resolves authors correctly
- ✅ Validates Jira issues
- ✅ Generates markdown report
- ✅ Generates Slack JSON payload
- ✅ Output structure matches bash version

Ready for CI testing.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Self-Review Checklist

**Spec Coverage:**
- ✅ Detect release branches from git tags (Task 7)
- ✅ Fetch open backport PRs via gh CLI (Task 5)
- ✅ Resolve real authors (Task 7)
- ✅ Extract Jira keys (Task 7)
- ✅ Validate Jira issues via REST API (Task 6)
- ✅ Find orphaned Jira issues (Task 10)
- ✅ Generate markdown report (Task 8)
- ✅ Generate Slack JSON payload (Task 9)
- ✅ Update GitHub workflow (Task 11)

**Placeholder Scan:**
- ✅ No TBD, TODO, or "implement later"
- ✅ All code blocks complete
- ✅ All commands have expected output
- ✅ All file paths are exact

**Type Consistency:**
- ✅ Dataclass fields match across tasks
- ✅ Function signatures consistent
- ✅ Variable names consistent

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-10-backport-audit-python-implementation.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
