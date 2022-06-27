#!/bin/bash
#
# Patches CHANGELOG.md on the main branch and creates a PR.
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
REF="$2"
BRANCH="$3"
DRY_RUN="$4"

for VAR in \
    GITHUB_STEP_SUMMARY GITHUB_SERVER_URL GITHUB_REPOSITORY GITHUB_ACTOR \
    main_branch \
    VERSION REF BRANCH DRY_RUN; do
    check_not_empty "$VAR"
done

if grep "^## \[${VERSION}\]$" CHANGELOG.md; then
    echo "\`CHANGELOG.md@$REF\` has got already the \`[$VERSION]\` section." >>"$GITHUB_STEP_SUMMARY"
    exit 0
fi

CHANGELOG_BRANCH="automation/changelog-$VERSION"
TITLE="Advance \`CHANGELOG.md\` to the next release"

if git ls-remote --quiet --exit-code origin "$CHANGELOG_BRANCH"; then
    echo "Branch \`$CHANGELOG_BRANCH\` already exists." >>"$GITHUB_STEP_SUMMARY"

    mapfile -t EXISTING_PR <<<"$(gh pr list --head "$CHANGELOG_BRANCH" --search "$TITLE in:title" --state open --json number,url --jq ".[] | .number,.url")"
    if [ "${#EXISTING_PR[@]}" -eq 2 ]; then
        echo ":arrow_right: There is already an open [PR ${EXISTING_PR[0]}](${EXISTING_PR[1]})," \
            "which needs to be merged, ensuring all lines added to \`CHANGELOG.md\` after the release branch are listed" \
            "above the \`[${VERSION}]\` section." >>"$GITHUB_STEP_SUMMARY"
    else
        echo ":arrow_right: There is no open PR from this branch. Please open a PR to the $main_branch if" \
            "\`[CHANGELOG.md](${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}/blob/$main_branch/CHANGELOG.md)\`" \
            "still needs to be updated." >>"$GITHUB_STEP_SUMMARY"
    fi
    exit 0
fi

git switch --create "$CHANGELOG_BRANCH"
sed -i "s/## \[NEXT RELEASE\]/\\0\n\n## [$VERSION]/" CHANGELOG.md
git add CHANGELOG.md

if ! git diff-index --quiet HEAD; then
    git commit --message "Next version in changelog after $VERSION"

    PR_URL=""
    if [ "$DRY_RUN" != "true" ]; then
        git push --set-upstream origin "$CHANGELOG_BRANCH"
        PR_URL=$(gh pr create \
            --title "$TITLE" \
            --base "$main_branch" \
            --body "Cutting \`CHANGELOG.md\` after the $BRANCH branch." \
            --assignee "$GITHUB_ACTOR")
    fi
    # TODO: Add labels to skip CI runs

    echo "::notice::Review and merge the [PR]($PR_URL) that has been created for the \`$main_branch\` branch with advanced \`CHANGELOG.md\`."
fi
