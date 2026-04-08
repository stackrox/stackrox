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

post_message() {
    info "Building and posting report in sections"

    local timestamp
    local total_prs
    local total_jira

    timestamp=$(grep "^Generated:" "$REPORT_FILE" | sed 's/Generated: //' || echo "Unknown")
    total_prs=$(grep -c "^- <@\|^- :konflux:" "$REPORT_FILE" 2>/dev/null || echo "0")
    total_jira=$(grep -c "^- <" "$REPORT_FILE" | grep -c "ROX-" || echo "0")

    # Post header
    jq -n \
        --arg channel "$SLACK_CHANNEL" \
        --arg timestamp "$timestamp" \
        --arg total_prs "$total_prs" \
        --arg total_jira "$total_jira" \
        --arg url "$GITHUB_RUN_URL" \
        '{
          "channel": $channel,
          "text": "Backport PR Audit Report",
          "blocks": [
            {
              "type": "header",
              "text": {
                "type": "plain_text",
                "text": "📋 Backport PR Audit Report"
              }
            },
            {
              "type": "section",
              "text": {
                "type": "mrkdwn",
                "text": ("*Generated:* " + $timestamp + "\n*Total PRs missing Jira:* " + $total_prs + "\n*Total Jira issues with missing metadata:* " + $total_jira)
              }
            },
            {
              "type": "section",
              "text": {
                "type": "mrkdwn",
                "text": ("<" + $url + "|View full report in GitHub Actions>")
              }
            },
            {
              "type": "divider"
            }
          ]
        }' | post_to_slack "$(cat -)"

    # Parse and post each release section
    awk '
        BEGIN {
            in_release = 0
            section = ""
            current_subsection = ""
        }
        /^## release-/ {
            # Post previous section if exists
            if (in_release && section != "") {
                print "SECTION_START"
                print section
                print "SECTION_END"
            }
            in_release = 1
            release_name = $0
            gsub(/^## /, "", release_name)
            section = "*" release_name "*\n"
            next
        }
        /^### / {
            if (in_release) {
                subsection_header = $0
                gsub(/^### /, "", subsection_header)
                current_subsection = subsection_header
                section = section "\n*" subsection_header "*\n"
            }
            next
        }
        /^- / {
            if (in_release) {
                # Check if section is getting too large (max ~2800 chars to be safe)
                if (length(section) > 2800) {
                    # Post current section
                    print "SECTION_START"
                    print section
                    print "SECTION_END"
                    # Start new section with continuation
                    section = "*" release_name " (continued)*\n\n*" current_subsection " (continued)*\n"
                }
                section = section $0 "\n"
            }
        }
        END {
            if (in_release && section != "") {
                print "SECTION_START"
                print section
                print "SECTION_END"
            }
        }
    ' "$REPORT_FILE" | {
        while IFS= read -r line; do
            if [[ "$line" == "SECTION_START" ]]; then
                content=""
            elif [[ "$line" == "SECTION_END" ]]; then
                if [[ -n "$content" ]]; then
                    jq -n \
                        --arg channel "$SLACK_CHANNEL" \
                        --arg text "$content" \
                        '{
                          "channel": $channel,
                          "text": "Release section",
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

                    # Small delay to avoid rate limiting
                    sleep 1
                fi
            else
                if [[ -z "$content" ]]; then
                    content="$line"
                else
                    content="$content
$line"
                fi
            fi
        done
    }
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
