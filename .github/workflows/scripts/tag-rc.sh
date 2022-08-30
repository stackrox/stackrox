#!/usr/bin/env bash
#
# Add an empty commit and delete the remote tag if the tag exists.
# Tag and push the branch.
#
set -euo pipefail

TAG="$1"

check_not_empty \
    TAG \
    \
    DRY_RUN

if git ls-remote --tags --exit-code origin "$TAG"; then
    git push --delete origin "$TAG"
    git commit --allow-empty --message "Empty commit to trigger CI"
    gh_log warning "Tag '$TAG' has been deleted and an empty commit has been added to trigger CI."
fi

git tag --force --annotate "$TAG" --message "Upstream release automation"

if [ "$DRY_RUN" = "false" ]; then
    git push --follow-tags
fi

gh_summary "Release branch has been tagged with $TAG."
