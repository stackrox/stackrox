#!/usr/bin/env bash
set -eo pipefail

if [[ ! -x "$(command -v "roxhelp")" ]]; then
  echo >&2 "Could not find github.com/stackrox/workflow commands. Is it installed?"
  exit 1
fi

DOCKER_USER="${1:-$ROX_DOCKER_USER}"
if [[ -z "$DOCKER_USER" ]]; then
  echo >&2 "Docker user not found, either set env variable ROX_DOCKER_USER, or invoke this script as $0 <USER> <PASSWORD>"
  exit 1
fi

DOCKER_PASSWORD="${2:-$ROX_DOCKER_PASSWORD}"
if [[ -z "$DOCKER_PASSWORD" ]]; then
  echo >&2 "Docker password not found, either set env variable ROX_DOCKER_PASSWORD, or invoke this script as $0 <USER> <PASSWORD>"
  exit 1
fi

if [[ -z "$MAIN_IMAGE_TAG" ]]; then
  MAIN_IMAGE_TAG="$(make tag)"
elif [[ "$MAIN_IMAGE_TAG" == "latest-local-build" ]]; then
  MAIN_IMAGE_TAG="$(docker images --filter="reference=stackrox/main" --format "{{.Tag}}" | head -1)"
fi
