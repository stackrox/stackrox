#!/usr/bin/env bash

set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$DIR/../../"
MAIN_TAG="$(make -C "$REPO_ROOT" --no-print-directory --quiet tag)"
SCANNER_VERSION="$(make -C "$REPO_ROOT" --no-print-directory --quiet scanner-tag)"
COLLECTOR_VERSION="$(make -C "$REPO_ROOT" --no-print-directory --quiet collector-tag)"

REGISTRIES=( "docker.io/stackrox" "quay.io/rhacs-eng" )

main() {
  local CIRCLE_TAG="$1"
  [[ -n "${MAIN_TAG}" ]]          || die "Error: MAIN_TAG undefined"
  [[ -n "${SCANNER_VERSION}" ]]   || die "Error: SCANNER_VERSION undefined"
  [[ -n "${COLLECTOR_VERSION}" ]] || die "Error: COLLECTOR_VERSION undefined"

  if is_release "$CIRCLE_TAG"; then
    REGISTRIES+=( "stackrox.io" )
  fi
  for TARGET_REGISTRY in "${REGISTRIES[@]}"; do
    retag --image "docker.io/stackrox/scanner:${SCANNER_VERSION}"    --change-registry "${TARGET_REGISTRY}" --retag "$MAIN_TAG"
    retag --image "docker.io/stackrox/scanner-db:${SCANNER_VERSION}" --change-registry "${TARGET_REGISTRY}" --retag "$MAIN_TAG"

    retag --image "${TARGET_REGISTRY/stackrox.io/collector.stackrox.io}/collector:${COLLECTOR_VERSION}"      --retag "$MAIN_TAG"
    retag --image "${TARGET_REGISTRY/stackrox.io/collector.stackrox.io}/collector:${COLLECTOR_VERSION}-base" --retag "${MAIN_TAG}-slim"
    retag --image "${TARGET_REGISTRY/stackrox.io/collector.stackrox.io}/collector:${COLLECTOR_VERSION}-base" --retag "$MAIN_TAG"        --add-suffix "-slim"
  done
  # Externally exposed registries require '-rhel' suffix for backward compatibility with some customers.
  if is_release "$CIRCLE_TAG"; then
    retag --image "stackrox.io/scanner:${MAIN_TAG}"                  --add-suffix "-rhel"
    retag --image "stackrox.io/scanner-db:${MAIN_TAG}"               --add-suffix "-rhel"
    retag --image "collector.stackrox.io/collector:${MAIN_TAG}"      --add-suffix "-rhel"
    retag --image "collector.stackrox.io/collector:${MAIN_TAG}-slim" --add-suffix "-rhel"
  fi
}

# retag is an alias introduced for brevity
retag() {
  "${DIR}/../ci/pull-retag-push.sh" "$@"
}

die() {
  echo >&2 "$@"
  exit 1
}

is_release() {
  [[ "${1}" =~ ^[0-9]+(\.[0-9]+){2}$ ]]
}

main "$@"
