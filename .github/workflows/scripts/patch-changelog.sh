#!/usr/bin/env bash
#
# Patches CHANGELOG.md on the main branch and creates a PR.
#
set -euo pipefail

VERSION="$1"
REF="$2"
BRANCH="$3"

check_not_empty \
    VERSION \
    REF \
    BRANCH \
    \
    GITHUB_SERVER_URL \
    GITHUB_REPOSITORY \
    GITHUB_ACTOR \
    DRY_RUN \
    main_branch

create_pr() {
    # shellcheck disable=SC2154
    gh pr create \
    --title "$TITLE" \
    --base "$main_branch" \
    --body "Cutting \`CHANGELOG.md\` after the $BRANCH branch." \
    --assignee "$GITHUB_ACTOR"
}

if grep "^## \[${VERSION}\]$" CHANGELOG.md; then
    gh_summary "\`CHANGELOG.md@$REF\` has got already the \`[$VERSION]\` section."
    exit 0
fi

CHANGELOG_BRANCH="automation/changelog-$VERSION"
TITLE="Advance \`CHANGELOG.md\` to the next release"

if git ls-remote --quiet --exit-code origin "$CHANGELOG_BRANCH"; then
    gh_summary "Branch \`$CHANGELOG_BRANCH\` already exists."

    mapfile -t EXISTING_PR <<<"$(gh pr list --head "$CHANGELOG_BRANCH" --search "$TITLE in:title" --state open --json number,url --jq ".[] | .number,.url")"
    if [ "${#EXISTING_PR[@]}" -eq 2 ]; then
        gh_summary ":arrow_right: There is already an open [PR ${EXISTING_PR[0]}](${EXISTING_PR[1]})," \
            "which needs to be merged, ensuring all lines added to \`CHANGELOG.md\` after the release branch are listed" \
            "above the \`[${VERSION}]\` section."
    else
        gh_summary ":arrow_right: There is no open PR from this branch. Please open a PR to the $main_branch if" \
            "\`[CHANGELOG.md](${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}/blob/$main_branch/CHANGELOG.md)\`" \
            "still needs to be updated."
    fi
    exit 0
fi

git switch --create "$CHANGELOG_BRANCH"
sed -i "s/## \[NEXT RELEASE\]/\\0\n\n### Added Features\n\n### Removed Features\n\n### Deprecated Fatures\n\n### Technical Changes\n\n## [$VERSION]\n\n/" CHANGELOG.md
git add CHANGELOG.md

if ! git diff-index --quiet HEAD; then
    git commit --message "Next version in changelog after $VERSION"

    PR_URL=""
    if [ "$DRY_RUN" = "false" ]; then
        git push --set-upstream origin "$CHANGELOG_BRANCH"
        PR_URL=$(create_pr)
    fi
    # TODO: Add labels to skip CI runs

    gh_summary ":arrow_right: Review and merge the [PR]($PR_URL)" \
        "that has been created for the \`$main_branch\` branch with advanced \`CHANGELOG.md\`."
fi
