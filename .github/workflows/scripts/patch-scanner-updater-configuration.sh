#!/usr/bin/env bash
#
# Patches scanner/updater/version/RELEASE_VERSION on the main branch and creates a PR.
#
set -euo pipefail

VERSION="$1"
CONFIG_FILE="scanner/updater/version/RELEASE_VERSION"

check_not_empty \
    VERSION \
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
    --body "Adding $VERSION to the scanner updater configuration." \
    --assignee "$GITHUB_ACTOR"
}



if grep -F "^${VERSION}$" ${CONFIG_FILE}; then
    gh_summary "\`${CONFIG_FILE}\` has got already the \`$VERSION\`."
    exit 0
fi

UPDATE_BRANCH="automation/scanner-updater-configuration-$VERSION"
TITLE="chore(release): Add $VERSION to scanner updater configuration"

if git ls-remote --quiet --exit-code origin "$UPDATE_BRANCH"; then
    gh_summary "Branch \`$UPDATE_BRANCH\` already exists."

    mapfile -t EXISTING_PR <<<"$(gh pr list --head "$UPDATE_BRANCH" --search "$TITLE in:title" --state open --json number,url --jq ".[] | .number,.url")"
    if [ "${#EXISTING_PR[@]}" -eq 2 ]; then
        gh_summary ":arrow_right: There is already an open [PR ${EXISTING_PR[0]}](${EXISTING_PR[1]}), which needs to be merged."
    else
        gh_summary ":arrow_right: There is no open PR from this branch. Please open a PR to the $main_branch if" \
            "\`[RELEASE_VERSION](${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}/blob/$main_branch/${CONFIG_FILE})\`" \
            "still needs to be updated."
    fi
    exit 0
fi

git switch --create "$UPDATE_BRANCH"

echo "${VERSION}" | sort -o "${CONFIG_FILE}" -m - "${CONFIG_FILE}"
git diff
git add "${CONFIG_FILE}"

if ! git diff-index --quiet HEAD; then
    git commit --message "Add $VERSION to scanner updater configuration"

    PR_URL=""
    if [ "$DRY_RUN" = "false" ]; then
        git push --set-upstream origin "$UPDATE_BRANCH"
        PR_URL=$(create_pr)
    fi
    # TODO: Add labels to skip CI runs

    gh_summary ":arrow_right: Review and merge the [PR]($PR_URL)" \
        "that has been created for the \`$main_branch\` branch with updated scanner updater configuration."
fi
