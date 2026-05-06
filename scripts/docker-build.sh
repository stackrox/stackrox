#!/usr/bin/env bash

set -e

echo "Building with platform linux/${GOARCH}"
if ! docker buildx version &>/dev/null; then
    echo "Error: Docker BuildKit is required to build this image." >&2
    echo "The Dockerfile uses BuildKit features (COPY --link, syntax directive)." >&2
    echo "Please install Docker Buildx or enable BuildKit with DOCKER_BUILDKIT=1." >&2
    exit 1
fi

# Pass SOURCE_DATE_EPOCH to BuildKit for reproducible layer timestamps
# BuildKit running in docker-container driver needs explicit build args
if [ -n "${SOURCE_DATE_EPOCH:-}" ]; then
    echo "Using SOURCE_DATE_EPOCH=$SOURCE_DATE_EPOCH for reproducible build"
    docker buildx build --platform "linux/${GOARCH}" --load --build-arg "SOURCE_DATE_EPOCH=${SOURCE_DATE_EPOCH}" "$@"
else
    docker buildx build --platform "linux/${GOARCH}" --load "$@"
fi
