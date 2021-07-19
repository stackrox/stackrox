#!/usr/bin/env bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

GITROOT="$(git rev-parse --show-toplevel)"
[[ -n "${GITROOT}" ]] || { echo >&2 "Could not determine git root!"; exit 1; }

die() {
  echo >&2 "$@"
  exit 1
}

# src_img "<name>:<tag>" returns the source image reference for the argument.
# This will be of the form `[collector.]stackrox.io/<name>[-rhel]:<tag>`.
src_img() {
  local name
  local tag
  if [[ "$1" =~ ^(.+):([^:]+)$ ]]; then
    name="${BASH_REMATCH[1]}"
    tag="${BASH_REMATCH[2]}"
  else
    echo >&2 "Invalid image format $1"
    return 1
  fi

  local registry="stackrox.io"
  if [[ "$name" == "collector" ]]; then
    registry="collector.stackrox.io"
  fi
  local fullname="${name}-rhel"
  if [[ "$name" == "roxctl" ]]; then
    fullname="$name"
  fi
  echo "${registry}/${fullname}:${tag}"
}

# dst_img "<name>:<tag>" returns the destination image reference for the argument.
# This will always be of the form `quay.io/rhacs/rh-acs----<name>:<tag>`.
# Note that the `rh-acs----` is required by the registry proxy to locate the
# correct image in quay.io/rhacs.
dst_img() {
  local name
  local tag
  if [[ "$1" =~ ^(.+):([^:]+)$ ]]; then
    name="${BASH_REMATCH[1]}"
    tag="${BASH_REMATCH[2]}"
  else
    echo >&2 "Invalid image format $1"
    return 1
  fi
  echo "quay.io/rhacs/rh-acs----${name}:${tag}"
}

# version_check <main version> <scanner version> <collector version> checks that:
# - all versions are non-empty
# - that the versions match the state of the Git repository (unless SKIP_REPO_VERSION_CHECK or
#   SKIP_VERSION_CHECK are set to "true")
# - that the scanner and collector versions match those baked into the main image of the respective
#   main version (unless SKIP_IMAGE_VERSION_CHECK or SKIP_VERSION_CHECK are set to "true")
version_check() {
  local main_version="$1"
  local scanner_version="$2"
  local collector_version="$3"

  [[ -n "$main_version" ]] || die "No main version specified"
  [[ -n "$scanner_version" ]] || die "No scanner version specified"
  [[ -n "$collector_version" ]] || die "No collector version specified"

  if [[ "${SKIP_VERSION_CHECK:-}" == "true" ]]; then
    return 0
  fi

  if [[ "${SKIP_REPO_VERSION_CHECK:-}" != "true" ]]; then
    local repo_main_version
    repo_main_version="$(make --no-print-directory --quiet -C "${GITROOT}" tag)"
    if [[ "$repo_main_version" != "$main_version" ]]; then
      echo >&2 "Main version ${main_version} does not match repository version ${repo_main_version}."
      echo >&2 "Set SKIP_REPO_VERSION_CHECK=true in order to suppress this check for local testing."
      exit 1
    fi
    local repo_collector_version
    repo_collector_version="$(make --no-print-directory --quiet -C "${GITROOT}" collector-tag)"
    if [[ "$repo_collector_version" != "$collector_version" ]]; then
      echo >&2 "Collector version ${collector_version} does not match repository version ${repo_collector_version}."
      echo >&2 "Set SKIP_REPO_VERSION_CHECK=true in order to suppress this check for local testing."
      exit 1
    fi
    local repo_scanner_version
    repo_scanner_version="$(make --no-print-directory --quiet -C "${GITROOT}" scanner-tag)"
    if [[ "$repo_scanner_version" != "$scanner_version" ]]; then
      echo >&2 "Scanner version ${scanner_version} does not match repository version ${repo_scanner_version}."
      echo >&2 "Set SKIP_REPO_VERSION_CHECK=true in order to suppress this check for local testing."
      exit 1
    fi
  fi

  if [[ "${SKIP_IMAGE_VERSION_CHECK:-}" != "true" ]]; then
    local main_image
    main_image="$(src_img "main:${main_version}")"

    local versions_json
    versions_json="$(docker run --entrypoint /stackrox/roxctl "$main_image" version --json)"

    local image_collector_version
    image_collector_version="$(jq -r '.CollectorVersion' <<<"$versions_json")"
    if [[ "$image_collector_version" != "$collector_version" ]]; then
      echo >&2 "Collector version ${collector_version} does not match image version ${image_collector_version}."
      echo >&2 "Set SKIP_IMAGE_VERSION_CHECK=true in order to suppress this check for local testing."
      exit 1
    fi
    local image_scanner_version
    image_scanner_version="$(jq -r '.ScannerVersion' <<<"$versions_json")"
    if [[ "$image_scanner_version" != "$scanner_version" ]]; then
      echo >&2 "Scanner version ${scanner_version} does not match repository version ${image_scanner_version}."
      echo >&2 "Set SKIP_IMAGE_VERSION_CHECK=true in order to suppress this check for local testing."
      exit 1
    fi
  fi
}
