#!/bin/bash
#
# Checks if there are open issues for the given release
# and adds comments to such issues.
#
set -euo pipefail

PROJECTS="$1"
RELEASE="$2"
PATCH="$3"
RELEASE_PATCH="$4"

for VAR in \
    JIRA_TOKEN DRY_RUN \
    PROJECTS RELEASE PATCH RELEASE_PATCH; do
    check_not_empty "$VAR"
done

JQL="project IN ($PROJECTS) \
AND fixVersion IN (\"$RELEASE_PATCH\", \"$RELEASE.$PATCH\") \
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

GH_MD_FORMAT_LINE=$(
    cat <<EOF
"* [\(.key)](https://issues.redhat.com/browse/\(.key)): **" \
+ (.fields.assignee.displayName // "unassigned") \
+ "** (\(.fields.status.name)) — _" \
+ (.fields.summary | gsub (" +$";"")) \
+ "_"
EOF
)

ISSUES=$(get_issues | jq -r ".issues[] | $GH_MD_FORMAT_LINE" | sort)

if [ -z "$ISSUES" ]; then
    gh_summary "All issues for Jira release $RELEASE_PATCH are closed."
    exit 0
fi

gh_summary <<EOF
The following Jira issues are still open for release $RELEASE_PATCH:

$ISSUES

Contact the assignees to clarify the status.
EOF

gh_log error "There are non-closed Jira issues for version $RELEASE_PATCH."

OPEN_ISSUES=$(get_issues)

echo "Open issues:"
echo "$OPEN_ISSUES" | jq -r '.issues[] | "\(.key) - \(.fields.assignee.displayName)"'

if [ "$DRY_RUN" != "true" ]; then
    echo "$OPEN_ISSUES" | jq -r ".issues[] | .key" | while read -r KEY; do
        comment_issue "$KEY"
    done
fi

exit 1
