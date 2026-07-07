#!/usr/bin/env bash

set -e

echo "Building with platform linux/${GOARCH}"
if command -v docker buildx &>/dev/null; then
    if docker info | grep buildx; then
        docker buildx build --platform "linux/${GOARCH}" --load "$@"
    else
        docker build --platform "linux/${GOARCH}" "$@"
    fi
    exit 0
fi
if command -v podman &>/dev/null; then
    podman build --platform "linux/${GOARCH}" "$@"
    exit 0
fi
echo "error: docker and podman are both not available"
exit 1
