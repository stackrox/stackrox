#!/usr/bin/env bash

set -e

echo "Building with platform linux/${GOARCH}"
if ! docker buildx version &>/dev/null; then
    echo "Error: Docker BuildKit is required to build this image." >&2
    echo "The Dockerfile uses BuildKit features (COPY --link, syntax directive)." >&2
    echo "Please install Docker Buildx or enable BuildKit with DOCKER_BUILDKIT=1." >&2
    exit 1
fi

# SOURCE_DATE_EPOCH is automatically propagated by buildx v0.10+, but we pass it
# explicitly for older versions. The key for reproducible layers is rewrite-timestamp.
if [ -n "${SOURCE_DATE_EPOCH:-}" ]; then
    echo "Using SOURCE_DATE_EPOCH=$SOURCE_DATE_EPOCH for reproducible build"
    # rewrite-timestamp=true applies SOURCE_DATE_EPOCH to file timestamps in layers
    # Without this, only image metadata uses SOURCE_DATE_EPOCH, not layer contents
    # Note: --output type=docker is equivalent to --load but allows extra options
    docker buildx build \
        --platform "linux/${GOARCH}" \
        --build-arg "SOURCE_DATE_EPOCH=${SOURCE_DATE_EPOCH}" \
        --output "type=docker,rewrite-timestamp=true" \
        "$@"
else
    docker buildx build --platform "linux/${GOARCH}" --load "$@"
fi
