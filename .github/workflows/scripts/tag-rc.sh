#!/usr/bin/env bash
#
# Add an empty commit and delete the remote tag if the tag exists.
# Tag and push the branch.
#
# Local wet run:
#
#   DRY_RUN=false bash local-env.sh tag-rc <tag>
#
set -euo pipefail

TAG="$1"

check_not_empty \
    TAG \
    \
    DRY_RUN

if [ "$(git tag "$TAG" --points-at HEAD)" = "$TAG" ]; then
    git push --delete origin "$TAG"
    gh_log warning "Tag '$TAG' has been deleted."
fi

git commit --allow-empty --message "Empty commit to trigger CI"
git tag --force --annotate "$TAG" --message "Upstream release automation"

if [ "$DRY_RUN" = "false" ]; then
    git push --follow-tags
fi

gh_summary "Release branch has been tagged with $TAG."
