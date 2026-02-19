#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/lib.sh
source "$ROOT/scripts/lib.sh"

set -euo pipefail

[[ "$#" == 2 ]] || die "Usage: $0 <image> <csv of architectures>"

image="$1"
IFS=',' read -ra architectures <<< "$2"

[[ -n "$image" ]] || die "No image specified"
[[ "$image" == *:* ]] || die "Must specify a tagged image reference when using this script"

# Push a multi-arch manifest index using docker buildx imagetools create, which
# produces an OCI image index (application/vnd.oci.image.index.v1+json) rather
# than a Docker manifest list (application/vnd.docker.distribution.manifest.list.v2+json).
#
# The Docker manifest list approach was abandoned because quay's
# DockerSchema2ManifestList validator strictly requires all referenced manifests
# to use Docker v2 mediaTypes. When amd64 and arm64 builds land on different
# GitHub Actions runner generations (e.g. ubuntu24/20260201.15 using overlay2
# vs ubuntu24/20260217.30 using the containerd image store), one arch produces a
# Docker v2 manifest and the other an OCI manifest. The resulting mixed manifest
# list is rejected by quay with "manifest invalid" (PROJQUAY-9687).
#
# docker buildx imagetools create produces an OCI index which quay validates
# under a different, more permissive schema that accepts both OCI and Docker v2
# manifest references regardless of their individual mediaTypes.
#
# Each retry re-pulls and re-inspects all arch images from the registry so that
# any stale or inconsistent manifest served by quay on a prior attempt does not
# poison subsequent retries.
pushed=0
for i in {1..5}; do
  echo "Pushing manifest index for ${image}. Attempt ${i}..."

  image_list=()
  for arch in "${architectures[@]}"; do
    arch_image="${image}-${arch}"
    docker pull "${arch_image}"
    echo "=== Arch manifest inspect: ${arch_image} ==="
    docker manifest inspect "${arch_image}" || true
    echo "============================================="
    image_list+=("$arch_image")
  done

  if docker buildx imagetools create --tag "$image" "${image_list[@]}"; then
    echo "=== Manifest index inspect: ${image} ==="
    docker buildx imagetools inspect "$image" || true
    echo "========================================"
    pushed=1
    break
  fi
  sleep 10
done
(( pushed ))
