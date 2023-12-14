#!/usr/bin/env bash

# Create CI triage slack reminder.

set -euo pipefail

total_issues_in_filter() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: total_issues_in_filter <filter>"
    fi

    local filter="$1"
    curl -sSfl \
       -H "Authorization: Bearer $JIRA_TOKEN" \
       -H "Content-Type: application/json" \
       "https://issues.redhat.com/rest/api/latest/search?jql=filter=$filter&maxResults=0" | jq '.total'
}

slack_triage_report() {
    local curr_filter=12388299
    local prev_filter=12388044

    local curr=$(total_issues_in_filter $curr_filter)
    local prev=$(total_issues_in_filter $prev_filter)

    local line="There are \`"${curr}+${prev}"\` *NOT* triaged issues from current + previous duty"
    local body
    # shellcheck disable=SC2016
    body='
    {
        "blocks": [
            {
                "type": "header",
                "text": {
                    "type": "plain_text",
                    "text": "Triage report"
                }
            },
            {
                "type": "section",
                "text": {
                    "type": "mrkdwn",
                    "text": "'"${line}"'"
                }
            }
        ]
    }'
    echo "Posting '$line' to slack"
    jq -n "$body" | curl -XPOST -d @- -H 'Content-Type: application/json' "$SLACK_CI_INTEGRATION_TESTING_WEBHOOK"
}

slack_triage_report