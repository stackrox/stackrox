#!/usr/bin/env bash

# Pull from a quay.io user account source and push to an organizational level
# quay.io account as required for scanning.

set -euo pipefail

GITROOT="$(git rev-parse --show-toplevel)"
[[ -n "${GITROOT}" ]] || { echo >&2 "Could not determine git root!"; exit 1; }

[[ -n "${QUAY_CGORMAN1_RO_USER}" ]] || { echo >&2 "Missing env QUAY_CGORMAN1_RO_USER for user account"; exit 1; }
[[ -n "${QUAY_CGORMAN1_RO_PASSWORD}" ]] || { echo >&2 "Missing env QUAY_CGORMAN1_RO_PASSWORD for user account"; exit 1; }

[[ -n "${QUAY_USERNAME}" ]] || { echo >&2 "Missing env QUAY_USERNAME for organization account"; exit 1; }
[[ -n "${QUAY_PASSWORD}" ]] || { echo >&2 "Missing env QUAY_PASSWORD for organization account"; exit 1; }

function pull {
  local image=$1

  docker login -u "${QUAY_CGORMAN1_RO_USER}" --password-stdin <<<"${QUAY_CGORMAN1_RO_PASSWORD}" quay.io
  src="quay.io/cgorman1/${image}"
  docker pull "$src"
  echo "Successfully pulled $src"

  dest="quay.io/stackrox/${image}"
  docker tag "$src" "$dest"
  docker login -u "${QUAY_USERNAME}" --password-stdin <<<"${QUAY_PASSWORD}" quay.io
  docker push "$dest"

  echo "Successfully pushed $dest"
}

function pull_without_rhel {
  local repo=$1
  local tag=$2
  pull "${repo}:${tag}"
}

function pull_with_rhel {
  local repo=$1
  local tag=$2

  pull "${repo}:${tag}"
  pull "${repo}-rhel:${tag}"
}

# Main images
RELEASE_TAG=$(make --no-print-directory --quiet -C "${GITROOT}" tag)
pull_with_rhel main "$RELEASE_TAG"

# Docs image
DOCS_PRERELEASE_TAG=$(make --no-print-directory --quiet -C "${GITROOT}" docs-tag)
pull_without_rhel docs "$DOCS_PRERELEASE_TAG"

# Collector images
COLLECTOR_TAG=$(make --no-print-directory --quiet -C "${GITROOT}" collector-tag)
pull_with_rhel "collector" "$COLLECTOR_TAG"

# Legacy scanner images
SCANNER_TAG=$(make --no-print-directory --quiet -C "${GITROOT}" scanner-tag)
pull_with_rhel scanner "$SCANNER_TAG"
pull_with_rhel "scanner-db" "$SCANNER_TAG"
