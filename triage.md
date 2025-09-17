# StackRox CI Triage Setup Guide

This guide explains how to set up Claude Code for automated StackRox CI failure investigation and triage.

## Prerequisites

To enable automated CI triage capabilities, you need to configure the following MCP (Model Context Protocol) servers in Claude Code:

- [mcp-atllasian](https://github.com/sooperset/mcp-atlassian)
- [Github MCP](https://github.com/github/github-mcp-server)
- [PROW MCP](https://github.com/redhat-community-ai-tools/prowject)

## Agent Configuration

The `stackrox-ci-failure-investigator` agent is automatically configured in `.claude/agents/` and provides:

- **Automated JIRA issue analysis** - Fetches ROX-XXXXX issues and analyzes comments for build IDs
- **Prow build log retrieval** - Downloads CI artifacts and logs for failure analysis
- **Root cause investigation** - Correlates test failures with service logs and source code
- **Team assignment** - Routes issues to appropriate teams based on failure patterns

## Usage

### Slash Command

You can use the `/triage` command for automated CI failure investigation:

```
/triage ROX-12345
```

This command automatically triggers the `stackrox-ci-failure-investigator` agent to:
- Fetch the JIRA issue details
- Analyze comments for build IDs and failure information
- Download relevant CI artifacts and logs
- Perform comprehensive root cause analysis
- Provide team assignment recommendations
- Suggest permanent fixes

### Manual Triggers

You can also trigger automated triage by providing one of the following as a claude code prompt:

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

## Additional Resources

For comprehensive triage procedures and ProdSec JIRA resolution, see:
- [How to triage and resolve ProdSec Jiras](https://spaces.redhat.com/spaces/StackRox/pages/459707157/How+to+triage+and+resolve+ProdSec+Jiras)
- [CI failures](https://spaces.redhat.com/spaces/StackRox/pages/580716357/CI+failures)
- [Team Alignments](https://spaces.redhat.com/spaces/StackRox/pages/256862805/Team+Alignments)
- [Test Flake Build Failure Process Proposal](https://spaces.redhat.com/spaces/StackRox/pages/259780495/Test+Flake+Build+Failure+Process+Proposal)
