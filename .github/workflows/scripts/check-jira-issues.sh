#!/usr/bin/env bash
#
# Checks if there are open issues for the given release
# and adds comments to such issues.
#
set -euo pipefail

PROJECTS="$1"
RELEASE="$2"
PATCH="$3"
NAMED_RELEASE_PATCH="$4"

check_not_empty \
    PROJECTS \
    RELEASE \
    PATCH \
    NAMED_RELEASE_PATCH \
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

comment_on_issues_list() {
    ISSUES=$1
    while read -r KEY; do
        comment_on_single_issue "$KEY"
    done < <(cut -d " " -f 1 <<< "$ISSUES")
}

comment_on_single_issue() {
    curl --fail --output /dev/null -sSL -X POST \
        -H "Authorization: Bearer $JIRA_TOKEN" \
        -H "Content-Type: application/json" \
        --data "{\"body\": \
\"Release $NAMED_RELEASE_PATCH is ongoing. \
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
    gh_summary "All issues for Jira release $NAMED_RELEASE_PATCH are closed."
    exit 0
fi

gh_summary <<EOF
:red_circle: The following Jira issues are still open for release $NAMED_RELEASE_PATCH:

$OPEN_ISSUES

:arrow_right: Contact the assignees to clarify the status.
EOF

gh_log error "There are non-closed Jira issues for version $NAMED_RELEASE_PATCH."

CLOSED_WITH_PR=$(get_open_issues "$JQL_CLOSED_WITH_PR")
OPEN_ISSUES=$(get_open_issues "$JQL_OPEN_ISSUES")

echo "Completed issues with open PR:"
echo "$CLOSED_WITH_PR"
echo
echo "Open issues:"
echo "$OPEN_ISSUES"

if [ "$DRY_RUN" = "false" ]; then
    if [ "$GITHUB_REPOSITORY" = "stackrox/stackrox" ] || { [ "$GITHUB_REPOSITORY" = "stackrox/test-gh-actions" ] && [ "$NAMED_RELEASE_PATCH" == "0.0.0" ]; }; then
        comment_on_issues_list "$OPEN_ISSUES"
        comment_on_issues_list "$CLOSED_WITH_PR"
    fi
fi

if [ -n "$OPEN_ISSUES" ] || [ -n "$CLOSED_WITH_PR" ]; then
    exit 1
fi
