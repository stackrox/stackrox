#!/usr/bin/env bash
#
# Queries Jira for the release date.
#
set -euo pipefail

VERSION="$1"
PROJECT="$2"

check_not_empty \
    VERSION \
    PROJECT \
    \
    JIRA_TOKEN

JIRA_RELEASE_DATE=$(curl --fail -sSL \
    -H "Authorization: Bearer $JIRA_TOKEN" \
    "https://issues.redhat.com/rest/api/2/project/$PROJECT/versions" |
    jq -r ".[] | select(.name == \"$VERSION\" and .released == false) | .releaseDate")

if [ -z "$JIRA_RELEASE_DATE" ]; then
    gh_log error "Couldn't find unreleased JIRA release \`$VERSION\`."
else
    gh_summary "Release date: $JIRA_RELEASE_DATE"
    gh_output date "$JIRA_RELEASE_DATE"
fi
