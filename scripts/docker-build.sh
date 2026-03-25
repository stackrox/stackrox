#!/usr/bin/env bash

set -e

# Build for the target platform and force Docker v2 manifest format by using
# --output type=docker explicitly. This is necessary because docker buildx
# build --load on runners with the containerd image store (Docker 29,
# ubuntu24/20260209.23+) produces OCI format manifests by default, which quay
# rejects when they appear alongside Docker v2 manifests in a manifest index
# (PROJQUAY-9687). --output type=docker forces the Docker Image Specification
# format regardless of the builder driver or image store configuration, while
# keeping BuildKit and the full buildx feature set available.
echo "Building with platform linux/${GOARCH}"
docker buildx build --platform "linux/${GOARCH}" --output "type=docker,dest=-" "$@" | docker load
