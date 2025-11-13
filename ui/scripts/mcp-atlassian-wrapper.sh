#!/bin/bash
# Wrapper script for MCP Atlassian server
# Passes environment variables to the Docker container

set -e

# Check for .env file in the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ENV_FILE="$PROJECT_ROOT/.env"

if [[ -f "$ENV_FILE" ]]; then
  source "$ENV_FILE"
fi

# Check if required environment variables are set
if [[ -z "$JIRA_USERNAME" ]]; then
  cat >&2 << 'EOF'
Error: JIRA_USERNAME environment variable is not set

You can set it in one of two ways:

1. In your .env file in the project root:
   JIRA_USERNAME=<your-username>
   JIRA_PERSONAL_TOKEN=<your-token>

2. In your shell profile (.bashrc, .zshrc, etc.):
   export JIRA_USERNAME=<your-username>
   export JIRA_PERSONAL_TOKEN=<your-token>
EOF
  exit 1
fi

if [[ -z "$JIRA_PERSONAL_TOKEN" ]]; then
  cat >&2 << 'EOF'
Error: JIRA_PERSONAL_TOKEN environment variable is not set

To generate a personal access token, visit:
  https://issues.redhat.com/secure/ViewProfile.jspa?currentTab=atlassian_token

You can set it in one of two ways:

1. In your .env file in the project root:
   JIRA_USERNAME=<your-username>
   JIRA_PERSONAL_TOKEN=<your-token>

2. In your shell profile (.bashrc, .zshrc, etc.):
   export JIRA_USERNAME=<your-username>
   export JIRA_PERSONAL_TOKEN=<your-token>
EOF
  exit 1
fi

docker run --rm -i \
  -e "JIRA_URL=https://issues.redhat.com" \
  -e "JIRA_USERNAME=$JIRA_USERNAME" \
  -e "JIRA_PERSONAL_TOKEN=$JIRA_PERSONAL_TOKEN" \
  -e "JIRA_SSL_VERIFY=true" \
  -v "${HOME}/.mcp-atlassian:/home/app/.mcp-atlassian" \
  ghcr.io/sooperset/mcp-atlassian:latest
