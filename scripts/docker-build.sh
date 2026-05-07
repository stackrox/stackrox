#!/usr/bin/env bash

set -e

echo "Building with platform linux/${GOARCH}"

# BuildKit cache configuration
CACHE_ARGS=""
if [[ -n "${BUILDKIT_CACHE_FROM:-}" ]]; then
    CACHE_ARGS="--cache-from=${BUILDKIT_CACHE_FROM}"
fi
if [[ -n "${BUILDKIT_CACHE_TO:-}" ]]; then
    CACHE_ARGS="${CACHE_ARGS} --cache-to=${BUILDKIT_CACHE_TO}"
fi

if docker info | grep buildx; then
    docker buildx build --platform "linux/${GOARCH}" --load ${CACHE_ARGS} "$@"
else
    docker build --platform "linux/${GOARCH}" "$@"
fi
