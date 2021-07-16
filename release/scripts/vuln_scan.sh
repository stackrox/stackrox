#!/usr/bin/env bash

set -eu

GITROOT="$(git rev-parse --show-toplevel)"
[[ -n "${GITROOT}" ]] || { echo >&2 "Could not determine git root!"; exit 1; }

# Pull all the required images and push them to Quay to be scanned

docker login quay.io

function retag {
  local image=$1

  docker pull "$image"

  new_image="quay.io/${image}"
  docker tag "$image" "$new_image"
  docker push "$new_image"

  echo "Successfully pushed $new_image"
}

function retag_without_rhel {
  local repo=$1
  local tag=$2
  retag "${repo}:${tag}"
}

function retag_with_rhel {
  local repo=$1
  local tag=$2

  retag "${repo}:${tag}"
  retag "${repo}-rhel:${tag}"
}

# Main images
RELEASE_TAG=$(make --no-print-directory --quiet -C "${GITROOT}" tag)
retag_with_rhel stackrox/main "$RELEASE_TAG"

# Docs image
DOCS_PRERELEASE_TAG=$(cat "${GITROOT}/DOCS_VERSION")
retag_without_rhel stackrox/docs "$DOCS_PRERELEASE_TAG"

# Collector images
COLLECTOR_TAG=$(cat "${GITROOT}/COLLECTOR_VERSION")
retag_with_rhel "stackrox/collector" "$COLLECTOR_TAG"

# Legacy scanner images
SCANNER_TAG=$(cat "${GITROOT}/SCANNER_VERSION")
retag_with_rhel stackrox/scanner "$SCANNER_TAG"
retag_with_rhel "stackrox/scanner-db" "$SCANNER_TAG"
