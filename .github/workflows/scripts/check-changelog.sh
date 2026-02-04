#!/usr/bin/env bash
#
# Checks the CHANGELOG.md for a tag.
# If there is no section for the version inferred from the tag, an error is logged and the script exits with status 1.
#
# Local run:
#
#   bash local-env.sh check-changelog <release-branch> <tag>
#
set -euo pipefail

RELEASE_BRANCH="$1"
TAG="$2"

check_not_empty \
  RELEASE_BRANCH \
  TAG

VERSION="${TAG/-rc.[0-9]*/}"
CHANGELOG="$(fetch_changelog "${RELEASE_BRANCH}" "${VERSION}")"
if [ -z "$CHANGELOG" ]; then
    gh_log error "No CHANGELOG content found for $VERSION (inferred from the tag '$TAG')."
    gh_summary "No CHANGELOG content found for $VERSION."
    gh_summary "Most likely, this is because the \`start-release\` workflow failed to update the \`CHANGELOG.md\`."
    gh_summary "➡️  Please check the \`start-release\` workflow that started this release and the \`CHANGELOG.md\` on the release branch."
    gh_summary "➡️  There should be a non-empty section for the version $VERSION in the \`CHANGELOG.md\`."
    exit 1
fi

CHANGELOG_LENGTH="$(wc -m <<< "$CHANGELOG" | awk '{ print $1 }')"
MAX_LENGTH=150000
if [ "$CHANGELOG_LENGTH" -gt "$MAX_LENGTH" ]; then
  MESSAGE="The CHANGELOG for $VERSION is too long, it has $CHANGELOG_LENGTH characters, when GitHub API allows only $MAX_LENGTH characters for release notes."
  gh_log error "$MESSAGE"
  gh_summary "$MESSAGE"
  gh_summary "➡️  Please shorten the CHANGELOG for $VERSION to $MAX_LENGTH characters or less."
  exit 1
fi
