#!/usr/bin/env bash
#
# Checks if there are open issues for the given release
# and adds comments to such issues.

set -euo pipefail

PROJECT="$1"
RELEASE="$2"
PATCH="$3"
NAMED_RELEASE_PATCH="$4"

check_not_empty \
    PROJECT \
    RELEASE \
    PATCH \
    NAMED_RELEASE_PATCH \
    \
    JIRA_TOKEN \
    JIRA_BASE_URL \
    JIRA_USER \
    DRY_RUN

# Finds closed issues that have open PRs for the given release.
JQL_CLOSED_WITH_PR="project = $PROJECT \
AND fixVersion = '$RELEASE.$PATCH' \
AND statusCategory = Done \
AND development[pullrequests].open > 0 \
AND (Component IS EMPTY or Component NOT IN (Documentation, 'ACS Cloud Service')) \
ORDER BY assignee"

# Finds open issues for the given release.
JQL_OPEN_ISSUES="project = $PROJECT \
AND fixVersion = '$RELEASE.$PATCH' \
AND statusCategory != done \
AND (Component IS EMPTY or Component NOT IN (Documentation, 'ACS Cloud Service')) \
AND type NOT IN (Epic, 'Feature Request') \
ORDER BY assignee"

comment_on_issues_list() {
    local FILE_NAME="$1"
    local COMMENT="$2"
    while read -r KEY; do
        comment_on_issue "$KEY" "$COMMENT"
    done < <(jq -r ".[] | .key" "$FILE_NAME")
}

get_issues_summary() {
    local FILE_NAME="$1"
    read -r GH_MD_FORMAT_LINE <<EOF
* [\(.key)](${JIRA_BASE_URL}/issues/\(.key)): \
**\(.fields.assignee.displayName // "unassigned")** \
(\(.fields.issuetype.name), \(.fields.status.name)) — _\(.fields.summary | gsub (" +$";""))_
EOF
    jq -r ".[] | \"$GH_MD_FORMAT_LINE\"" "$FILE_NAME" | sort
}

# First, validate that the release exists in Jira.
# Otherwise, we'll get 400 errors when querying issues from Jira API.
release=$(get_jira_release "$PROJECT" "$RELEASE.$PATCH")
if [ -z "$release" ]; then
    gh_log error "Couldn't find JIRA release \`$RELEASE.$PATCH\`."
    exit 1
fi

get_issues "$JQL_CLOSED_WITH_PR" "closed-with-open-prs.json"
CLOSED_WITH_PR_SUMMARY=$(get_issues_summary "closed-with-open-prs.json")
if [ -n "$CLOSED_WITH_PR_SUMMARY" ]; then
    gh_summary <<EOF
:warning: The following Jira issues have been marked complete, but have open PRs:

$CLOSED_WITH_PR_SUMMARY

EOF
fi

get_issues "$JQL_OPEN_ISSUES" "open-issues.json"
OPEN_ISSUES_SUMMARY=$(get_issues_summary "open-issues.json")
if [ -z "$OPEN_ISSUES_SUMMARY" ]; then
    gh_summary "All issues for Jira release $NAMED_RELEASE_PATCH are closed."
    exit 0
fi

gh_summary <<EOF
:red_circle: The following Jira issues are still open for release $NAMED_RELEASE_PATCH:

$OPEN_ISSUES_SUMMARY

:arrow_right: Contact the assignees to clarify the status.
EOF

gh_log error "There are non-closed Jira issues for version $NAMED_RELEASE_PATCH. For details, see the task log."

echo "--------------------------------"
echo  "Open issues for release $NAMED_RELEASE_PATCH:"
echo  "$OPEN_ISSUES_SUMMARY"
echo "--------------------------------"
echo  "Completed issues with open PRs:"
echo  "$CLOSED_WITH_PR_SUMMARY"
echo "--------------------------------"

if [ "$DRY_RUN" = "false" ]; then
    if [ "$GITHUB_REPOSITORY" = "stackrox/stackrox" ] || { [ "$GITHUB_REPOSITORY" = "stackrox/test-gh-actions" ] && [ "$NAMED_RELEASE_PATCH" == "0.0.0" ]; }; then
        comment_on_issues_list "open-issues.json" "Release $NAMED_RELEASE_PATCH is ongoing. This issue is still open. Please update the status of this issue and notify the release engineer."
        comment_on_issues_list "closed-with-open-prs.json" "Release $NAMED_RELEASE_PATCH is ongoing. This issue has been marked complete, but has open PRs. Please update the status of this issue and notify the release engineer."
    fi
fi

if [ -n "$OPEN_ISSUES_SUMMARY" ] || [ -n "$CLOSED_WITH_PR_SUMMARY" ]; then
    exit 1
fi
