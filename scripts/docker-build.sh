#!/usr/bin/env bash

set -e

cache_args=()
if [[ -n "${DOCKER_BUILDX_CACHE}" ]]; then
    # GHA buildx cache: reuse Docker layers (base image pulls, package installs)
    # across CI runs. Scope per-arch to avoid cache collisions.
    cache_args+=(
        --cache-from "type=gha,scope=${DOCKER_BUILDX_CACHE}"
        --cache-to "type=gha,mode=max,scope=${DOCKER_BUILDX_CACHE}"
    )
fi

echo "Building with platform linux/${GOARCH}"
if docker info | grep buildx; then
    docker buildx build --platform "linux/${GOARCH}" "${cache_args[@]}" --load "$@"
else
    docker build --platform "linux/${GOARCH}" "$@"
fi
