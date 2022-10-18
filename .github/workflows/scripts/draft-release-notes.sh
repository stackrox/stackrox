#!/usr/bin/env bash
#
# Drafts release notes from CHANGELOG for the current
# release, uses link to previous tag as fallback.
#
set -euo pipefail

VERSION="$1"
RELEASE_BRANCH="$2"
OUTPUT="${3:-RELEASE_NOTES_GENERATED.md}"

check_not_empty \
    VERSION \
    RELEASE_BRANCH \
    OUTPUT

get_current_release_changelog() {
    gh api \
        -H "Accept: application/vnd.github.v3.raw" \
        "/repos/${REPO_NAME}/contents/CHANGELOG.md?ref=${RELEASE_BRANCH}" \
    > CHANGELOG.md
}

create_release_notes() {
    sed -n "/^## \[$ESCAPED_VERSION]$/,/^## \[/p" CHANGELOG.md | sed '1d;$d' > "$OUTPUT"
    if [ $(wc -m "$OUTPUT" | awk '{ print $1 }') -gt 150000 ]; then
        PREVIOUS_VERSION=$(sed -n "/^## \[$ESCAPED_VERSION]$/,/^## \[/p" CHANGELOG.md | tail -n 1 | tr -d '#[] ')
        echo "**Full Changelog**: https://github.com/${REPO_NAME}/compare/${PREVIOUS_VERSION}...${VERSION}" > "$OUTPUT"
    fi
}

REPO_NAME=$(gh repo view --json nameWithOwner --jq .nameWithOwner)
ESCAPED_VERSION="${VERSION//./\.}"
get_current_release_changelog
create_release_notes
