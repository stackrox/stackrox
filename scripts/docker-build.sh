#!/usr/bin/env bash

set -e

CONTAINER_ENGINE="${CONTAINER_ENGINE:-docker}"

echo "Building with platform linux/${GOARCH}"

buildx_enabled=$(${CONTAINER_ENGINE} info 2>/dev/null | grep -q buildx && echo true || echo false)

if [[ "${CONTAINER_ENGINE}" == "docker" ]] && [[ "${buildx_enabled}" == "true" ]]; then
    docker buildx build --platform "linux/${GOARCH}" --load "$@"
else
    ${CONTAINER_ENGINE} build --platform "linux/${GOARCH}" "$@"
fi
