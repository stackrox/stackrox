#!/usr/bin/env bash

set -e

echo "Building with platform linux/${GOARCH}"
if ! docker buildx version &>/dev/null; then
    echo "Error: Docker BuildKit is required to build this image." >&2
    echo "The Dockerfile uses BuildKit features (COPY --link, syntax directive)." >&2
    echo "Please install Docker Buildx or enable BuildKit with DOCKER_BUILDKIT=1." >&2
    exit 1
fi
docker buildx build --platform "linux/${GOARCH}" --load "$@"
