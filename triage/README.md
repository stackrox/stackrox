# StackRox CI Failure Analysis with Claude Code

This directory contains tools and documentation for debugging StackRox CI failures using Claude Code with MCP (Model Context Protocol) integrations.

## 🚀 Quick Setup

### Prerequisites
1. **Claude Code CLI** [installed and configured](https://source.redhat.com/departments/it/itx/document_management_and_collaboration/claude_code)
2. **MCP Atlassian Integration** from https://github.com/sooperset/mcp-atlassian, see also below
3. **JIRA Access** to Red Hat JIRA instance (issues.redhat.com)
4. **Google Cloud SDK** (`gsutil`) for artifact downloads

### MCP Atlassian Setup
```bash
# Configure MCP server in your Claude Code settings
# Edit ~/.claude.json to add the following MCP server configuration, or use `claude mcp add`:
      "mcpServers": {
        "mcp-atlassian": {
          "command": "docker",
          "args": [
            "run",
            "-i",
            "--rm",
            "-e",
            "CONFLUENCE_URL",
            "-e",
            "CONFLUENCE_API_TOKEN",
            "-e",
            "JIRA_URL",
            "-e",
            "JIRA_PERSONAL_TOKEN",
            "ghcr.io/sooperset/mcp-atlassian:latest"
          ],
          "env": {
            "CONFLUENCE_URL": "https://spaces.redhat.com/",
            "CONFLUENCE_API_TOKEN": "Generate at: https://spaces.redhat.com/plugins/personalaccesstokens/usertokens.action",
            "JIRA_URL": "https://issues.redhat.com",
            "JIRA_PERSONAL_TOKEN": "Generate at: https://issues.redhat.com/secure/ViewProfile.jspa?selectedTab=com.atlassian.pats.pats-plugin:jira-user-personal-access-tokens"
          }
        },
    }
```

## 🎯 Primary Use Case: CI Failure Debugging

### Step 1: Start Investigation
```bash
claude --append-system-prompt "$(cat prompt.md)" --allowed-tools "Read,Grep,Glob,Bash,LS,TodoWrite" "triage ROX-21719"
```

### Step 2: Claude Code Workflow
Claude Code will automatically perform:

1. **JIRA Investigation** (via MCP Atlassian)
   - Retrieve full issue details using `mcp__mcp-atlassian__jira_get_issue`
   - Search for related issues with `mcp__mcp-atlassian__jira_search`
   - Check latest comments for recent build IDs and failure patterns

2. **Artifact Analysis**
   - Download Prow artifacts to `triage/ROX-XXXXX-analysis/`
   - Determine correct GCS bucket (`origin-ci-test` vs `test-platform-results`)
   - Analyze build logs, test outputs, and failure patterns

3. **Root Cause Analysis**
   - Correlate test failures with infrastructure issues
   - Map errors to source code locations
   - Identify team ownership for escalation

4. **Documentation**
   - Add findings to JIRA tickets via `mcp__mcp-atlassian__jira_add_comment`
   - Update this directory with investigation artifacts

## 📁 Directory Structure

```
artifacts/
├── README.md                 # This file - usage instructions
├── CLAUDE.md                 # Comprehensive debugging guide (1095 lines)
├── .gitignore               # Excludes analysis artifacts except docs
├── ROX-XXXXX-analysis/      # Downloaded Prow artifacts (git-ignored)
│   ├── build-log.txt
│   ├── artifacts/
│   └── ...
└── prompt.md               # Custom prompts for specific scenarios (if needed)
```

## 🧪 Test Framework Coverage

### Groovy/Spock Tests (`qa-tests-backend/`)
**Job Pattern**: `*-qa-e2e-tests`
```bash
# NetworkFlowTest failures:
"triage ROX-21719 - focus on NetworkFlowTest timeout patterns"

# Other common Groovy test patterns:
"triage ROX-XXXXX - analyze AuthServiceTest authentication failures"
"triage ROX-XXXXX - investigate PolicyConfigurationTest policy evaluation errors"
"triage ROX-XXXXX - debug ImageScanningTest vulnerability detection issues"

# Artifact locations:
# - Test logs: spec-logs/{TestName}.log
# - JUnit results: junit-part-1-tests/, junit-part-2-tests/
# - Source code: qa-tests-backend/src/test/groovy/{TestName}.groovy
```

### Go Integration Tests (`tests/`)
**Job Patterns**: `*-compliance-e2e-tests`, `*-compatibility-tests`, `*-nongroovy-*`
```bash
# Compliance tests:
"triage ROX-XXXXX - analyze compliance operator v2 test failures"

# Compatibility tests:
"triage ROX-XXXXX - investigate pods_test.go container runtime compatibility"

# NonGroovy tests:
"triage ROX-XXXXX - debug central gRPC connection failures"

# Artifact locations:
# - Test logs: junit-{test-type}-results/{test-type}-results/test.log
# - JUnit results: junit-compliance-v2-tests-results/, junit-compatibility-test-*/
# - Source code: tests/compliance_operator_v2_test.go, tests/pods_test.go, tests/common.go
# - Error traces: Exact file:line locations (e.g., tests/compliance_operator_v2_test.go:211)
```

### Platform-Specific Investigation
```bash
# Infrastructure issues:
"triage ROX-30001 - check for PowerVS/IBM Cloud platform-specific issues"

# Cross-platform comparison:
"analyze ROX-XXXXX across multiple test platforms (AKS, OSD, ARO)"
```

## 📚 Additional Resources

- **Complete Debugging Guide**: See `CLAUDE.md` for comprehensive debugging reference
- **MCP Atlassian Docs**: https://github.com/sooperset/mcp-atlassian
- **StackRox CI Overview**: See `AGENTS.md` in repository root
- **Groovy Test Framework**: Check `qa-tests-backend/` directory structure
- **Nongroovy Test Framework**: Check `tests/` directory structure
