#!/usr/bin/env bash

set -e

echo "Building with platform linux/${GOARCH}"
if ! docker buildx version &>/dev/null; then
    echo "Error: Docker BuildKit is required to build this image." >&2
    echo "The Dockerfile uses BuildKit features (COPY --link, syntax directive)." >&2
    echo "Please install Docker Buildx or enable BuildKit with DOCKER_BUILDKIT=1." >&2
    exit 1
fi

# SOURCE_DATE_EPOCH enables reproducible builds. For reproducible layers, we need
# rewrite-timestamp and direct registry push (BuildKit's gzip is deterministic,
# Docker daemon's is not).
if [ -n "${SOURCE_DATE_EPOCH:-}" ]; then
    echo "Using SOURCE_DATE_EPOCH=$SOURCE_DATE_EPOCH for reproducible build"
    echo "Pushing directly to registry from BuildKit for reproducible layers"

    docker buildx build \
        --platform "linux/${GOARCH}" \
        --build-arg "SOURCE_DATE_EPOCH=${SOURCE_DATE_EPOCH}" \
        --output "type=registry,push=true,rewrite-timestamp=true,compression=gzip" \
        "$@"
else
    # Local development builds without SOURCE_DATE_EPOCH
    docker buildx build --platform "linux/${GOARCH}" --load "$@"
fi
