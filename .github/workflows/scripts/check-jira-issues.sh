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

echo check_not_empty \
    PROJECTS \
    RELEASE \
    PATCH \
    RELEASE_PATCH \
    \
    JIRA_TOKEN \
    DRY_RUN

# TODO: Jira returns 400 if requested fixVersion does not exist. That means
# the named release must exist on Jira, which is not given.
JQL_OPEN_ISSUES="project IN ($PROJECTS) \
AND fixVersion = \"$RELEASE.$PATCH\" \
AND statusCategory != done \
AND (Component IS EMPTY or Component NOT IN (Documentation, \"ACS Cloud Service\")) \
AND type NOT IN (Epic, \"Feature Request\") \
ORDER BY assignee"

JQL_CLOSED_WITH_PR="project IN ($PROJECTS) \
AND fixVersion = \"$RELEASE.$PATCH\" \
AND statusCategory = done \
AND (Component IS EMPTY or Component NOT IN (Documentation, \"ACS Cloud Service\")) \
AND issue.property[development].openprs > 0 \
ORDER BY assignee"

get_issues() {
    curl --fail -sSL --get --data-urlencode "jql=$1" \
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
(\(.fields.issuetype.name), \(.fields.status.name)) — _\(.fields.summary | gsub (" +$";""))_
EOF
    get_issues "$1" | jq -r ".issues[] | \"$GH_MD_FORMAT_LINE\"" | sort
}

get_open_issues() {
    get_issues "$1" | jq -r '.issues[] | "\(.key) - \(.fields.assignee.displayName // "unassigned")"' | sort
}

CLOSED_WITH_PR=$(get_issues_summary "$JQL_CLOSED_WITH_PR")

if [ -n "$CLOSED_WITH_PR" ]; then
    gh_summary <<EOF
:warning: The following Jira issues have been marked complete, but have open PRs:

$CLOSED_WITH_PR

EOF
fi

OPEN_ISSUES=$(get_issues_summary "$JQL_OPEN_ISSUES")

if [ -z "$OPEN_ISSUES" ]; then
    gh_summary "All issues for Jira release $RELEASE_PATCH are closed."
    exit 0
fi

gh_summary <<EOF
:red_circle: The following Jira issues are still open for release $RELEASE_PATCH:

$OPEN_ISSUES

:arrow_right: Contact the assignees to clarify the status.
EOF

gh_log error "There are non-closed Jira issues for version $RELEASE_PATCH."

CLOSED_WITH_PR=$(get_open_issues "$JQL_CLOSED_WITH_PR")
OPEN_ISSUES=$(get_open_issues "$JQL_OPEN_ISSUES")

echo "Completed issues with open PR:"
echo "$CLOSED_WITH_PR"
echo
echo "Open issues:"
echo "$OPEN_ISSUES"

REPO_NAME=$(gh repo view --json nameWithOwner --jq .nameWithOwner)

if [ "$DRY_RUN" = "false" ] && [ "$REPO_NAME" = "stackrox/stackrox" ]; then
    while read -r KEY; do
        comment_issue "$KEY"
    done < <(cut -d " " -f 1 <<< "$OPEN_ISSUES")
else
    exit 0
fi

exit 1
