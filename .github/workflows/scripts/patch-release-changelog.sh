#!/usr/bin/env bash

set -euo pipefail

NEW_VERSION="$1"

check_not_empty \
    NEW_VERSION

replace_next_release_section() {
    sed -i "s/## \[NEXT RELEASE\]/## [${NEW_VERSION}]/" CHANGELOG.md
}

add_new_section() {
    old_version=$(grep "^## \[" CHANGELOG.md | head -n 1 | sed 's/## \[//; s/\]//')
    major_minor_version=$(echo "${NEW_VERSION}" | cut -d. -f1,2)

    changelog_section=$(cat <<EOF
**Full Changelog**: [${old_version}...${NEW_VERSION}](https://github.com/${GITHUB_REPOSITORY}/compare/${old_version}...${NEW_VERSION})

For a description of the changes, review the [Release Notes](https://docs.redhat.com/en/documentation/red_hat_advanced_cluster_security_for_kubernetes/${major_minor_version}/html/release_notes/index) on the Red Hat Documentation portal.
EOF
    )

    # Escape special characters for sed: & \ / and newlines
    escaped_section=$(printf '%s' "$changelog_section" | sed 's/[&\\/]/\\&/g' | sed ':a;N;$!ba;s/\n/\\n/g')

    # Insert the new section with heading before the old version heading
    sed -i "s|## \[${old_version}\]|## [${NEW_VERSION}]\n\n${escaped_section}\n\n&|" CHANGELOG.md
}

if grep "^## \[${NEW_VERSION}\]$" CHANGELOG.md; then
    gh_summary "\`CHANGELOG.md\` already has the \`[${NEW_VERSION}]\` section. No update required."
    exit 0
fi

if [[ "${NEW_VERSION}" == *.0 ]]; then
    gh_log debug "The new version \`${NEW_VERSION}\` ends with .0. Need to replace the NEXT RELEASE section."
    replace_next_release_section
else
    gh_log debug "The new version \`${NEW_VERSION}\` does not end with .0. Need to add a new section with the template."
    add_new_section
fi

git add CHANGELOG.md
if ! git diff-index --quiet HEAD; then
    git commit --message "chore(release): update changelog for ${NEW_VERSION}"
    gh_summary "\`CHANGELOG.md\` has been updated on the release branch."
fi
