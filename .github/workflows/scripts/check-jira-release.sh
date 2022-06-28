#!/bin/bash
#
# Queries Jira for the release date.
#
set -euo pipefail

check_not_empty() {
    for V in "$@"; do
        typeset -n VAR="$V"
        if [ -z "${VAR:-}" ]; then
            echo "::error::Variable $V is not set or empty"
            exit 1
        fi
    done
}

VERSION="$1"

check_not_empty \
    GITHUB_STEP_SUMMARY \
    JIRA_TOKEN jira_project \
    VERSION

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
