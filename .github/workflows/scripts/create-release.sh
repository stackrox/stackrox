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
PRERELEASE="$2"

check_not_empty \
  TAG \
  PRERELEASE \
  \
  DRY_RUN

create_release_notes() {
    CHANGELOG="$(gh api \
      -H "Accept: application/vnd.github.v3.raw" \
      "/repos/${GITHUB_REPOSITORY}/contents/CHANGELOG.md?ref=${TAG}"
    )"
    VERSION_WITHOUT_RC="${TAG/-rc.[0-9]*/}"
    ESCAPED_VERSION="${VERSION_WITHOUT_RC//./\.}"
    OUTPUT="$(echo "$CHANGELOG" | sed -n "/^## \[$ESCAPED_VERSION]$/,/^## \[/p" | sed '1d;$d')"
    if [ "$(wc -m <<< "$OUTPUT" | awk '{ print $1 }')" -gt 150000 ]; then
        PREVIOUS_VERSION=$(echo "$CHANGELOG" | sed -n "/^## \[$ESCAPED_VERSION]$/,/^## \[/p" | tail -n 1 | tr -d '#[] ')
        OUTPUT="**Full Changelog**: https://github.com/${GITHUB_REPOSITORY}/compare/${PREVIOUS_VERSION}...${TAG}"
    fi
}

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
      --notes "$OUTPUT" \
      --repo "$GITHUB_REPOSITORY" \
      --target "$COMMIT_HASH_FOR_TAG")
  fi
  echo "url=$URL" >> "$GITHUB_OUTPUT"
}

create_release_notes
create_release
