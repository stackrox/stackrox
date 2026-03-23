#!/usr/bin/env bash

# Create CI triage slack reminder.

set -euo pipefail

total_issues_in_filter() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: total_issues_in_filter <filter>"
    fi

    local filter="$1"
    local filter_jql
    filter_jql=$(curl -sSfl \
       -u "${JIRA_USER}:${JIRA_TOKEN}" \
       -H "Content-Type: application/json" \
       "https://redhat.atlassian.net/rest/api/3/filter/${filter}" | jq -r '.jql')

    curl -sSfl \
       -u "${JIRA_USER}:${JIRA_TOKEN}" \
       -H "Content-Type: application/json" \
       --get \
       --data-urlencode "jql=${filter_jql}" \
       --data-urlencode "maxResults=5000" \
       "https://redhat.atlassian.net/rest/api/3/search/jql" | jq '.issues | length'
}

slack_triage_report() {
    local curr_filter=103399
    local prev_filter=95004

    local curr
    curr=$(total_issues_in_filter $curr_filter)
    local prev
    prev=$(total_issues_in_filter $prev_filter)

    if [[ "$curr" -eq 0 && "$prev" -eq 0 ]]; then
        echo "No issues to report (curr=0, prev=0), skipping message"
        return
    fi

    local line
    if [[ "$prev" -eq 0 ]]; then
        line="<!subteam^S04SU9AHJ4C> There are ${curr} untriaged issues"
    else
        line="<!subteam^S04SU9AHJ4C> There are ${curr} untriaged issues (not including ${prev} leftovers from previous duty)"
    fi

    local body
    # shellcheck disable=SC2016
    body='{
    "blocks": [{
        "type": "section",
        "text": {
            "type": "mrkdwn",
            "text": "'"${line}"'"
        },
        "accessory": {
            "type": "button",
            "text": {
                "type": "plain_text",
                "text": "Triage them 🔥",
                "emoji": true
            },
            "url": "https://redhat.atlassian.net/jira/dashboards/22941"
        }
    }]}'
    echo "Posting '$line' to slack"
    jq -n "$body" | curl -sSfl -d @- -H 'Content-Type: application/json' "$SLACK_WEBHOOK"
}

slack_triage_report
