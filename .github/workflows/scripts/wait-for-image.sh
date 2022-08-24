#!/usr/bin/env bash
#
# Wait for an image to appear on Quay.io
#
set -euo pipefail

NAME="$1"
TAG="$2"

check_not_empty \
    NAME \
    TAG \
    \
    QUAY_TOKEN \
    DRY_RUN

find_tag() {
    curl --silent --show-error --fail --location \
        -H "Authorization: Bearer $QUAY_TOKEN" \
        -X GET "https://quay.io/api/v1/repository/rhacs-eng/$1/tag?specificTag=$2" |
        jq -r ".tags[0].name"
}

# Seconds:
TIME_LIMIT=1200
INTERVAL=30

# bash built-in variable
SECONDS=0

FOUND_TAG=""
while [ "$SECONDS" -le "$TIME_LIMIT" ]; do
    FOUND_TAG="$(find_tag "$NAME" "$TAG")"
    if [ "$FOUND_TAG" = "$TAG" ]; then
        gh_log notice "Image '$NAME:$TAG' has been found on Quay.io."
        exit 0
    fi
    if [ "$DRY_RUN" = "true" ]; then
        break
    fi
    echo "Waiting..."
    sleep "$INTERVAL"
done

gh_log error "Image '$NAME:$TAG' has not been found on Quay.io."
exit 1
