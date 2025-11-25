#!/usr/bin/env bash
#
# Create a release for a tag.
# If a release already exists under the tag (for a different/previous commit), a new release is published.
#
# Local wet run:
#
#   DRY_RUN=false bash local-env.sh create-release <tag> <prerelease>
#
set -euo pipefail

TAG="$1"
RELEASE_BRANCH="$2"
PRERELEASE="$3"

check_not_empty \
  TAG \
  PRERELEASE \
  \
  DRY_RUN

delete_existing_release() {
  if gh release view "$TAG" > /dev/null 2>&1; then
    gh_log warning "Release $TAG exists."
    if [ "$DRY_RUN" = "false" ]; then
      gh release delete \
      --yes \
      --repo "$GITHUB_REPOSITORY" \
      "$TAG"
    fi
    gh_log warning "Existing release '$TAG' has been deleted."
  fi
}

create_release() {
  delete_existing_release
  COMMIT_HASH_FOR_TAG="$(git rev-list -1 "$TAG")"
  if [ "$DRY_RUN" = "false" ]; then
    URL=$(gh release create "$TAG" \
      --prerelease="${PRERELEASE}" \
      --notes "$RELEASE_NOTES" \
      --repo "$GITHUB_REPOSITORY" \
      --target "$COMMIT_HASH_FOR_TAG")
  fi
  gh_summary "Created GitHub release [$TAG]($URL)"
  echo "url=$URL" >> "$GITHUB_OUTPUT"
}

VERSION="${TAG/-rc.[0-9]*/}"
RELEASE_NOTES="$(fetch_changelog "${RELEASE_BRANCH}" "${VERSION}")"
create_release
