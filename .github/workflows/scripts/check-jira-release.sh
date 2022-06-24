#!/bin/bash
#
# Queries Jira for the release date.
#
set -euo pipefail

cat << EOF > /dev/null
GitHub variables: $GITHUB_STEP_SUMMARY
Custom variables: $JIRA_TOKEN $jira_project
EOF

VERSION="$1"

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
