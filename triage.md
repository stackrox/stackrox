# StackRox CI Triage Setup Guide

This guide explains how to set up Claude Code for automated StackRox CI failure investigation and triage.

## Prerequisites

To enable automated CI triage capabilities, you need to configure the following MCP (Model Context Protocol) servers in Claude Code:

### Required Environment Variables

Set up the following environment variables in your shell or `.bashrc`/`.zshrc`:

```bash
export GITHUB_TOKEN="your_github_personal_access_token"
export CONFLUENCE_API_TOKEN="your_confluence_api_token"
export JIRA_PERSONAL_TOKEN="your_jira_personal_access_token"
```

### MCP Server Configuration

Create or update your `.mcp.json` file in your project root with the following configuration:

```json
{
  "mcpServers": {
    "prowject": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "-e",
        "MCP_TRANSPORT",
        "ghcr.io/janisz/prowject:latest"
      ],
      "env": {
        "MCP_TRANSPORT": "stdio"
      }
    },
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
        "CONFLUENCE_API_TOKEN": "${CONFLUENCE_API_TOKEN}",
        "JIRA_URL": "https://issues.redhat.com",
        "JIRA_PERSONAL_TOKEN": "${JIRA_PERSONAL_TOKEN}"
      }
    },
    "github": {
      "type": "http",
      "url": "https://api.githubcopilot.com/mcp/",
      "headers": {
        "Authorization": "Bearer ${GITHUB_TOKEN}"
      }
    }
  }
}
```

## Agent Configuration

The `stackrox-ci-failure-investigator` agent is automatically configured in `.claude/agents/` and provides:

- **Automated JIRA issue analysis** - Fetches ROX-XXXXX issues and analyzes comments for build IDs
- **Prow build log retrieval** - Downloads CI artifacts and logs for failure analysis
- **Root cause investigation** - Correlates test failures with service logs and source code
- **Team assignment** - Routes issues to appropriate teams based on failure patterns

## Usage

Once configured, you can trigger automated triage by providing one of the following as a claude code prompt:

1. **JIRA issue keys**: `ROX-28636`, `ROX-30813`
2. **Prow build IDs**: `1963388448995807232`
3. **Error logs or stack traces** directly
4. **CI failure URLs** from GitHub or Prow

## Capabilities

### Automated Investigation
- Downloads complete CI artifacts using `gsutil`
- Analyzes service logs (central, scanner, sensor, admission-control)
- Correlates test timestamps with failure events
- Searches source code for error origins
- Identifies flaky tests vs new failures

## Manual Fallback

If MCP tools are unavailable, the agent provides manual investigation guidance using:
- Direct JIRA dashboard access
- `gsutil` commands for artifact download
- Log analysis patterns and team assignment rules
