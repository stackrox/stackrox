#!/usr/bin/env bash

set -e

CONTAINER_ENGINE="${CONTAINER_ENGINE:-docker}"
GOARCH="${GOARCH:-$(go env GOARCH)}"

echo "Building with platform linux/${GOARCH}"

buildx_enabled=$(${CONTAINER_ENGINE} info 2>/dev/null | grep -q buildx && echo true || echo false)

if [[ "${CONTAINER_ENGINE}" == "docker" ]] && [[ "${buildx_enabled}" == "true" ]]; then
    cache_args=()
    # DOCKER_CACHE_FROM: registry image to use as layer cache (prior
    # session proved COPY --link + deterministic binaries make layers
    # byte-stable across runs; --cache-from then hits and the push
    # dedups to zero layers on unchanged content).
    if [[ -n "${DOCKER_CACHE_FROM:-}" ]]; then
        cache_args+=(--cache-from "type=registry,ref=${DOCKER_CACHE_FROM}")
    fi
    docker buildx build --platform "linux/${GOARCH}" --load "${cache_args[@]}" "$@"
else
    ${CONTAINER_ENGINE} build --platform "linux/${GOARCH}" "$@"
fi
