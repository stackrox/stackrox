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

post_header() {
    local timestamp
    local total_prs
    local total_jira

    timestamp=$(grep "^Generated:" "$REPORT_FILE" | sed 's/Generated: //' || echo "Unknown")
    total_prs=$(grep -c "^- <@\|^- :konflux:" "$REPORT_FILE" 2>/dev/null || echo "0")
    total_jira=$(grep -c "^- \[ROX-" "$REPORT_FILE" 2>/dev/null || echo "0")

    info "Posting header: $total_prs PRs, $total_jira Jira issues"

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
}

post_release_sections() {
    info "Posting release sections"

    # Extract each release section and post separately
    awk '
        BEGIN {
            in_release = 0
            content = ""
            release_name = ""
        }
        /^## release-/ {
            if (in_release && content != "") {
                # Output previous release
                print "RELEASE_START:" release_name
                print content
                print "RELEASE_END"
            }
            in_release = 1
            release_name = $0
            gsub(/^## /, "", release_name)
            content = ""
            next
        }
        /^### / {
            if (in_release) {
                section_header = $0
                gsub(/^### /, "", section_header)
                content = content "\n*" section_header "*\n"
            }
            next
        }
        /^- / {
            if (in_release) {
                content = content $0 "\n"
            }
        }
        END {
            if (in_release && content != "") {
                print "RELEASE_START:" release_name
                print content
                print "RELEASE_END"
            }
        }
    ' "$REPORT_FILE" | {
        local release_name=""
        local content=""

        while IFS= read -r line; do
            if [[ "$line" =~ ^RELEASE_START: ]]; then
                release_name="${line#RELEASE_START:}"
                content=""
            elif [[ "$line" == "RELEASE_END" ]]; then
                if [[ -n "$content" ]]; then
                    info "  Posting: $release_name"

                    # Build message with release header and content
                    local message="*${release_name}*\n\n${content}"

                    jq -n \
                        --arg channel "$SLACK_CHANNEL" \
                        --arg text "$message" \
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
                            },
                            {
                              "type": "divider"
                            }
                          ]
                        }' | post_to_slack "$(cat -)"

                    # Small delay to avoid rate limiting
                    sleep 1
                fi
                release_name=""
                content=""
            else
                content="${content}${line}\n"
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
    post_header
    post_release_sections

    info "✅ Posted to Slack channel: $SLACK_CHANNEL"
}

main "$@"
