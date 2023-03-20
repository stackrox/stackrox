#!/usr/bin/env bash
#
# Tries to cherry-pick merge commits from the main branch to the release branch.
# Adds comments to the original PRs of the problematic commits.
#
set -euo pipefail

MILESTONE="$1"
BRANCH="$2"
RELEASE_PATCH="$3"

check_not_empty \
    MILESTONE \
    BRANCH \
    RELEASE_PATCH \
    \
    GITHUB_REPOSITORY \
    DRY_RUN \
    main_branch

SLACK_MESSAGE_FILE=$(mktemp)

# Increments PICKED variable if managed to pick a cherry.
cherry_pick() {
    IFS=$'\t' read -r PR URL COMMIT _ AUTHOR TITLE <<<"$1"

    # Skip commits merged before branching.
    if git merge-base --is-ancestor "$COMMIT" HEAD; then
        gh_log debug "$COMMIT is already on the release branch"
        return
    fi

    # Find commits with the specific commit message after the fork point
    # to not cherry-pick twice the same commit.
    ALREADY_PICKED_CHERRIES=$(git log \
        --grep "^(cherry picked from commit $COMMIT)\$" --format='%H' \
        HEAD..."$FORK_POINT")

    if [ -n "$ALREADY_PICKED_CHERRIES" ]; then
        gh_summary "Already picked cherries for commit $COMMIT:\n\`\`\`\n$ALREADY_PICKED_CHERRIES\n\`\`\`"
        return
    fi

    if git cherry-pick -x "$COMMIT"; then
        gh_summary "Cherry-picked $COMMIT from PR $PR"

        [ "$DRY_RUN" = "false" ] &&
            gh pr comment "$PR" --body "Merge commit has been cherry-picked to branch \`$BRANCH\`."

        PICKED=$((PICKED+1))
    else
        git cherry-pick --abort

        [ "$DRY_RUN" = "false" ] &&
            gh pr comment "$PR" --body "Please merge the changes to branch \`$BRANCH\`."

        gh_summary "* [PR $PR]($URL) by **${AUTHOR}** (_${TITLE}_) could not be cherry-picked. Merge commit: \`$COMMIT\`."

        echo "- <$URL|PR $PR> by *$AUTHOR* — $TITLE" \
            >>"$SLACK_MESSAGE_FILE"
    fi
}

find_fork_point() {
    set +o pipefail
    diff -u \
        <(git rev-list --first-parent "$1") \
        <(git rev-list --first-parent "$2") |
        sed -ne 's/^ //p' |
        head -1
    set -o pipefail
}

# shellcheck disable=SC2154
PR_COMMITS=$(gh pr list -s merged \
    --search "milestone:$MILESTONE" \
    --base "$main_branch" \
    --json mergedAt,number,url,mergeCommit,author,title \
    --jq 'sort_by(.mergedAt) | .[] | "\(.number)\t\(.url)\t\(.mergeCommit.oid)\t\(.mergedAt)\t\(.author.login)\t\(.title)"')

if [ -n "$PR_COMMITS" ]; then
    echo "Commits of merged PRs:"
    echo "$PR_COMMITS"
    echo "..."
fi

FORK_POINT=$(find_fork_point "$BRANCH" "$main_branch")

gh_log debug "Fork point: $FORK_POINT"

if [ -n "$PR_COMMITS" ]; then
    PICKED=0
    while read -r PR_COMMIT; do
        cherry_pick "$PR_COMMIT"
    done <<<"$PR_COMMITS"
fi

# Replace % with %25, escape \n and " to pass as step output to Slack message.
FAILED=$(sed ':a; N; $!ba; s/%/%25/g; s/\n/\\n/g; s/"/\\"/g' "$SLACK_MESSAGE_FILE")

rm -f "$SLACK_MESSAGE_FILE"

if [ -n "$FAILED" ]; then
    gh_output bad-cherries "$FAILED"
    gh_summary <<EOF

As some of the PRs could not be cherry-picked, please help the authors to merge
their changes to the release branch via opening PRs.

The commands may look like the following (mind the placeholders):

    # In the directory of the $GITHUB_REPOSITORY clone:
    git switch "$main_branch"
    git pull
    git switch --create <AUTHOR>/merge-pr-<PR NUMBER>-"$RELEASE_PATCH" \\
      "$BRANCH"
    git cherry-pick -x <MERGE COMMIT SHA>

    # Proceed to resolve merge conflicts. Once done:
    git push --set-upstream origin \$(git branch --show-current)

    # Create PR to the release branch (assuming 'gh' is installed):
    gh create pr --base "$BRANCH" --fill \\
      --milestone "$MILESTONE"
EOF
    exit 1
fi
