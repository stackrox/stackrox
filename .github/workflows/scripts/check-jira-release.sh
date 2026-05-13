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
    JIRA_TOKEN \
    JIRA_BASE_URL \
    JIRA_USER

release=$(get_jira_release "$PROJECT" "$VERSION")
if [ -z "$release" ]; then
    gh_log error "Couldn't find JIRA release \`$VERSION\`."
    exit 1
fi

IS_RELEASED=$(echo "$release" | jq -r ".released")
if [ "$IS_RELEASED" == "true" ]; then
    gh_log error "JIRA release \`$VERSION\` is already released."
    exit 1
fi

JIRA_RELEASE_DATE=$(echo "$release" | jq -r ".releaseDate // \"\"")
if [ -z "${JIRA_RELEASE_DATE}" ]; then
    gh_log error "Couldn't find release date for JIRA release \`$VERSION\`."
    exit 1
fi

gh_summary "Release date: $JIRA_RELEASE_DATE"
gh_output date "$JIRA_RELEASE_DATE"
