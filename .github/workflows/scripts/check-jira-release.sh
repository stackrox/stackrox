#!/bin/bash
#
# Queries Jira for the release date.
#
set -euo pipefail

check_not_empty() {
    local VAR
    typeset -n VAR
    VAR="$1"
    if [ -z "${VAR:-}" ]; then
        echo "::error::Variable $1 is not set or empty"
        exit 1
    fi
}

VERSION="$1"

for VAR in \
    GITHUB_STEP_SUMMARY \
    JIRA_TOKEN jira_project \
    VERSION; do
    check_not_empty "$VAR"
done

JIRA_RELEASE_DATE=$(curl --fail -sSL \
    -H "Authorization: Bearer $JIRA_TOKEN" \
    "https://issues.redhat.com/rest/api/2/project/$jira_project/versions" |
    jq -r ".[] | select(.name == \"$VERSION\" and .released == false) | .releaseDate")

if [ -z "$JIRA_RELEASE_DATE" ]; then
    echo "::error::Couldn't find unreleased JIRA release \`$VERSION\`."
else
    echo "Release date: $JIRA_RELEASE_DATE" >>"$GITHUB_STEP_SUMMARY"
    echo "::set-output name=date::$JIRA_RELEASE_DATE"
fi
