#!/usr/bin/env bash

set -e

echo "Building with platform linux/${GOARCH}"
if docker info | grep buildx; then
    docker buildx build --platform "linux/${GOARCH}" --load "$@"
else
    docker build --platform "linux/${GOARCH}" "$@"
fi
