# Backport PR Audit Tool - Python Rewrite Design

**Date:** 2026-04-10
**Branch:** backport-prs-notifier

## Context

Release managers need visibility into backport PR status before cutting releases to ensure all required fixes are included and properly tracked in Jira. The current implementation uses ~776 lines of bash across two scripts (`audit-backport-prs.sh` and `post-backport-audit-to-slack.sh`) with heavy string manipulation, associative arrays, and complex pipelines through `jq`, `curl`, and `gh` CLI.

**Problem:** Bash code is hard to follow, difficult to test, and error-prone for complex data transformations.

**Solution:** Rewrite the tool in Python (stdlib only) with clear class structure, type hints, and better error handling. The Slack posting will be handled by a GitHub Action, so Python only needs to generate the formatted JSON payload.

## Requirements

### Functional Requirements
- Detect release branches from git tags
- Fetch open backport PRs via `gh` CLI
- Resolve real authors (handle rhacs-bot, dependabot)
- Extract Jira keys from PR titles
- Validate Jira issues via REST API
- Find orphaned Jira issues (in fixVersion but no PR)
- Generate markdown report for GitHub step summary
- Generate Slack JSON blocks payload for posting

### Technical Constraints
- Python 3.9+ (per project's `.openshift-ci/.pylintrc`)
- **No pip dependencies** - stdlib only
- Single executable script (~600-800 lines)
- Call `gh` CLI via subprocess (already authenticated in CI)
- Call Jira REST API via `urllib.request`
- Compatible with existing pylint configuration
- Use Ruff for development linting only

### Non-Functional Requirements
- Fail fast on environment validation errors
- Partial results OK for individual item failures
- Always generate report even with incomplete data
- Clear error messages to stderr
- Exit codes: 0 (success), 1 (expected error), 2 (unexpected error)

## Architecture

### High-Level Design

```
┌─────────────────────────────────────────────────────────────┐
│              scripts/ci/backport-audit.py                    │
│                (Single executable script)                     │
└─────────────────────────────────────────────────────────────┘
                            │
                ┌───────────┼───────────┐
                ▼           ▼           ▼
         ┌──────────┐ ┌─────────┐ ┌──────────┐
         │ GitHub   │ │  Jira   │ │ Report   │
         │ Client   │ │ Client  │ │Generator │
         └──────────┘ └─────────┘ └──────────┘
              │            │            │
         (subprocess)  (urllib)    (templates)
              │            │            │
              ▼            ▼            ▼
         gh CLI      Jira REST    Markdown +
         JSON        API JSON     Slack JSON
```

### Data Flow

1. **Configuration** → Load from environment variables (`JIRA_USER`, `JIRA_TOKEN`, etc.)
2. **Git** → Detect release branches and versions from tags
3. **GitHub** → Fetch PRs, resolve authors, extract Jira keys
4. **Jira** → Validate issues, find orphaned issues
5. **Report Generation** → Transform data to markdown + Slack JSON
6. **Output** → Write files (`backport-audit-report.md`, `slack-payload.json`)

## Data Structures

### Core Dataclasses

```python
@dataclass
class PR:
    """Represents a pull request."""
    number: int
    title: str
    author: str           # Resolved (not rhacs-bot)
    base_ref: str         # e.g., release-4.10
    jira_keys: List[str]  # Extracted ROX-XXXXX
    body: str

@dataclass
class JiraIssue:
    """Represents a Jira issue."""
    key: str
    summary: str
    fix_versions: List[str]
    affected_versions: List[str]
    assignee: Optional[str]
    team: Optional[str]
    component: Optional[str]

@dataclass
class ReleaseBranch:
    """Represents a release branch."""
    name: str                    # release-4.10
    expected_version: str        # 4.10.3
    latest_tag: Optional[str]    # 4.10.2
```

### Client Classes

```python
class GitHubClient:
    """GitHub operations via gh CLI subprocess calls."""

    def fetch_prs(self, label: str, state: str = "open") -> List[Dict]
    def get_pr_details(self, pr_number: int) -> Dict
    def get_issue_events(self, pr_number: int) -> List[Dict]

class JiraClient:
    """Jira REST API client using urllib (no external libs)."""

    def __init__(self, user: str, token: str, base_url: str)
    def get_issue(self, issue_key: str) -> Optional[JiraIssue]
    def search_issues(self, jql: str) -> List[JiraIssue]

class ReportGenerator:
    """Generates markdown and Slack JSON outputs."""

    def __init__(self, slack_user_map: Dict[str, str])
    def generate_markdown(...) -> str
    def generate_slack_payload(...) -> Dict
```

## Key Implementation Details

### Author Resolution Logic

Handle three cases:
1. **rhacs-bot** → Extract original PR from body → Get real author
2. **dependabot (via rhacs-bot)** → Find who added backport label to original
3. **dependabot (direct)** → Find who added backport label

```python
def resolve_author(pr: Dict, gh_client: GitHubClient) -> str:
    author = pr['author']['login']

    if author == 'rhacs-bot':
        match = re.search(r'from #(\d+)', pr['body'])
        if match:
            original_pr = gh_client.get_pr_details(int(match.group(1)))
            author = original_pr['author']['login']
            if author == 'app/dependabot':
                author = find_backport_label_adder(original_pr['number'])

    elif author == 'app/dependabot':
        author = find_backport_label_adder(pr['number'])

    return author
```

### Jira Key Extraction

Simple regex pattern matching:
```python
def extract_jira_keys(title: str) -> List[str]:
    return sorted(set(re.findall(r'ROX-\d+', title)))
```

### Release Version Detection

From git tags:
```python
def detect_release_versions(branches: List[str]) -> Dict[str, ReleaseBranch]:
    # For each branch (release-4.10):
    # 1. Extract base version (4.10)
    # 2. Find latest tag: git tag | grep "^4.10" | sort -V | tail -1
    # 3. Calculate next patch: 4.10.2 → 4.10.3
```

### Slack Section Splitting

Split markdown into Slack-compatible sections (max 2800 chars):
```python
def split_slack_sections(text: str, max_chars: int = 2800) -> List[str]:
    # Split on lines, keep subsection headers together
    # Add "(continued)" headers when splitting
    # Return list of section strings
```

### Slack User Mapping

Convert `get-slack-user-id.sh` to Python dict:
```python
SLACK_USER_MAP: Dict[str, str] = {
    'janisz': 'U0218FUVDMJ',
    'porridge': 'U020XCUG2LA',
    # ... ~60 entries from bash script
}

def get_slack_mention(github_login: str) -> str:
    """
    Get Slack mention for GitHub user.

    Returns:
        - "<@SLACK_ID>" if mapping exists
        - "@github_login" if no mapping found
        - ":konflux:" if user is app/red-hat-konflux
    """
    if github_login == 'app/red-hat-konflux':
        return ':konflux:'

    slack_id = SLACK_USER_MAP.get(github_login)
    if slack_id:
        return f'<@{slack_id}>'
    return f'@{github_login}'
```

## Output Formats

### Markdown Report

```markdown
# Backport PR Audit Report

Generated: 2026-04-10 14:30:00 UTC

## release-4.10 (Expected: 4.10.3)

### PRs Missing Jira Reference (2)

- <@U0218FUVDMJ> #12345: Fix bug
- @dependabot #12346: Bump version

### Jira Issues with Missing Metadata (1)

- ROX-12345: :x: fixVersion, :white_check_mark: affectedVersion
  (Assignee: John Doe, Team: Platform, Component: Auth)
  (PRs: #12345)

### Orphaned Jira Issues (1)

- ROX-12348
```

### Slack JSON Payload

```json
{
  "channel": "C05AZF8T7GW",
  "blocks": [
    {"type": "header", "text": {"type": "plain_text", "text": "📋 Backport PR Audit Report"}},
    {"type": "section", "text": {"type": "mrkdwn", "text": "*Generated:* ...\n*Total PRs missing Jira:* 5"}},
    {"type": "divider"},
    {"type": "section", "text": {"type": "mrkdwn", "text": "*release-4.10 (Expected: 4.10.3)*\n\n..."}}
  ]
}
```

## Error Handling Strategy

### Philosophy
1. **Environment validation** - Fail fast (missing `JIRA_TOKEN`)
2. **Data fetching** - Retry with backoff, then fail
3. **Individual items** - Log warning, continue (partial results OK)
4. **Report generation** - Always succeed (even with incomplete data)

### Exception Hierarchy

```python
class BackportAuditError(Exception):
    """Base exception."""

class GitHubError(BackportAuditError):
    """GitHub API/CLI errors."""

class JiraError(BackportAuditError):
    """Jira API errors."""
```

### Error Handling Example

```python
def main():
    try:
        validate_environment()
        prs = fetch_prs_with_retry()
        jira_issues = fetch_jira_with_partial_failures()
        report = generate_report(prs, jira_issues)
    except BackportAuditError as e:
        print(f"ERROR: {e}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"UNEXPECTED ERROR: {e}", file=sys.stderr)
        traceback.print_exc()
        sys.exit(2)
```

## Configuration

### Environment Variables

```python
@dataclass
class Config:
    jira_user: str              # Required: JIRA_USER
    jira_token: str             # Required: JIRA_TOKEN
    jira_base_url: str = "redhat.atlassian.net"
    jira_project: str = "ROX"
    github_token: Optional[str] = None  # gh CLI handles auth
    output_dir: str = "."
    report_file: str = "backport-audit-report.md"
    slack_payload_file: str = "slack-payload.json"

    @classmethod
    def from_env(cls) -> 'Config':
        return cls(
            jira_user=os.environ['JIRA_USER'],
            jira_token=os.environ['JIRA_TOKEN'],
        )
```

### Command-Line Arguments

```bash
#!/usr/bin/env python3
# Usage: backport-audit.py [--branches BRANCHES] [--output-dir DIR]

Arguments:
  --branches BRANCHES    Comma-separated or "all" (default: all)
  --output-dir DIR       Output directory (default: .)
  --github-run-url URL   GitHub Actions run URL for Slack link
```

## Type Hints & Code Quality

### Type Annotations

Full type hints throughout:
```python
def fetch_prs(self, label: str, state: str = "open") -> List[Dict[str, any]]:
    """Fetch PRs using gh CLI."""

def get_issue(self, issue_key: str) -> Optional[JiraIssue]:
    """Fetch Jira issue via REST API."""
```

### Linting

**Development:** Use Ruff (fast, modern)
```bash
# Install: pip install ruff (in venv for development)
ruff check scripts/ci/backport-audit.py
ruff format scripts/ci/backport-audit.py
```

**CI Compatibility:** Script follows existing `.openshift-ci/.pylintrc` rules:
- Max line length: 120 chars
- snake_case functions/variables
- PascalCase classes
- UPPER_CASE constants
- 4-space indentation
- Max 6 args per function
- Python 3.9+ compatible

### Script Header

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
```

## Critical Files

**Modified:**
- `.github/workflows/audit-backport-prs.yml` - Update to call Python script
- `scripts/ci/backport-audit.py` - New Python implementation

**Replaced (can be deleted after verification):**
- `scripts/ci/audit-backport-prs.sh` - Old bash implementation
- `scripts/ci/post-backport-audit-to-slack.sh` - Slack posting (now in Action)
- `scripts/ci/get-slack-user-id.sh` - Converted to Python dict

**Unchanged:**
- `scripts/ci/lib.sh` - Keep for other scripts

## Workflow Integration

Update `.github/workflows/audit-backport-prs.yml`:

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

- name: Post to Slack
  if: success()
  uses: slackapi/slack-github-action@v1
  with:
    payload-file-path: slack-payload.json
  env:
    SLACK_BOT_TOKEN: ${{ secrets.SLACK_BOT_TOKEN }}
```

**Key Changes:**
- Call `backport-audit.py` instead of `audit-backport-prs.sh`
- Pass GitHub run URL as command-line arg
- Use existing Slack action (reads `slack-payload.json`)
- Remove `post-backport-audit-to-slack.sh` step

## Verification

### Unit Testing Approach

While we won't write formal unit tests initially, the class structure enables easy testing:

```python
# Manual verification script
if __name__ == "__main__":
    # Test GitHub client
    gh = GitHubClient()
    prs = gh.fetch_prs("backport", "open")
    print(f"Found {len(prs)} PRs")

    # Test Jira client
    jira = JiraClient(os.environ['JIRA_USER'], os.environ['JIRA_TOKEN'])
    issue = jira.get_issue("ROX-12345")
    print(f"Issue: {issue}")
```

### Integration Testing

1. **Local run:**
   ```bash
   export JIRA_USER="user@example.com"
   export JIRA_TOKEN="secret"
   ./scripts/ci/backport-audit.py --branches all
   ```

2. **Verify outputs:**
   - `backport-audit-report.md` exists and is readable
   - `slack-payload.json` is valid JSON
   - Slack blocks structure matches expected format

3. **Test in GitHub Actions:**
   - Push to branch
   - Workflow runs Python script
   - Step summary shows markdown report
   - Slack action receives valid JSON payload

### Comparison Testing

Run both bash and Python versions, compare outputs:
```bash
# Bash version
./scripts/ci/audit-backport-prs.sh --branches all
mv backport-audit-report.md bash-report.md

# Python version
./scripts/ci/backport-audit.py --branches all
mv backport-audit-report.md python-report.md

# Compare
diff bash-report.md python-report.md
```

## Migration Strategy

### Phase 1: Parallel Implementation
- Keep bash scripts
- Add Python script
- Add flag to workflow to choose implementation
- Default to bash for safety

### Phase 2: Validation Period (1-2 weeks)
- Run both in CI, compare outputs
- Monitor for discrepancies
- Fix any Python bugs found
- Build confidence

### Phase 3: Cutover
- Switch workflow default to Python
- Keep bash as fallback for 1 release cycle
- Monitor Slack messages for issues

### Phase 4: Cleanup
- Remove bash scripts
- Remove fallback logic from workflow
- Update documentation

## Risks & Mitigations

### Risk: Urllib complexity vs requests library
**Mitigation:** Jira API calls are simple (GET with Basic Auth). urllib.request is sufficient.

### Risk: Slack formatting differences
**Mitigation:** Preserve exact markdown format, test Slack block rendering manually.

### Risk: Author resolution edge cases
**Mitigation:** Log warnings for unresolved cases, include fallback to original author.

### Risk: Git tag parsing inconsistencies
**Mitigation:** Use same regex patterns as bash, add validation for version format.

### Risk: Missing environment variables in CI
**Mitigation:** Fail fast with clear error message, validate before any API calls.

## Future Enhancements (Out of Scope)

- Add proper unit tests with `unittest`
- Support for multiple Jira projects
- Cache Jira results to reduce API calls
- Progress indicators for long-running operations
- JSON schema validation for Slack payload
- Configurable retry logic for API calls
- Support for private/enterprise GitHub instances
