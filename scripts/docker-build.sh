#!/usr/bin/env bash

set -e

CONTAINER_RUNTIME="${CONTAINER_RUNTIME:-docker}"

echo "Building with platform linux/${GOARCH}"
if ${CONTAINER_RUNTIME} info | grep buildx; then
    # --load is only needed for docker buildx (podman loads images by default)
    LOAD_FLAG=""
    [[ "${CONTAINER_RUNTIME}" == "docker" ]] && LOAD_FLAG="--load"
    ${CONTAINER_RUNTIME} buildx build --platform "linux/${GOARCH}" ${LOAD_FLAG} "$@"
else
    ${CONTAINER_RUNTIME} build --platform "linux/${GOARCH}" "$@"
fi
