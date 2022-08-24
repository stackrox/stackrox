#!/usr/bin/env bash
#
# Checks if there are open issues for the given release
# and adds comments to such issues.
#
set -euo pipefail

PROJECTS="$1"
RELEASE="$2"
PATCH="$3"
RELEASE_PATCH="$4"

check_not_empty \
    PROJECTS \
    RELEASE \
    PATCH \
    RELEASE_PATCH \
    \
    JIRA_TOKEN \
    DRY_RUN

# TODO: Jira returns 400 if requested fixVersion does not exist. That means
# the named release must exist on Jira, which is not given.
JQL="project IN ($PROJECTS) \
AND fixVersion IN (\"$RELEASE.$PATCH\") \
AND status != CLOSED \
AND Component != Documentation \
AND type != Epic \
ORDER BY assignee"

get_issues() {
    curl --fail -sSL --get --data-urlencode "jql=$JQL" \
        -H "Authorization: Bearer $JIRA_TOKEN" \
        -H "Accept: application/json" \
        "https://issues.redhat.com/rest/api/2/search"
}

comment_issue() {
    curl --fail -sSL -X POST \
        -H "Authorization: Bearer $JIRA_TOKEN" \
        -H "Content-Type: application/json" \
        --data "{\"body\": \
\"Release $RELEASE_PATCH is ongoing. \
Please update the status of this issue and notify the release engineer.\"}" \
        "https://issues.redhat.com/rest/api/2/issue/$1/comment"
}

get_issues_summary() {
    read -r GH_MD_FORMAT_LINE <<EOF
* [\(.key)](https://issues.redhat.com/browse/\(.key)): \
**\(.fields.assignee.displayName // "unassigned")** \
(\(.fields.status.name)) — _\(.fields.summary | gsub (" +$";""))_
EOF
    get_issues | jq -r ".issues[] | \"$GH_MD_FORMAT_LINE\"" | sort
}

get_open_issues() {
    get_issues | jq -r '.issues[] | "\(.key) - \(.fields.assignee.displayName)"' | sort
}

ISSUES=$(get_issues_summary)

if [ -z "$ISSUES" ]; then
    gh_summary "All issues for Jira release $RELEASE_PATCH are closed."
    exit 0
fi

gh_summary <<EOF
:red_circle: The following Jira issues are still open for release $RELEASE_PATCH:

$ISSUES

:arrow_right: Contact the assignees to clarify the status.
EOF

gh_log error "There are non-closed Jira issues for version $RELEASE_PATCH."

OPEN_ISSUES=$(get_open_issues)

echo "Open issues:"
echo "$OPEN_ISSUES"

if [ "$DRY_RUN" = "false" ]; then
    while read -r KEY; do
        comment_issue "$KEY"
    done <<<"$OPEN_ISSUES"
else
    exit 0
fi

exit 1
