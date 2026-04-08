#!/usr/bin/env bash

# Post backport audit report to Slack in structured sections

set -euo pipefail

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "$SCRIPTS_ROOT/scripts/ci/lib.sh"

# Configuration
REPORT_FILE="${1:-backport-audit-report.md}"
SLACK_CHANNEL="${2:-}"
GITHUB_RUN_URL="${3:-}"

usage() {
    cat <<EOF
Usage: $0 <report_file> <slack_channel> <github_run_url>

Post backport audit report to Slack in sections.

Arguments:
    report_file       Path to the markdown report file
    slack_channel     Slack channel ID (e.g., C05AZF8T7GW)
    github_run_url    GitHub Actions run URL

Environment Variables (required):
    SLACK_BOT_TOKEN   Slack bot token

Example:
    $0 backport-audit-report.md C05AZF8T7GW https://github.com/stackrox/stackrox/actions/runs/123
EOF
}

validate_environment() {
    if [[ -z "$SLACK_CHANNEL" ]]; then
        die "Slack channel is required"
    fi

    require_environment "SLACK_BOT_TOKEN"

    if [[ ! -f "$REPORT_FILE" ]]; then
        die "Report file not found: $REPORT_FILE"
    fi
}

post_to_slack() {
    local payload="$1"

    curl -s -X POST https://slack.com/api/chat.postMessage \
        -H "Authorization: Bearer ${SLACK_BOT_TOKEN}" \
        -H "Content-Type: application/json" \
        -d "$payload"
}

build_full_message() {
    local timestamp
    local total_prs
    local total_jira

    timestamp=$(grep "^Generated:" "$REPORT_FILE" | sed 's/Generated: //' || echo "Unknown")
    total_prs=$(grep -c "^- <@\|^- :konflux:" "$REPORT_FILE" 2>/dev/null || echo "0")
    total_jira=$(grep -c "^- <" "$REPORT_FILE" | grep -c "ROX-" || echo "0")

    # Build header
    local message
    message=$(cat <<EOF
*📋 Backport PR Audit Report*

*Generated:* ${timestamp}
*Total PRs missing Jira:* ${total_prs}
*Total Jira issues with missing metadata:* ${total_jira}

<${GITHUB_RUN_URL}|View full report in GitHub Actions>

───────────────────────────────────────
EOF
)

    # Extract report content (skip first two lines: title and timestamp)
    local report_content
    report_content=$(tail -n +4 "$REPORT_FILE")

    message="${message}
${report_content}"

    echo "$message"
}


post_message() {
    info "Building and posting complete report"

    local message
    message=$(build_full_message)

    jq -n \
        --arg channel "$SLACK_CHANNEL" \
        --arg text "$message" \
        '{
          "channel": $channel,
          "text": "Backport PR Audit Report",
          "blocks": [
            {
              "type": "section",
              "text": {
                "type": "mrkdwn",
                "text": $text
              }
            }
          ]
        }' | post_to_slack "$(cat -)"
}

main() {
    if [[ "${1:-}" == "--help" ]]; then
        usage
        exit 0
    fi

    validate_environment
    post_message

    info "✅ Posted to Slack channel: $SLACK_CHANNEL"
}

main "$@"
